package controltower

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/metrics"
)

const defaultHistoryRange = 24 * time.Hour

// handleDeviceHistory returns metric samples recorded by the master from this
// device's heartbeats. It is a master-owned read; no agent call is involved.
func (d *deviceHandlers) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if _, err := d.fleetSvc.Get(deviceID); err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}

	metric := r.URL.Query().Get("metric")
	within := defaultHistoryRange
	if raw := r.URL.Query().Get("range"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 || parsed > 365*24*time.Hour {
			httpapi.BadRequest(w, "invalid range")
			return
		}
		within = parsed
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > metrics.MaxHistoryLimit {
			httpapi.BadRequest(w, "invalid limit")
			return
		}
		limit = parsed
	}

	samples, err := d.metricStore.History(r.Context(), deviceID, metric, within, limit)
	if err != nil {
		httpapi.InternalError(w, "failed to load metric history", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{
		"samples": samples,
		"metric":  metric,
		"range":   within.String(),
	})
}
