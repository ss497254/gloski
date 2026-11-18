package httpapi

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ss497254/gloski/internal/logger"
)

// OriginAllowed reports whether a request may change state or open a
// WebSocket. Browsers attach an Origin header to every WebSocket handshake and
// every cross-site write, so a mismatch means another website is driving the
// user's browser. Requests without an Origin (curl, scripts) are not browser
// requests and pass through.
//
// The Origin must match the Host header, the X-Forwarded-Host set by a reverse
// proxy, or an entry in allowed ("*" disables the check). A page cannot forge
// X-Forwarded-Host: WebSockets cannot set headers, and a fetch that sets one
// needs a CORS preflight this server never grants.
func OriginAllowed(r *http.Request, allowed []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := normalizeHost(parsed.Host, parsed.Scheme)

	if host == normalizeHost(r.Host, parsed.Scheme) {
		return true
	}
	for _, forwarded := range strings.Split(r.Header.Get("X-Forwarded-Host"), ",") {
		if forwarded = strings.TrimSpace(forwarded); forwarded != "" && host == normalizeHost(forwarded, parsed.Scheme) {
			return true
		}
	}
	for _, entry := range allowed {
		entry = strings.TrimRight(strings.TrimSpace(entry), "/")
		if entry == "" {
			continue
		}
		if entry == "*" || strings.EqualFold(entry, origin) || normalizeHost(entry, parsed.Scheme) == host {
			return true
		}
	}
	return false
}

// SameOrigin rejects cross-site writes and WebSocket handshakes. Plain GETs
// are left alone: the browser's same-origin policy already stops another site
// from reading their responses.
func SameOrigin(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (isWrite(r.Method) || isWebSocket(r)) && !OriginAllowed(r, allowed) {
				logger.Warn("Rejected cross-origin %s %s from origin %q (host %q); if this is your reverse proxy, forward the Host header or set GLOSKI_ALLOWED_ORIGINS",
					r.Method, r.URL.Path, r.Header.Get("Origin"), r.Host)
				Forbidden(w, "cross-origin request rejected")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isWrite(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// normalizeHost lowercases a host and drops the scheme's default port so
// "example.com" and "example.com:443" compare equal.
func normalizeHost(host, scheme string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	switch scheme {
	case "https", "wss":
		host = strings.TrimSuffix(host, ":443")
	case "http", "ws":
		host = strings.TrimSuffix(host, ":80")
	}
	return host
}
