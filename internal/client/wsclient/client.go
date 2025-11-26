package wsclient

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/R44VC0RP/ossgrok/internal/client/proxy"
	"github.com/R44VC0RP/ossgrok/internal/protocol"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

// Client represents a WebSocket client for tunneling
type Client struct {
	serverURL string
	domain    string
	proxy     *proxy.Proxy
	conn      *websocket.Conn
	tunnelID  string
}

// New creates a new WebSocket client
func New(serverURL, domain string, localPort int) *Client {
	localURL := fmt.Sprintf("http://localhost:%d", localPort)
	return &Client{
		serverURL: serverURL,
		domain:    domain,
		proxy:     proxy.New(localURL),
	}
}

// Connect connects to the server and registers the tunnel
func (c *Client) Connect() error {
	logger.Info("Connecting to server: %s", c.serverURL)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(c.serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.conn = conn

	// Send registration message
	registerMsg, err := protocol.EncodeMessage(protocol.TypeRegister, &protocol.RegisterMessage{
		Domain:          c.domain,
		ProtocolVersion: "1.0",
	})
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to encode register message: %w", err)
	}

	if err := c.conn.WriteJSON(registerMsg); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send register message: %w", err)
	}

	// Wait for registration confirmation
	var msg protocol.Message
	if err := c.conn.ReadJSON(&msg); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to read registration response: %w", err)
	}

	if msg.Type == protocol.TypeError {
		errMsg, _ := protocol.DecodeError(&msg)
		c.conn.Close()
		return fmt.Errorf("registration failed: %s - %s", errMsg.Code, errMsg.Message)
	}

	if msg.Type != protocol.TypeRegistered {
		c.conn.Close()
		return fmt.Errorf("unexpected message type: %s", msg.Type)
	}

	registered, err := protocol.DecodeRegistered(&msg)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to decode registered message: %w", err)
	}

	c.tunnelID = registered.TunnelID

	logger.Info("Tunnel registered successfully!")
	logger.Info("  Tunnel ID: %s", c.tunnelID)
	logger.Info("  Public URL: %s", registered.ServerURL)
	logger.Info("  Forwarding to: http://localhost:%s", c.proxy)
	logger.Info("")
	logger.Info("Tunnel is active. Press Ctrl+C to stop.")

	return nil
}

// Run starts the client event loop
func (c *Client) Run() error {
	// Start heartbeat
	go c.heartbeat()

	// Listen for messages
	for {
		var msg protocol.Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			logger.Error("Connection error: %v", err)
			return fmt.Errorf("connection closed: %w", err)
		}

		switch msg.Type {
		case protocol.TypeHTTPRequest:
			go c.handleHTTPRequest(&msg)
		case protocol.TypePong:
			// Heartbeat response, ignore
		default:
			logger.Warn("Unknown message type: %s", msg.Type)
		}
	}
}

// handleHTTPRequest handles an incoming HTTP request from the server
func (c *Client) handleHTTPRequest(msg *protocol.Message) {
	req, err := protocol.DecodeHTTPRequest(msg)
	if err != nil {
		logger.Error("Failed to decode HTTP request: %v", err)
		return
	}

	logger.Debug("Received request: %s %s", req.Method, req.Path)

	// Proxy request to local application
	resp, err := c.proxy.ProxyRequest(req)
	if err != nil {
		logger.Error("Failed to proxy request: %v", err)

		// Send error response
		resp = &protocol.HTTPResponseMessage{
			RequestID:  req.RequestID,
			StatusCode: 502,
			Headers:    make(map[string][]string),
			Body:       []byte("Bad Gateway: " + err.Error()),
		}
	}

	// Send response back to server
	respMsg, err := protocol.EncodeMessage(protocol.TypeHTTPResponse, resp)
	if err != nil {
		logger.Error("Failed to encode response: %v", err)
		return
	}

	if err := c.conn.WriteJSON(respMsg); err != nil {
		logger.Error("Failed to send response: %v", err)
	}
}

// heartbeat sends periodic ping messages to keep the connection alive
func (c *Client) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pingMsg, _ := protocol.EncodeMessage(protocol.TypePing, nil)
		if err := c.conn.WriteJSON(pingMsg); err != nil {
			logger.Error("Failed to send ping: %v", err)
			return
		}
	}
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	if c.conn != nil {
		logger.Info("Closing tunnel connection...")
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		c.conn.WriteMessage(websocket.CloseMessage, closeMsg)
		return c.conn.Close()
	}
	return nil
}
