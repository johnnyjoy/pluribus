package evidence

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestStorage_Save_by_kind_and_digest(t *testing.T) {
	dir := t.TempDir()
	s := &Storage{RootPath: dir}
	content := []byte("hello world")
	contentB64 := base64.StdEncoding.EncodeToString(content)

	path, digest, err := s.Save("testkind", "", contentB64)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if digest == "" || len(digest) < 7 || digest[:7] != "sha256:" {
		t.Errorf("digest = %q", digest)
	}
	if path != filepath.Join(dir, "testkind", digest) {
		t.Errorf("path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("file content = %q", data)
	}
}

func TestStorage_Save_with_provided_digest(t *testing.T) {
	dir := t.TempDir()
	s := &Storage{RootPath: dir}
	contentB64 := base64.StdEncoding.EncodeToString([]byte("x"))
	path, digest, err := s.Save("k", "sha256:abc", contentB64)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if digest != "sha256:abc" {
		t.Errorf("digest = %q", digest)
	}
	if path != filepath.Join(dir, "k", "sha256:abc") {
		t.Errorf("path = %q", path)
	}
}
