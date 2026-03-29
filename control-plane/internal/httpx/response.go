package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// DecodeJSON reads the request body and decodes it into v. Returns an error if body is invalid.
func DecodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// WriteJSON writes v as JSON with 200 and Content-Type application/json.
func WriteJSON(w http.ResponseWriter, v interface{}) {
	WriteJSONStatus(w, http.StatusOK, v)
}

// WriteJSONStatus writes v as JSON with the given status and Content-Type application/json.
func WriteJSONStatus(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error body and status code.
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
