package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// GracefulServer wraps Echo server with graceful shutdown capabilities
type GracefulServer struct {
	echo   *echo.Echo
	logger *logger.ZapLogger
	port   int
}

// NewGracefulServer creates a new server with graceful shutdown
func NewGracefulServer(e *echo.Echo, zapLogger *logger.ZapLogger, port int) *GracefulServer {
	return &GracefulServer{
		echo:   e,
		logger: zapLogger,
		port:   port,
	}
}

// Start starts the server with graceful shutdown handling
func (s *GracefulServer) Start() error {
	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", s.port)
		s.logger.Info("Starting HTTP server", logger.String("address", addr))

		if err := s.echo.Start(addr); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", logger.Err(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Kill signal sent from terminal (Ctrl+C)
	// SIGTERM signal sent from Kubernetes or Docker
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	sig := <-quit
	s.logger.Info("Received shutdown signal", logger.String("signal", sig.String()))

	// Graceful shutdown with timeout
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *GracefulServer) Shutdown() error {
	s.logger.Info("Shutting down server gracefully...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the server
	if err := s.echo.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", logger.Err(err))
		return err
	}

	s.logger.Info("Server shutdown completed")
	return nil
}

// ShutdownComponents provides a way to register cleanup functions
type ShutdownManager struct {
	logger    *logger.ZapLogger
	functions []func(context.Context) error
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(zapLogger *logger.ZapLogger) *ShutdownManager {
	return &ShutdownManager{
		logger:    zapLogger,
		functions: make([]func(context.Context) error, 0),
	}
}

// Register adds a cleanup function to be called during shutdown
func (sm *ShutdownManager) Register(fn func(context.Context) error) {
	sm.functions = append(sm.functions, fn)
}

// Shutdown executes all registered cleanup functions
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	sm.logger.Info("Starting graceful shutdown of components", logger.Int("components", len(sm.functions)))

	for i, fn := range sm.functions {
		if err := fn(ctx); err != nil {
			sm.logger.Error("Error during component shutdown",
				logger.Int("component", i),
				logger.Err(err))
			// Continue with other components even if one fails
		}
	}

	sm.logger.Info("All components shutdown completed")
	return nil
}
