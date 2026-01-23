package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/terminal"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Implement proper origin checking
	},
}

// TerminalHandler handles terminal WebSocket connections
type TerminalHandler struct {
	config      *config.Config
	authService *auth.Service
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(cfg *config.Config, authService *auth.Service) *TerminalHandler {
	return &TerminalHandler{
		config:      cfg,
		authService: authService,
	}
}

// Handle handles GET /api/terminal (WebSocket upgrade)
func (h *TerminalHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Auth via query param for WebSocket - try API key first, then JWT
	apiKey := r.URL.Query().Get("api_key")
	token := r.URL.Query().Get("token")

	if apiKey != "" {
		if err := h.authService.ValidateAPIKey(apiKey); err != nil {
			http.Error(w, "invalid API key", http.StatusUnauthorized)
			return
		}
	} else if token != "" {
		if err := h.authService.ValidateToken(token); err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	} else {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return
	}

	cwd := r.URL.Query().Get("cwd")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}

	term, err := terminal.New(uuid.New().String(), conn, h.config.Shell, cwd)
	if err != nil {
		logger.Error("Terminal creation failed: %v", err)
		conn.Close()
		return
	}

	logger.Debug("Terminal session started")
	term.Run()
	logger.Debug("Terminal session ended")
}
