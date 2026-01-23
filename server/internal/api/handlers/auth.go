package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/logger"
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

// LoginRequest represents a login request body
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.Password == "" {
		BadRequest(w, "password is required")
		return
	}

	token, expiresIn, err := h.authService.Login(req.Password)
	if err != nil {
		logger.Warn("Login failed from %s", r.RemoteAddr)
		Unauthorized(w, "invalid password")
		return
	}

	logger.Info("Login successful from %s", r.RemoteAddr)
	Success(w, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// Status handles GET /api/auth/status
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	Success(w, map[string]bool{"authenticated": true})
}
