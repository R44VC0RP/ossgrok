package httphandler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/R44VC0RP/ossgrok/internal/protocol"
	"github.com/R44VC0RP/ossgrok/internal/server/wsmanager"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

// Handler handles HTTP requests and routes them to tunnels
type Handler struct {
	wsManager *wsmanager.Manager
}

// New creates a new HTTP handler
func New(wsManager *wsmanager.Manager) *Handler {
	return &Handler{
		wsManager: wsManager,
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract domain from Host header
	domain := r.Host
	if domain == "" {
		http.Error(w, "Missing Host header", http.StatusBadRequest)
		return
	}

	logger.Debug("Received request for domain: %s, path: %s", domain, r.URL.Path)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Generate unique request ID
	requestID := generateRequestID()

	// Create HTTP request message
	req := &protocol.HTTPRequestMessage{
		RequestID: requestID,
		Method:    r.Method,
		Path:      r.URL.RequestURI(),
		Headers:   r.Header,
		Body:      body,
	}

	// Send request to tunnel and wait for response
	resp, err := h.wsManager.SendHTTPRequest(domain, req)
	if err != nil {
		logger.Error("Failed to send request to tunnel: %v", err)

		if err.Error() == fmt.Sprintf("no tunnel found for domain: %s", domain) {
			http.Error(w, fmt.Sprintf("No tunnel registered for domain: %s", domain), http.StatusServiceUnavailable)
		} else if err.Error() == "timeout waiting for response" {
			http.Error(w, "Gateway timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Write response headers
	for key, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Write response body
	if len(resp.Body) > 0 {
		if _, err := w.Write(resp.Body); err != nil {
			logger.Error("Failed to write response body: %v", err)
		}
	}

	logger.Debug("Request completed: domain=%s, path=%s, status=%d", domain, r.URL.Path, resp.StatusCode)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "req-" + hex.EncodeToString(b)
}
