package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/R44VC0RP/ossgrok/internal/server/httphandler"
	"github.com/R44VC0RP/ossgrok/internal/server/registry"
	"github.com/R44VC0RP/ossgrok/internal/server/wsmanager"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

func main() {
	// Set log level from environment
	logLevel := getEnv("LOG_LEVEL", "info")
	logger.SetLevel(logLevel)

	logger.Info("Starting ossgrok server...")

	// Get configuration from environment
	httpPort := getEnv("SERVER_HTTP_PORT", "80")
	httpsPort := getEnv("SERVER_HTTPS_PORT", "443")
	wsPort := getEnv("SERVER_WS_PORT", "4443")
	autocertDomains := getEnv("AUTOCERT_DOMAINS", "")
	autocertEmail := getEnv("AUTOCERT_EMAIL", "")
	autocertCacheDir := getEnv("AUTOCERT_CACHE_DIR", "/var/lib/autocert")

	if autocertDomains == "" {
		logger.Fatal("AUTOCERT_DOMAINS environment variable is required")
	}

	domains := strings.Split(autocertDomains, ",")
	for i, d := range domains {
		domains[i] = strings.TrimSpace(d)
	}

	logger.Info("Configured domains: %v", domains)

	// Create tunnel registry
	reg := registry.New()

	// Create WebSocket manager
	wsManager := wsmanager.New(reg)

	// Create HTTP handler
	httpHandler := httphandler.New(wsManager)

	// Setup autocert manager
	certManager := &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(domains...),
		Cache:       autocert.DirCache(autocertCacheDir),
		Email:       autocertEmail,
	}

	// Create HTTP server for ACME challenges and redirect
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: certManager.HTTPHandler(http.HandlerFunc(redirectToHTTPS)),
	}

	// Create HTTPS server for tunnel traffic
	httpsServer := &http.Server{
		Addr:      ":" + httpsPort,
		Handler:   httpHandler,
		TLSConfig: certManager.TLSConfig(),
	}

	// Create WebSocket server for control plane
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/tunnel", wsManager.HandleWebSocket)

	wsServer := &http.Server{
		Addr:      ":" + wsPort,
		Handler:   wsMux,
		TLSConfig: certManager.TLSConfig(),
	}

	// Start HTTP server (for ACME challenges)
	go func() {
		logger.Info("Starting HTTP server on port %s (ACME challenges & redirects)", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS server (for tunnel traffic)
	go func() {
		logger.Info("Starting HTTPS server on port %s (tunnel traffic)", httpsPort)
		if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTPS server error: %v", err)
		}
	}()

	// Start WebSocket server (for control plane)
	go func() {
		logger.Info("Starting WebSocket server on port %s (control plane)", wsPort)
		if err := wsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Fatal("WebSocket server error: %v", err)
		}
	}()

	logger.Info("ossgrok server is running!")
	logger.Info("Active tunnels: 0")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")

	// Shutdown servers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error: %v", err)
	}

	if err := httpsServer.Shutdown(ctx); err != nil {
		logger.Error("HTTPS server shutdown error: %v", err)
	}

	if err := wsServer.Shutdown(ctx); err != nil {
		logger.Error("WebSocket server shutdown error: %v", err)
	}

	logger.Info("Server stopped")
}

// redirectToHTTPS redirects HTTP requests to HTTPS
func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.RequestURI()
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
