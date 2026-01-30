// Package testutil provides common test utilities and helpers
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ss497254/gloski/internal/config"
)

// TestConfig returns a config suitable for testing
func TestConfig(t *testing.T) *config.Config {
	t.Helper()

	homeDir, _ := os.UserHomeDir()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Host:            "127.0.0.1",
		Port:            8080,
		APIKey:          "test-api-key",
		DataDir:         tmpDir,
		AllowedOrigins:  []string{"*"},
		AllowedPaths:    []string{homeDir, tmpDir},
		Shell:           "/bin/bash",
		LogLevel:        "error",
		DetailedErrors:  true,
		MaxJSONBodySize: 1024 * 1024,
		ShutdownTimeout: 5,
	}

	return cfg
}

// TestTempDir creates a temporary directory with test files
func TestTempDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create some test files and directories
	testFiles := map[string]string{
		"test.txt":           "test content",
		"dir1/file1.txt":     "file1 content",
		"dir1/file2.txt":     "file2 content",
		"dir2/nested/test.md": "# Test\nmarkdown content",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	return tmpDir
}

// HTTPRequest is a helper for making test HTTP requests
type HTTPRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// MakeRequest creates and executes an HTTP test request
func MakeRequest(t *testing.T, handler http.Handler, req HTTPRequest) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			body = bytes.NewBufferString(v)
		case []byte:
			body = bytes.NewBuffer(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}
			body = bytes.NewBuffer(data)
		}
	}

	r := httptest.NewRequest(req.Method, req.Path, body)

	// Set default headers
	if req.Body != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		r.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	return w
}

// DecodeJSON decodes JSON response body into target
func DecodeJSON(t *testing.T, body io.Reader, target interface{}) {
	t.Helper()

	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
}

// AssertStatus checks if response has expected status code
func AssertStatus(t *testing.T, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("status code = %d, want %d", got, want)
	}
}

// AssertContains checks if string contains substring
func AssertContains(t *testing.T, got, want string) {
	t.Helper()

	if got == "" {
		t.Error("got empty string")
		return
	}

	if want != "" && !contains(got, want) {
		t.Errorf("string does not contain expected substring\ngot: %s\nwant substring: %s", got, want)
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNoError checks that error is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError checks that error is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
