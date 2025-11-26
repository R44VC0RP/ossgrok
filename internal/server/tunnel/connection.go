package tunnel

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/R44VC0RP/ossgrok/internal/protocol"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

// Connection represents a tunnel connection to a client
type Connection struct {
	domain   string
	tunnelID string
	conn     *websocket.Conn
	mu       sync.Mutex
}

// NewConnection creates a new tunnel connection
func NewConnection(domain, tunnelID string, conn *websocket.Conn) *Connection {
	return &Connection{
		domain:   domain,
		tunnelID: tunnelID,
		conn:     conn,
	}
}

// Domain returns the domain this tunnel serves
func (c *Connection) Domain() string {
	return c.domain
}

// TunnelID returns the tunnel ID
func (c *Connection) TunnelID() string {
	return c.tunnelID
}

// SendMessage sends a message to the client
func (c *Connection) SendMessage(msg *protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendHTTPRequest sends an HTTP request to the client
func (c *Connection) SendHTTPRequest(req *protocol.HTTPRequestMessage) error {
	msg, err := protocol.EncodeMessage(protocol.TypeHTTPRequest, req)
	if err != nil {
		return fmt.Errorf("failed to encode HTTP request: %w", err)
	}

	return c.SendMessage(msg)
}

// ReadMessage reads a message from the client
func (c *Connection) ReadMessage() (*protocol.Message, error) {
	var msg protocol.Message
	if err := c.conn.ReadJSON(&msg); err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	return &msg, nil
}

// Close closes the tunnel connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Debug("Closing tunnel connection for domain: %s (tunnel_id: %s)", c.domain, c.tunnelID)

	// Send close message
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "tunnel closed")
	if err := c.conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
		logger.Warn("Failed to send close message: %v", err)
	}

	return c.conn.Close()
}

// Conn returns the underlying WebSocket connection
func (c *Connection) Conn() *websocket.Conn {
	return c.conn
}

// MarshalJSON implements json.Marshaler for logging
func (c *Connection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"domain":    c.domain,
		"tunnel_id": c.tunnelID,
	})
}
