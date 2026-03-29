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

// PostPreflight calls POST /v1/recall/preflight on the control plane.
func PostPreflight(ctx context.Context, baseURL string, req PreflightRequestMirror, client *http.Client) (PreflightResultMirror, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(req)
	if err != nil {
		return PreflightResultMirror{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/recall/preflight", bytes.NewReader(body))
	if err != nil {
		return PreflightResultMirror{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return PreflightResultMirror{}, err
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return PreflightResultMirror{}, fmt.Errorf("%s: %s", resp.Status, string(slurp))
	}
	var out PreflightResultMirror
	if err := json.Unmarshal(slurp, &out); err != nil {
		return PreflightResultMirror{}, fmt.Errorf("decode: %w", err)
	}
	return out, nil
}
