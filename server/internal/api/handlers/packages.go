package handlers

import (
	"net/http"
	"strconv"

	"github.com/ss497254/gloski/internal/packages"
)

// PackagesHandler handles package management requests.
type PackagesHandler struct {
	service *packages.Service
}

// NewPackagesHandler creates a new packages handler.
func NewPackagesHandler(service *packages.Service) *PackagesHandler {
	return &PackagesHandler{service: service}
}

// Available returns whether package management is available.
func (h *PackagesHandler) Available() bool {
	return h.service != nil
}

// Info handles GET /api/packages/info
func (h *PackagesHandler) Info(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Package management not available")
		return
	}

	Success(w, map[string]interface{}{
		"manager": h.service.Manager(),
	})
}

// ListInstalled handles GET /api/packages/installed
func (h *PackagesHandler) ListInstalled(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Package management not available")
		return
	}

	limit := 100
	if n := r.URL.Query().Get("limit"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	pkgs, err := h.service.ListInstalled(limit)
	if err != nil {
		InternalError(w, "failed to list installed packages", err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"manager":  h.service.Manager(),
		"packages": pkgs,
		"count":    len(pkgs),
	})
}

// CheckUpgrades handles GET /api/packages/upgrades
func (h *PackagesHandler) CheckUpgrades(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Package management not available")
		return
	}

	info, err := h.service.CheckUpgrades()
	if err != nil {
		InternalError(w, "failed to check upgrades", err.Error())
		return
	}

	Success(w, info)
}

// Search handles GET /api/packages/search
func (h *PackagesHandler) Search(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Package management not available")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		BadRequest(w, "search query required")
		return
	}

	limit := 50
	if n := r.URL.Query().Get("limit"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	pkgs, err := h.service.Search(query, limit)
	if err != nil {
		InternalError(w, "failed to search packages", err.Error())
		return
	}

	Success(w, map[string]interface{}{
		"packages": pkgs,
		"count":    len(pkgs),
	})
}

// GetPackageInfo handles GET /api/packages/{name}
func (h *PackagesHandler) GetPackageInfo(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		Error(w, http.StatusServiceUnavailable, "Package management not available")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		BadRequest(w, "package name required")
		return
	}

	pkg, err := h.service.GetPackageInfo(name)
	if err != nil {
		InternalError(w, "failed to get package info", err.Error())
		return
	}

	Success(w, pkg)
}
