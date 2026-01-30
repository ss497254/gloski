package handlers_test

import (

	"github.com/ss497254/gloski/internal/api/handlers"
	"net/http"
	"testing"

	"github.com/ss497254/gloski/tests/testutil"
)

func TestHealthHandler(t *testing.T) {
	handler := handlers.NewHealthHandler()

	tests := []struct {
		name       string
		path       string
		method     string
		wantStatus int
		wantBody   map[string]string
	}{
		{
			name:       "health check",
			path:       "/health",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ok"},
		},
		{
			name:       "ready check",
			path:       "/ready",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "ready"},
		},
		{
			name:       "live check",
			path:       "/live",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   map[string]string{"status": "live"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/health", handler.Check)
			mux.HandleFunc("/ready", handler.Ready)
			mux.HandleFunc("/live", handler.Live)

			w := testutil.MakeRequest(t, mux, testutil.HTTPRequest{
				Method: tt.method,
				Path:   tt.path,
			})

			testutil.AssertStatus(t, w.Code, tt.wantStatus)

			// Success returns data directly, not wrapped
			var data map[string]interface{}
			testutil.DecodeJSON(t, w.Body, &data)

			for key, want := range tt.wantBody {
				got, ok := data[key].(string)
				if !ok {
					t.Errorf("key %s not found or not a string", key)
					continue
				}
				if got != want {
					t.Errorf("%s = %s, want %s", key, got, want)
				}
			}
		})
	}
}

func TestResponseHelpers(t *testing.T) {
	t.Run("Success response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.Success(w, map[string]string{"test": "data"})
		})

		w := testutil.MakeRequest(t, handler, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusOK)

		// Success returns data directly, not wrapped in Response struct
		var data map[string]interface{}
		testutil.DecodeJSON(t, w.Body, &data)

		if data["test"] != "data" {
			t.Errorf("data.test = %v, want 'data'", data["test"])
		}
	})

	t.Run("BadRequest response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.BadRequest(w, "invalid input")
		})

		w := testutil.MakeRequest(t, handler, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusBadRequest)

		var response map[string]interface{}
		testutil.DecodeJSON(t, w.Body, &response)

		if response["success"] != false {
			t.Error("success field should be false")
		}

		if response["error"] != "invalid input" {
			t.Errorf("error = %v, want 'invalid input'", response["error"])
		}
	})

	t.Run("NotFound response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.NotFound(w, "resource not found")
		})

		w := testutil.MakeRequest(t, handler, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusNotFound)
	})

	t.Run("Forbidden response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.Forbidden(w, "access denied")
		})

		w := testutil.MakeRequest(t, handler, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusForbidden)
	})

	t.Run("InternalError response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlers.InternalError(w, "something went wrong", "detailed error")
		})

		w := testutil.MakeRequest(t, handler, testutil.HTTPRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})

		testutil.AssertStatus(t, w.Code, http.StatusInternalServerError)
	})
}
