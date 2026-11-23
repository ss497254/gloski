package controltower

import (
	"net/http"

	"github.com/ss497254/gloski/internal/httpapi"
)

func (t *Tower) routes() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints. Agents enroll over the gRPC stream; the dashboard is
	// protected by tailnet access and, optionally, by an API key / JWT.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpapi.Success(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/tower/info", t.handleTowerInfo)
	mux.Handle("GET /api/dashboard", t.userAuth(http.HandlerFunc(t.handleDashboard)))
	mux.Handle("GET /api/fleet/ws", t.userAuth(http.HandlerFunc(t.handleFleetWS)))

	// Fleet API. If the operator configured an API key or JWT, require it.
	mux.Handle("GET /api/devices", t.userAuth(http.HandlerFunc(t.handleListDevices)))
	mux.Handle("POST /api/devices/deploy", t.userAuth(http.HandlerFunc(t.handleDeployDevice)))
	mux.Handle("GET /api/devices/{id}", t.userAuth(http.HandlerFunc(t.handleGetDevice)))
	mux.Handle("DELETE /api/devices/{id}", t.userAuth(http.HandlerFunc(t.handleDeleteDevice)))
	mux.Handle("POST /api/devices/{id}/start", t.userAuth(http.HandlerFunc(t.handleStartDevice)))
	mux.Handle("POST /api/devices/{id}/stop", t.userAuth(http.HandlerFunc(t.handleStopDevice)))
	mux.Handle("POST /api/devices/{id}/diagnose", t.userAuth(http.HandlerFunc(t.handleDiagnoseDevice)))
	mux.Handle("POST /api/devices/{id}/trust-host", t.userAuth(http.HandlerFunc(t.handleTrustHost)))

	// Browser-facing device API. The master translates each call into gRPC
	// commands on the device's agent stream.
	mux.Handle("GET /api/devices/{id}/api/system/stats", t.userAuth(http.HandlerFunc(t.handleDeviceStats)))
	mux.Handle("GET /api/devices/{id}/api/system/stats/ws", t.userAuth(http.HandlerFunc(t.handleDeviceStatsWS)))
	mux.Handle("GET /api/devices/{id}/api/system/processes", t.userAuth(http.HandlerFunc(t.handleDeviceProcesses)))
	mux.Handle("GET /api/devices/{id}/api/jobs", t.userAuth(http.HandlerFunc(t.handleListJobs)))
	mux.Handle("POST /api/devices/{id}/api/jobs", t.userAuth(http.HandlerFunc(t.handleRunJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}", t.userAuth(http.HandlerFunc(t.handleGetJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}/logs", t.userAuth(http.HandlerFunc(t.handleJobLogs)))
	mux.Handle("POST /api/devices/{id}/api/jobs/{jobId}/stop", t.userAuth(http.HandlerFunc(t.handleCancelJob)))
	mux.Handle("DELETE /api/devices/{id}/api/jobs/{jobId}", t.userAuth(http.HandlerFunc(t.handleDeleteJob)))
	mux.Handle("GET /api/devices/{id}/api/files", t.userAuth(http.HandlerFunc(t.handleListFiles)))
	mux.Handle("DELETE /api/devices/{id}/api/files", t.userAuth(http.HandlerFunc(t.handleDeleteFile)))
	mux.Handle("GET /api/devices/{id}/api/files/read", t.userAuth(http.HandlerFunc(t.handleReadFile)))
	mux.Handle("GET /api/devices/{id}/api/files/download", t.userAuth(http.HandlerFunc(t.handleDownloadFile)))
	mux.Handle("POST /api/devices/{id}/api/files/write", t.userAuth(http.HandlerFunc(t.handleWriteFile)))
	mux.Handle("POST /api/devices/{id}/api/files/mkdir", t.userAuth(http.HandlerFunc(t.handleMkdir)))
	mux.Handle("POST /api/devices/{id}/api/files/rename", t.userAuth(http.HandlerFunc(t.handleRename)))
	mux.Handle("POST /api/devices/{id}/api/files/upload", t.userAuth(http.HandlerFunc(t.handleUploadFile)))
	mux.Handle("GET /api/devices/{id}/api/terminal", t.userAuth(http.HandlerFunc(t.handleTerminalWS)))

	// SSH deployment and tailnet discovery.
	mux.Handle("GET /api/deployments", t.userAuth(http.HandlerFunc(t.handleListDeployments)))
	mux.Handle("GET /api/deployments/{id}", t.userAuth(http.HandlerFunc(t.handleGetDeployment)))
	mux.Handle("GET /api/tailnet/peers", t.userAuth(http.HandlerFunc(t.handleTailnetPeers)))

	// Dashboard static assets and SPA fallback.
	mux.HandleFunc("/", t.serveSPA)

	return httpapi.Logging(securityHeaders(httpapi.SameOrigin(t.cfg.AllowedOrigins)(mux)))
}

func (t *Tower) userAuth(next http.Handler) http.Handler {
	if t.cfg.APIKey == "" && t.cfg.JWTPublicKey == "" {
		return next
	}
	return httpapi.Auth(t.auth)(next)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
