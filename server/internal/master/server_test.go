package master

import (
	"net/http"
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
	server := &Tower{webDir: dir, indexHTML: []byte(first)}

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

func TestUnknownSPARouteReturns404WithIndex(t *testing.T) {
	dir := t.TempDir()
	index := `<html><body>spa</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0600); err != nil {
		t.Fatal(err)
	}
	server := &Tower{webDir: dir, indexHTML: []byte(index)}

	cases := map[string]int{
		"/":                         http.StatusOK,
		"/devices":                  http.StatusOK,
		"/devices/abc/files":        http.StatusOK,
		"/settings":                 http.StatusOK,
		"/fleet/abc":                http.StatusOK,
		"/servers/abc/anything":     http.StatusOK,
		"/devices/abc/bogus":        http.StatusNotFound,
		"/fleet/abc/bogus":          http.StatusNotFound,
		"/settings/extra":           http.StatusNotFound,
		"/this-page-does-not-exist": http.StatusNotFound,
	}
	for path, want := range cases {
		response := httptest.NewRecorder()
		server.serveSPA(response, httptest.NewRequest("GET", path, nil))
		if response.Code != want {
			t.Errorf("%s status = %d, want %d", path, response.Code, want)
		}
		if !strings.Contains(response.Body.String(), "spa") {
			t.Errorf("%s did not render the SPA body", path)
		}
	}
}
