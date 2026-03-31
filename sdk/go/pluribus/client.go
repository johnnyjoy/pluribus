package pluribus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 15 * time.Second

// Client is a thin HTTP client for the agent memory loop.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a client. apiKey may be empty when the server has no API key configured.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// RecallContextOpts optional fields for RecallContext (behavior: recall_context).
type RecallContextOpts struct {
	Tags          []string
	CorrelationID string
}

// RecallContext compiles governing memory for the current situation (GET memory into the agent).
func (c *Client) RecallContext(ctx context.Context, query string, opts RecallContextOpts) (*RecallBundle, error) {
	body := map[string]any{"retrieval_query": query}
	if len(opts.Tags) > 0 {
		body["tags"] = opts.Tags
	}
	if opts.CorrelationID != "" {
		body["correlation_id"] = opts.CorrelationID
	}
	var out RecallBundle
	if err := c.doJSON(ctx, http.MethodPost, "/v1/recall/compile", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RecordExperienceOpts optional fields for RecordExperience (behavior: record_experience).
type RecordExperienceOpts struct {
	Tags          []string
	Entities      []string
	CorrelationID string
}

// RecordExperience logs an advisory episode after meaningful work (not canonical memory until promotion).
func (c *Client) RecordExperience(ctx context.Context, summary string, opts RecordExperienceOpts) (*AdvisoryEpisode, error) {
	body := map[string]any{
		"summary": summary,
		"source":  "mcp",
	}
	if len(opts.Tags) > 0 {
		body["tags"] = opts.Tags
	}
	if len(opts.Entities) > 0 {
		body["entities"] = opts.Entities
	}
	if opts.CorrelationID != "" {
		body["correlation_id"] = opts.CorrelationID
	}
	var out AdvisoryEpisode
	if err := c.doJSON(ctx, http.MethodPost, "/v1/advisory-episodes", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPendingCandidates lists curation candidates awaiting review.
func (c *Client) ListPendingCandidates(ctx context.Context) ([]CandidateEvent, error) {
	var out []CandidateEvent
	if err := c.doJSON(ctx, http.MethodGet, "/v1/curation/pending", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReviewCandidate fetches review assistance for one candidate.
func (c *Client) ReviewCandidate(ctx context.Context, candidateID string) (*CandidateReview, error) {
	if strings.TrimSpace(candidateID) == "" {
		return nil, fmt.Errorf("candidateID is required")
	}
	var out CandidateReview
	path := fmt.Sprintf("/v1/curation/candidates/%s/review", candidateID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PromoteCandidate materializes a candidate into durable memory when policy allows.
func (c *Client) PromoteCandidate(ctx context.Context, candidateID string) (*MaterializeOutcome, error) {
	if strings.TrimSpace(candidateID) == "" {
		return nil, fmt.Errorf("candidateID is required")
	}
	var out MaterializeOutcome
	path := fmt.Sprintf("/v1/curation/candidates/%s/materialize", candidateID)
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	url := c.baseURL + path

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Method:          method,
			Path:            path,
			StatusCode:      resp.StatusCode,
			ResponseSnippet: trimSnippet(string(data), 400),
		}
	}

	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	return nil
}

func trimSnippet(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
