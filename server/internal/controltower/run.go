package controltower

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/logger"
)

// Run starts the control tower and blocks until it receives SIGINT/SIGTERM.
func Run(cfg *config.Config, version string) error {
	srv, err := New(cfg, version)
	if err != nil {
		return err
	}
	defer srv.Close()

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("Control tower listening on http://%s", httpServer.Addr)
		logger.Info("Join token file: %s", cfg.DataDir+"/master-join-token")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Control tower failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal %v, shutting down control tower...", sig)

	timeout := cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Control tower forced to shut down: %v", err)
	}
	logger.Info("Control tower stopped")
	return nil
}
