package api

import (
	"net/http"

	"github.com/ss497254/gloski/internal/api/routes"
	"github.com/ss497254/gloski/internal/app"
)

// Server represents the HTTP server
type Server struct {
	router   http.Handler
	app      *app.App
	handlers *routes.RouteHandlers
}

// ServerOptions contains options for creating a new server.
type ServerOptions struct {
	Version string
}

// NewServer creates a new API server from the application container
func NewServer(application *app.App, opts ...ServerOptions) *Server {
	var version string
	if len(opts) > 0 {
		version = opts[0].Version
	}

	// Setup routes with all services from the app
	router, handlers := routes.Setup(routes.Config{
		Cfg:             application.Config,
		AuthService:     application.Auth,
		FileService:     application.Files,
		JobsService:     application.Jobs,
		SysService:      application.System,
		DB:              application.DB.DB(),
		PackagesService: application.Packages,
		CronService:     application.Cron,
		DownloadService: application.Downloads,
		Features:        application.Features(),
		Version:         version,
	})

	return &Server{
		router:   router,
		app:      application,
		handlers: handlers,
	}
}

// Router returns the HTTP handler for the server
func (s *Server) Router() http.Handler {
	return s.router
}

// Shutdown closes all server resources (terminal sessions, etc.)
func (s *Server) Shutdown() {
	if s.handlers != nil && s.handlers.TerminalHandler != nil {
		s.handlers.TerminalHandler.Shutdown()
	}
}
