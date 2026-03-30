package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunProofHarnessREST runs all proof-*.json scenarios against baseURL (no path suffix; e.g. http://127.0.0.1:8123).
// Every step uses HTTP only — no in-process service shortcuts.
func RunProofHarnessREST(ctx context.Context, baseURL string, hc *http.Client) (*ProofHarnessReport, error) {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	scenarios, err := LoadProofScenarios()
	if err != nil {
		return nil, err
	}
	runID := uuid.NewString()
	rep := &ProofHarnessReport{RunID: runID, AllPassed: true}

	for _, sc := range scenarios {
		sr := ProofScenarioReport{
			ScenarioID:  sc.ID,
			Suite:       sc.Suite,
			Description: sc.Description,
			AllPassed:   true,
		}
		pctx := &ProofRunContext{
			RunID:  runID,
			Vars:   make(map[string]string),
			Stored: make(map[string][]byte),
		}

		for _, step := range sc.Steps {
			stepRes := runProofStep(ctx, hc, base, pctx, sc.ID, sc.Suite, step)
			sr.Steps = append(sr.Steps, stepRes)
			if !stepRes.Pass {
				sr.AllPassed = false
				rep.AllPassed = false
			}
		}

		if len(sc.AfterInvariants) > 0 {
			inv := RunAfterInvariants(pctx, sc.AfterInvariants)
			sr.Invariants = inv
			for _, ir := range inv {
				if !ir.Pass {
					sr.AllPassed = false
					rep.AllPassed = false
				}
			}
		}

		lp := "[PROOF]"
		if strings.EqualFold(strings.TrimSpace(sc.Suite), "episodic") {
			lp = "[EPISODIC PROOF]"
		}
		log.Printf("%s scenario=%s suite=%s aggregate=%s", lp, sc.ID, sc.Suite, passFailStr(sr.AllPassed))
		rep.Scenarios = append(rep.Scenarios, sr)
	}

	return rep, nil
}

func passFailStr(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

func runProofStep(ctx context.Context, hc *http.Client, base string, pctx *ProofRunContext, scenarioID, suite string, step ProofStep) ProofStepResult {
	res := ProofStepResult{ScenarioID: scenarioID, StepID: step.ID, Path: step.Path, Pass: true}
	logPrefix := "[PROOF]"
	if strings.EqualFold(strings.TrimSpace(suite), "episodic") {
		logPrefix = "[EPISODIC PROOF]"
	}
	method := strings.ToUpper(strings.TrimSpace(step.Method))
	if method == "" {
		method = http.MethodGet
	}
	path := substituteProofVars(step.Path, pctx)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	body := substituteProofBody(step.Body, pctx)
	req, err := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(body))
	if err != nil {
		res.Pass = false
		res.Detail = err.Error()
		res.Status = 0
		log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
		return res
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := hc.Do(req)
	if err != nil {
		res.Pass = false
		res.Detail = err.Error()
		log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
		return res
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	res.Status = resp.StatusCode

	expect := step.ExpectStatus
	if expect == 0 {
		expect = http.StatusOK
	}
	if resp.StatusCode != expect {
		res.Pass = false
		res.Detail = fmt.Sprintf("want status %d got %d body=%s", expect, resp.StatusCode, truncateBody(rb))
		log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
		return res
	}

	for _, a := range step.Asserts {
		if ok, msg := applyProofAssert(a, rb); !ok {
			res.Pass = false
			res.Detail = msg
			log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, msg)
			return res
		}
	}

	if len(step.CaptureFields) > 0 {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(rb, &root); err == nil {
			for varName, field := range step.CaptureFields {
				raw, ok := root[field]
				if !ok {
					res.Pass = false
					res.Detail = fmt.Sprintf("capture missing field %q", field)
					log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
					return res
				}
				var s string
				_ = json.Unmarshal(raw, &s)
				if s == "" {
					s = strings.Trim(string(raw), `"`)
				}
				pctx.Vars[varName] = s
			}
		} else if len(step.CaptureFields) > 0 {
			res.Pass = false
			res.Detail = "capture_fields set but response is not json object"
			log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
			return res
		}
	}

	for varName, jpath := range step.CaptureJSONPath {
		val, err := captureJSONPath(rb, jpath)
		if err != nil {
			res.Pass = false
			res.Detail = fmt.Sprintf("capture_json_path %q: %v", varName, err)
			log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
			return res
		}
		if strings.TrimSpace(val) == "" {
			res.Pass = false
			res.Detail = fmt.Sprintf("capture_json_path %q: empty value at %q", varName, jpath)
			log.Printf("%s scenario=%s phase=%s path=%s status=fail details=%s", logPrefix, scenarioID, step.ID, path, res.Detail)
			return res
		}
		pctx.Vars[varName] = val
	}

	if step.StoreAs != "" {
		cp := make([]byte, len(rb))
		copy(cp, rb)
		pctx.Stored[step.StoreAs] = cp
	}

	log.Printf("%s scenario=%s phase=%s path=%s http=%d status=pass", logPrefix, scenarioID, step.ID, path, resp.StatusCode)
	return res
}

// captureJSONPath reads a string or number from JSON using dot-separated path; array segments use numeric indices (e.g. candidates.0.candidate_id).
func captureJSONPath(body []byte, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	parts := strings.Split(path, ".")
	cur := root
	for _, p := range parts {
		if idx, err := strconv.Atoi(p); err == nil {
			arr, ok := cur.([]interface{})
			if !ok || idx < 0 || idx >= len(arr) {
				return "", fmt.Errorf("invalid array index %q in path %q", p, path)
			}
			cur = arr[idx]
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("expected object at segment %q in path %q", p, path)
		}
		v, ok := m[p]
		if !ok {
			return "", fmt.Errorf("missing key %q in path %q", p, path)
		}
		cur = v
	}
	switch v := cur.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "", fmt.Errorf("null at end of path %q", path)
	default:
		return "", fmt.Errorf("unsupported JSON type %T at end of path %q", cur, path)
	}
}

func substituteProofVars(s string, pctx *ProofRunContext) string {
	s = strings.ReplaceAll(s, "{{RUN_ID}}", pctx.RunID)
	for k, v := range pctx.Vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

func substituteProofBody(raw json.RawMessage, pctx *ProofRunContext) []byte {
	if len(raw) == 0 {
		return nil
	}
	return []byte(substituteProofVars(string(raw), pctx))
}

func applyProofAssert(a ProofAssert, body []byte) (bool, string) {
	txt := string(body)
	switch a.Kind {
	case "body_contains":
		if !strings.Contains(txt, a.Value) {
			return false, fmt.Sprintf("body_contains missing %q", a.Value)
		}
	case "body_not_contains":
		if strings.Contains(txt, a.Value) {
			return false, fmt.Sprintf("body_not_contains forbidden %q found", a.Value)
		}
	case "substring_order":
		i1 := strings.Index(txt, a.Before)
		i2 := strings.Index(txt, a.After)
		if i1 < 0 || i2 < 0 {
			return false, fmt.Sprintf("substring_order missing markers before=%q after=%q", a.Before, a.After)
		}
		if i1 >= i2 {
			return false, fmt.Sprintf("substring_order want %q before %q", a.Before, a.After)
		}
	case "body_contains_any":
		if len(a.Values) == 0 {
			return false, "body_contains_any: no values"
		}
		var hit bool
		for _, v := range a.Values {
			if strings.Contains(txt, v) {
				hit = true
				break
			}
		}
		if !hit {
			return false, fmt.Sprintf("body_contains_any: none of %v matched", a.Values)
		}
	default:
		return false, "unknown assert kind " + a.Kind
	}
	return true, ""
}

func truncateBody(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// RunProofHarnessRESTDeterminism runs the harness twice and compares stable summaries (scenario pass bits + aggregate step statuses).
func RunProofHarnessRESTDeterminism(ctx context.Context, baseURL string, hc *http.Client) (*ProofHarnessReport, error) {
	r1, err := RunProofHarnessREST(ctx, baseURL, hc)
	if err != nil {
		return r1, err
	}
	r2, err := RunProofHarnessREST(ctx, baseURL, hc)
	if err != nil {
		return r1, err
	}
	s1 := proofRunSignature(r1)
	s2 := proofRunSignature(r2)
	r1.DeterminismPass = s1 == s2 && r1.AllPassed && r2.AllPassed
	if s1 != s2 {
		r1.DeterminismNote = fmt.Sprintf("signature mismatch:\n%s\nvs\n%s", s1, s2)
	} else {
		r1.DeterminismNote = "identical pass/fail signature across two runs"
	}
	r1.AllPassed = r1.AllPassed && r1.DeterminismPass
	return r1, nil
}

func proofRunSignature(r *ProofHarnessReport) string {
	var sb strings.Builder
	for _, sc := range r.Scenarios {
		sb.WriteString(sc.ScenarioID)
		sb.WriteByte(':')
		sb.WriteString(passFailStr(sc.AllPassed))
		sb.WriteByte(';')
		for _, st := range sc.Steps {
			sb.WriteString(st.StepID)
			sb.WriteByte(':')
			sb.WriteString(passFailStr(st.Pass))
			sb.WriteByte(':')
			fmt.Fprintf(&sb, "%d", st.Status)
			sb.WriteByte(';')
		}
		for _, iv := range sc.Invariants {
			sb.WriteString(iv.Name)
			sb.WriteByte(':')
			sb.WriteString(passFailStr(iv.Pass))
			sb.WriteByte(';')
		}
	}
	return sb.String()
}
