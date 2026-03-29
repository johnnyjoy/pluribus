package evidence

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Storage writes evidence blobs to the filesystem by kind and digest.
type Storage struct {
	RootPath string
}

// Save decodes base64 content, computes digest if not provided, writes to RootPath/kind/digest, returns path and digest.
func (s *Storage) Save(kind, digest, contentBase64 string) (path, digestOut string, err error) {
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return "", "", fmt.Errorf("evidence: invalid base64 content: %w", err)
	}
	if digest == "" {
		h := sha256.Sum256(content)
		digest = fmt.Sprintf("sha256:%x", h[:])
	}
	// Sanitize kind and digest for filesystem (no path separators)
	kindClean := strings.ReplaceAll(strings.TrimSpace(kind), "/", "_")
	if kindClean == "" {
		kindClean = "misc"
	}
	digestClean := strings.ReplaceAll(digest, "/", "_")
	dir := filepath.Join(s.RootPath, kindClean)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	path = filepath.Join(dir, digestClean)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", "", err
	}
	return path, digest, nil
}
