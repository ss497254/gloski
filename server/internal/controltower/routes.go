package controltower

import (
	"net/http"

	"github.com/ss497254/gloski/internal/httpapi"
)

func (t *Tower) routes() http.Handler {
	mux := http.NewServeMux()

	device := t.deviceAPI
	deploy := t.deployAPI
	fleetH := t.fleetAPI
	dash := t.dashboardAPI

	// Public endpoints. Agents enroll over the gRPC stream; the dashboard is
	// protected by tailnet access and, optionally, by an API key / JWT.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpapi.Success(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/tower/info", fleetH.handleTowerInfo)
	mux.Handle("GET /api/dashboard", t.userAuth(http.HandlerFunc(dash.handleDashboard)))
	mux.Handle("GET /api/system/self-test", t.userAuth(http.HandlerFunc(dash.handleSelfTest)))
	mux.Handle("GET /api/fleet/ws", t.userAuth(http.HandlerFunc(fleetH.handleFleetWS)))

	// Fleet API. If the operator configured an API key or JWT, require it.
	mux.Handle("GET /api/devices", t.userAuth(http.HandlerFunc(fleetH.handleListDevices)))
	mux.Handle("POST /api/devices/deploy", t.userAuth(http.HandlerFunc(deploy.handleDeployDevice)))
	mux.Handle("GET /api/devices/{id}", t.userAuth(http.HandlerFunc(fleetH.handleGetDevice)))
	mux.Handle("DELETE /api/devices/{id}", t.userAuth(http.HandlerFunc(fleetH.handleDeleteDevice)))
	mux.Handle("POST /api/devices/{id}/start", t.userAuth(http.HandlerFunc(deploy.handleStartDevice)))
	mux.Handle("POST /api/devices/{id}/stop", t.userAuth(http.HandlerFunc(deploy.handleStopDevice)))
	mux.Handle("POST /api/devices/{id}/diagnose", t.userAuth(http.HandlerFunc(deploy.handleDiagnoseDevice)))
	mux.Handle("GET /api/devices/{id}/actions", t.userAuth(http.HandlerFunc(deploy.handleListDeviceActions)))
	mux.Handle("GET /api/devices/{id}/history", t.userAuth(http.HandlerFunc(device.handleDeviceHistory)))
	mux.Handle("POST /api/devices/{id}/actions/{actionId}/undo", t.userAuth(http.HandlerFunc(deploy.handleUndoAction)))
	mux.Handle("POST /api/devices/{id}/trust-host", t.userAuth(http.HandlerFunc(deploy.handleTrustHost)))

	// Browser-facing device API. The master translates each call into gRPC
	// commands on the device's agent stream.
	mux.Handle("GET /api/devices/{id}/api/system/stats", t.userAuth(http.HandlerFunc(device.handleDeviceStats)))
	mux.Handle("GET /api/devices/{id}/api/system/stats/ws", t.userAuth(http.HandlerFunc(device.handleDeviceStatsWS)))
	mux.Handle("GET /api/devices/{id}/api/system/processes", t.userAuth(http.HandlerFunc(device.handleDeviceProcesses)))
	mux.Handle("GET /api/devices/{id}/api/jobs", t.userAuth(http.HandlerFunc(device.handleListJobs)))
	mux.Handle("POST /api/devices/{id}/api/jobs", t.userAuth(http.HandlerFunc(device.handleRunJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}", t.userAuth(http.HandlerFunc(device.handleGetJob)))
	mux.Handle("GET /api/devices/{id}/api/jobs/{jobId}/logs", t.userAuth(http.HandlerFunc(device.handleJobLogs)))
	mux.Handle("POST /api/devices/{id}/api/jobs/{jobId}/stop", t.userAuth(http.HandlerFunc(device.handleCancelJob)))
	mux.Handle("DELETE /api/devices/{id}/api/jobs/{jobId}", t.userAuth(http.HandlerFunc(device.handleDeleteJob)))
	mux.Handle("GET /api/devices/{id}/api/files", t.userAuth(http.HandlerFunc(device.handleListFiles)))
	mux.Handle("DELETE /api/devices/{id}/api/files", t.userAuth(http.HandlerFunc(device.handleDeleteFile)))
	mux.Handle("GET /api/devices/{id}/api/files/read", t.userAuth(http.HandlerFunc(device.handleReadFile)))
	mux.Handle("GET /api/devices/{id}/api/files/download", t.userAuth(http.HandlerFunc(device.handleDownloadFile)))
	mux.Handle("POST /api/devices/{id}/api/files/write", t.userAuth(http.HandlerFunc(device.handleWriteFile)))
	mux.Handle("POST /api/devices/{id}/api/files/mkdir", t.userAuth(http.HandlerFunc(device.handleMkdir)))
	mux.Handle("POST /api/devices/{id}/api/files/rename", t.userAuth(http.HandlerFunc(device.handleRename)))
	mux.Handle("POST /api/devices/{id}/api/files/upload", t.userAuth(http.HandlerFunc(device.handleUploadFile)))
	mux.Handle("GET /api/devices/{id}/api/terminal", t.userAuth(http.HandlerFunc(device.handleTerminalWS)))

	// SSH deployment and tailnet discovery.
	mux.Handle("GET /api/deployments", t.userAuth(http.HandlerFunc(deploy.handleListDeployments)))
	mux.Handle("GET /api/deployments/{id}", t.userAuth(http.HandlerFunc(deploy.handleGetDeployment)))
	mux.Handle("GET /api/tailnet/peers", t.userAuth(http.HandlerFunc(deploy.handleTailnetPeers)))

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
