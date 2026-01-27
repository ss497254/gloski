package handlers

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/auth"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/terminal"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Origin checking handled by auth - requires valid API key or JWT
	},
}

// TerminalHandler handles terminal WebSocket connections
type TerminalHandler struct {
	config      *config.Config
	authService *auth.Service
	sessions    sync.Map // map[string]*terminal.Terminal - active terminal sessions
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(cfg *config.Config, authService *auth.Service) *TerminalHandler {
	return &TerminalHandler{
		config:      cfg,
		authService: authService,
	}
}

// Shutdown closes all active terminal sessions
func (h *TerminalHandler) Shutdown() {
	h.sessions.Range(func(key, value interface{}) bool {
		if term, ok := value.(*terminal.Terminal); ok {
			term.Close()
		}
		return true
	})
	logger.Info("All terminal sessions closed")
}

// ActiveSessions returns the count of active terminal sessions
func (h *TerminalHandler) ActiveSessions() int {
	count := 0
	h.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Handle handles GET /api/terminal (WebSocket upgrade)
func (h *TerminalHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Auth via query param for WebSocket - try API key first, then JWT
	apiKey := r.URL.Query().Get("api_key")
	token := r.URL.Query().Get("token")

	authenticated := false

	if apiKey != "" {
		if err := h.authService.ValidateAPIKey(apiKey); err == nil {
			authenticated = true
		}
	}

	if !authenticated && token != "" {
		if err := h.authService.ValidateToken(token); err == nil {
			authenticated = true
		}
	}

	if !authenticated {
		http.Error(w, "invalid or missing authentication", http.StatusUnauthorized)
		return
	}

	cwd := r.URL.Query().Get("cwd")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}

	sessionID := uuid.New().String()
	term, err := terminal.New(sessionID, conn, h.config.Shell, cwd)
	if err != nil {
		logger.Error("Terminal creation failed: %v", err)
		conn.Close()
		return
	}

	// Track the session
	h.sessions.Store(sessionID, term)
	logger.Debug("Terminal session started: %s", sessionID)

	// Run terminal (blocks until closed)
	term.Run()

	// Remove from tracking
	h.sessions.Delete(sessionID)
	logger.Debug("Terminal session ended: %s", sessionID)
}
