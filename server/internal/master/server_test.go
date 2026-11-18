package master

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexTracksRebuiltBundle(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.html")

	first := `<html><script src="/assets/old.js"></script></html>`
	if err := os.WriteFile(index, []byte(first), 0600); err != nil {
		t.Fatal(err)
	}
	server := &Server{webDir: dir, indexHTML: []byte(first)}

	request := httptest.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()
	server.serveSPA(response, request)
	if !strings.Contains(response.Body.String(), "old.js") {
		t.Fatalf("first response = %q", response.Body.String())
	}

	second := `<html><script src="/assets/new.js"></script></html>`
	if err := os.WriteFile(index, []byte(second), 0600); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	server.serveSPA(response, request)
	if !strings.Contains(response.Body.String(), "new.js") || strings.Contains(response.Body.String(), "old.js") {
		t.Fatalf("response after rebuild = %q", response.Body.String())
	}
}
