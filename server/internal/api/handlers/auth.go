package handlers

import (
	"net/http"

	"github.com/ss497254/gloski/internal/auth"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Status handles GET /api/auth/status
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]bool{"authenticated": true})
}
