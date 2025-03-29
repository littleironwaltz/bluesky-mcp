package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/littleironwaltz/bluesky-mcp/configs/fallbacks"
	"github.com/littleironwaltz/bluesky-mcp/internal/auth"
	"github.com/littleironwaltz/bluesky-mcp/internal/handlers"
	"github.com/littleironwaltz/bluesky-mcp/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Application instance
type App struct {
	server      *echo.Echo
	config      config.Config
	shutdownWg  sync.WaitGroup
	healthySrv  *http.Server
	healthyStop chan struct{}
}

func main() {
	// Initialize application
	app := &App{
		healthyStop: make(chan struct{}),
	}

	// Set up signal handling
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	app.config = config.LoadConfig()
	
	// Validate configuration
	if err := config.ValidateConfig(app.config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Register backup credentials if available from environment
	backupID := os.Getenv("BSKY_BACKUP_ID")
	backupPassword := os.Getenv("BSKY_BACKUP_PASSWORD")
	if backupID != "" && backupPassword != "" {
		auth.RegisterBackupCredentials(auth.BackupCredentials{
			BskyID:       backupID,
			BskyPassword: backupPassword,
		})
		log.Println("Registered backup credentials")
	}
	
	// Initialize the auth token manager to ensure it's ready
	tokenManager := auth.GetTokenManager(app.config)
	
	// Initialize fallbacks for the auth token manager's client
	if err := fallbacks.InitializeFallbacks(tokenManager.GetClient()); err != nil {
		log.Printf("Warning: Failed to initialize fallbacks: %v\n", err)
	}

	// Initialize API server
	if err := app.initServer(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start health check server on a different port
	app.startHealthCheckServer()

	// Start main server
	go func() {
		if err := app.server.Start(":3000"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("Server started on port 3000")
	log.Println("Health check server started on port 3001")

	// Wait for termination signal
	<-done
	log.Println("Shutting down...")

	// Shutdown gracefully
	app.shutdown()

	log.Println("Server stopped")
}

// initServer initializes the Echo server
func (a *App) initServer() error {
	// Set up Echo
	a.server = echo.New()
	
	// Middleware
	a.server.Use(middleware.Recover())
	a.server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${remote_ip} ${method} ${uri} ${status} ${latency_human}\n",
	}))
	a.server.Use(middleware.CORS())
	a.server.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}))
	
	// Add request ID middleware
	a.server.Use(middleware.RequestID())
	
	// Add response timeout middleware
	a.server.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	// Routes
	a.server.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})
	
	a.server.POST("/mcp/:method", func(c echo.Context) error {
		return handlers.HandleMCPRequest(c, a.config)
	})

	return nil
}

// startHealthCheckServer starts a separate HTTP server for health checks
func (a *App) startHealthCheckServer() {
	a.healthySrv = &http.Server{
		Addr: ":3001",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}),
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()
		if err := a.healthySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check server error: %v", err)
		}
	}()

	// Watch for stop signal
	go func() {
		<-a.healthyStop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.healthySrv.Shutdown(ctx); err != nil {
			log.Printf("Health check server shutdown error: %v", err)
		}
	}()
}

// shutdown gracefully stops the application
func (a *App) shutdown() {
	// First, stop the main server
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	// Signal health check server to stop
	close(a.healthyStop)

	// Wait for health check server to complete shutdown
	waitCh := make(chan struct{})
	go func() {
		a.shutdownWg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// Shutdown completed normally
	case <-time.After(5 * time.Second):
		log.Println("Health check server shutdown timed out")
	}
	
	// Stop background token refreshes
	auth.GetTokenManager(a.config).Stop()
}