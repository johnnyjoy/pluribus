package pluribus

import "fmt"

// APIError wraps non-2xx HTTP responses from Pluribus.
type APIError struct {
	Method          string
	Path            string
	StatusCode      int
	ResponseSnippet string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"pluribus api error: %s %s returned %d: %s",
		e.Method,
		e.Path,
		e.StatusCode,
		e.ResponseSnippet,
	)
}
