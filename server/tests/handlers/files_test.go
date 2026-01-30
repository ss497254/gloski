package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ss497254/gloski/internal/api/handlers"
	"testing"

	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/tests/testutil"
)

func setupFilesHandler(t *testing.T) (*handlers.FilesHandler, string) {
	t.Helper()

	cfg := testutil.TestConfig(t)
	tmpDir := testutil.TestTempDir(t)
	cfg.AllowedPaths = []string{tmpDir}

	fileService := files.NewService(cfg)
	handler := handlers.NewFilesHandler(fileService)

	return handler, tmpDir
}

func TestFilesHandler_List(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files", handler.List)

	t.Run("list directory", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files?path=" + tmpDir,
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Success returns data directly
		var data map[string]interface{}
		testutil.DecodeJSON(t, w.Body, &data)

		entries, ok := data["entries"].([]interface{})
		if !ok {
			t.Fatal("entries is not an array")
		}

		if len(entries) == 0 {
			t.Error("expected some entries in test directory")
		}
	})

	t.Run("list with default path (home)", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files",
		})

		// Should use ~ as default which should succeed if home dir is accessible
		if w.Code != http.StatusOK && w.Code != http.StatusForbidden {
			t.Errorf("unexpected status code: %d", w.Code)
		}
	})

	t.Run("list non-existent directory", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files?path=" + filepath.Join(tmpDir, "nonexistent"),
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Read(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/read", handler.Read)

	testFile := filepath.Join(tmpDir, "test.txt")

	t.Run("read existing file", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files/read?path=" + testFile,
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Success returns data directly
		var data map[string]interface{}
		testutil.DecodeJSON(t, w.Body, &data)

		content, ok := data["content"].(string)
		if !ok {
			t.Fatal("content is not a string")
		}

		if content != "test content" {
			t.Errorf("content = %s, want 'test content'", content)
		}
	})

	t.Run("read without path parameter", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files/read",
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})

	t.Run("read non-existent file", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files/read?path=" + filepath.Join(tmpDir, "nonexistent.txt"),
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Write(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/write", handler.Write)

	t.Run("write new file", func(t *testing.T) {
		testPath := filepath.Join(tmpDir, "new_file.txt")
		defer os.Remove(testPath)

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/write",
			Body: map[string]string{
				"path":    testPath,
				"content": "new content",
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify file was created
		content, err := os.ReadFile(testPath)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}

		if string(content) != "new content" {
			t.Errorf("file content = %s, want 'new content'", string(content))
		}
	})

	t.Run("write with missing path", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/write",
			Body: map[string]string{
				"content": "content",
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})

	t.Run("write with invalid JSON", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/write",
			Body:   "invalid json",
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Mkdir(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/mkdir", handler.Mkdir)

	t.Run("create directory", func(t *testing.T) {
		newDir := filepath.Join(tmpDir, "new_directory")
		defer os.RemoveAll(newDir)

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/mkdir",
			Body: map[string]string{
				"path": newDir,
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify directory was created
		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("created path is not a directory")
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		nestedDir := filepath.Join(tmpDir, "parent", "child", "grandchild")
		defer os.RemoveAll(filepath.Join(tmpDir, "parent"))

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/mkdir",
			Body: map[string]string{
				"path": nestedDir,
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify nested directories were created
		if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
			t.Error("nested directories were not created")
		}
	})
}

func TestFilesHandler_Delete(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files", handler.Delete)

	t.Run("delete file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "to_delete.txt")
		os.WriteFile(testFile, []byte("delete me"), 0644)

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodDelete,
			Path:   "/api/files?path=" + testFile,
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify file was deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("file was not deleted")
		}
	})

	t.Run("delete without path", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodDelete,
			Path:   "/api/files",
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Rename(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/rename", handler.Rename)

	t.Run("rename file", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "old_name.txt")
		newPath := filepath.Join(tmpDir, "new_name.txt")
		os.WriteFile(oldPath, []byte("content"), 0644)
		defer os.Remove(newPath) // cleanup

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/rename",
			Body: map[string]string{
				"old_path": oldPath,
				"new_path": newPath,
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify old file doesn't exist
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Error("old file still exists")
		}

		// Verify new file exists
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Error("new file was not created")
		}
	})

	t.Run("rename with missing parameters", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/rename",
			Body: map[string]string{
				"old_path": "/some/path",
			},
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Upload(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/upload", handler.Upload)

	t.Run("upload file", func(t *testing.T) {
		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		writer.WriteField("path", tmpDir)

		fileWriter, err := writer.CreateFormFile("file", "uploaded.txt")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}

		fileWriter.Write([]byte("uploaded content"))
		writer.Close()

		req := testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/files/upload",
			Body:   body.Bytes(),
			Headers: map[string]string{
				"Content-Type": writer.FormDataContentType(),
			},
		}

		w := testutil.MakeRequest(t, mux, req)

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Verify file was uploaded
		uploadedPath := filepath.Join(tmpDir, "uploaded.txt")
		defer os.Remove(uploadedPath)

		content, err := os.ReadFile(uploadedPath)
		if err != nil {
			t.Fatalf("uploaded file not found: %v", err)
		}

		if string(content) != "uploaded content" {
			t.Errorf("uploaded content = %s, want 'uploaded content'", string(content))
		}
	})
}

func TestFilesHandler_Search(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", handler.Search)

	t.Run("search by filename", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/search?path=" + tmpDir + "&q=test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Success returns data directly
		var data map[string]interface{}
		testutil.DecodeJSON(t, w.Body, &data)

		results, ok := data["results"].([]interface{})
		if !ok {
			t.Fatal("results is not an array")
		}

		// Should find at least test.txt
		if len(results) == 0 {
			t.Error("expected to find some results")
		}
	})

	t.Run("search without query", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/search?path=" + tmpDir,
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestFilesHandler_Download(t *testing.T) {
	handler, tmpDir := setupFilesHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files/download", handler.Download)

	t.Run("download file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.txt")

		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files/download?path=" + testFile,
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Check headers
		contentDisposition := w.Header().Get("Content-Disposition")
		if contentDisposition == "" {
			t.Error("Content-Disposition header not set")
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/octet-stream" {
			t.Errorf("Content-Type = %s, want application/octet-stream", contentType)
		}
	})

	t.Run("download directory fails", func(t *testing.T) {
		w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/api/files/download?path=" + tmpDir,
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)
	})
}
