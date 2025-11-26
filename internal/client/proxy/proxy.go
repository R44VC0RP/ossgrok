package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/R44VC0RP/ossgrok/internal/protocol"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

// Proxy handles proxying HTTP requests to a local application
type Proxy struct {
	localURL string
	client   *http.Client
}

// New creates a new HTTP proxy
func New(localURL string) *Proxy {
	return &Proxy{
		localURL: localURL,
		client: &http.Client{
			Timeout: 0, // No timeout, let the server handle it
		},
	}
}

// ProxyRequest proxies an HTTP request to the local application
func (p *Proxy) ProxyRequest(req *protocol.HTTPRequestMessage) (*protocol.HTTPResponseMessage, error) {
	// Build target URL
	targetURL := p.localURL + req.Path

	logger.Debug("Proxying request: %s %s", req.Method, targetURL)

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Execute request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Create response message
	resp := &protocol.HTTPResponseMessage{
		RequestID:  req.RequestID,
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
	}

	logger.Debug("Request proxied successfully: %s %s -> %d", req.Method, req.Path, httpResp.StatusCode)

	return resp, nil
}
