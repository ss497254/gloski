package master

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/devices"
	"github.com/ss497254/gloski/internal/logger"
)

// handleDeviceProxy forwards /api/devices/{id}/... to the agent's private API.
// The agent never receives the dashboard's credentials; the master injects the
// device token it stored during enrollment.
func (s *Server) handleDeviceProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dev, err := s.devices.Get(id)
	if err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			handlers.NotFound(w, "device not found")
			return
		}
		handlers.InternalError(w, "failed to load device", err.Error())
		return
	}
	if dev.Status == devices.StatusOffline {
		handlers.Error(w, http.StatusBadGateway, "device is offline")
		return
	}

	target, err := url.Parse(dev.URL)
	if err != nil || target.Host == "" {
		handlers.Error(w, http.StatusBadGateway, "device has an invalid agent URL")
		return
	}

	prefix := "/api/devices/" + id
	remainder := strings.TrimPrefix(r.URL.Path, prefix)
	if remainder == "" {
		remainder = "/"
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path = strings.TrimRight(target.Path, "/") + remainder
			pr.Out.URL.RawPath = ""
			query := r.URL.Query()
			// Never forward dashboard credentials to an agent.
			query.Del("api_key")
			query.Del("token")
			pr.Out.URL.RawQuery = query.Encode()
			pr.Out.Host = target.Host
			pr.Out.Header.Set("X-API-Key", dev.Token)
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("Cookie")
		},
		Transport:     s.transport,
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("Device proxy error for %s: %v", dev.ID, err)
			handlers.Error(w, http.StatusBadGateway, "failed to reach device agent")
		},
	}
	proxy.ServeHTTP(w, r)
}
