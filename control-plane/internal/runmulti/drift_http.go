package runmulti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DriftCheckOptions are optional fields for POST /v1/drift/check.
type DriftCheckOptions struct {
	Tags             []string
	SlowPathRequired bool
	IsFollowupCheck  bool
}

// PostDriftCheck calls POST /v1/drift/check with proposal only (backward compatible).
func PostDriftCheck(ctx context.Context, baseURL, proposal string, client *http.Client) (DriftResult, error) {
	return PostDriftCheckEx(ctx, baseURL, proposal, client, nil)
}

// PostDriftCheckEx calls POST /v1/drift/check with optional slow-path / follow-up flags.
func PostDriftCheckEx(ctx context.Context, baseURL, proposal string, client *http.Client, opts *DriftCheckOptions) (DriftResult, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if client == nil {
		client = http.DefaultClient
	}
	req := DriftCheckRequest{Proposal: proposal}
	if opts != nil {
		req.Tags = opts.Tags
		req.SlowPathRequired = opts.SlowPathRequired
		req.IsFollowupCheck = opts.IsFollowupCheck
	}
	body, err := json.Marshal(req)
	if err != nil {
		return DriftResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/drift/check", bytes.NewReader(body))
	if err != nil {
		return DriftResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return DriftResult{}, err
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return DriftResult{}, fmt.Errorf("%s: %s", resp.Status, string(slurp))
	}
	var out DriftResult
	if err := json.Unmarshal(slurp, &out); err != nil {
		return DriftResult{}, fmt.Errorf("decode: %w", err)
	}
	return out, nil
}

// PostDriftCheckSlowPathOptional calls drift with optional tags and slow_path_required; if the server returns
// requires_followup_check, performs a second call with is_followup_check.
func PostDriftCheckSlowPathOptional(ctx context.Context, baseURL, proposal string, client *http.Client, tags []string, slowPathRequired bool) (DriftResult, error) {
	opts := &DriftCheckOptions{Tags: tags}
	if slowPathRequired {
		opts.SlowPathRequired = true
	}
	driftResp, err := PostDriftCheckEx(ctx, baseURL, proposal, client, opts)
	if err != nil {
		return driftResp, err
	}
	if driftResp.RequiresFollowupCheck {
		follow, err := PostDriftCheckEx(ctx, baseURL, proposal, client, &DriftCheckOptions{
			Tags:            tags,
			IsFollowupCheck: true,
		})
		if err != nil {
			return DriftResult{}, err
		}
		return follow, nil
	}
	return driftResp, nil
}
