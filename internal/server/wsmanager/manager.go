package wsmanager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/R44VC0RP/ossgrok/internal/protocol"
	"github.com/R44VC0RP/ossgrok/internal/server/registry"
	"github.com/R44VC0RP/ossgrok/internal/server/tunnel"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (configure as needed)
	},
}

// PendingRequest represents a pending HTTP request awaiting response
type PendingRequest struct {
	ResponseChan chan *protocol.HTTPResponseMessage
	Timeout      *time.Timer
}

// Manager handles WebSocket connections and message routing
type Manager struct {
	registry        *registry.Registry
	pendingRequests sync.Map // map[requestID]*PendingRequest
}

// New creates a new WebSocket manager
func New(reg *registry.Registry) *Manager {
	return &Manager{
		registry: reg,
	}
}

// HandleWebSocket handles incoming WebSocket connections
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection: %v", err)
		return
	}

	logger.Info("New WebSocket connection from %s", r.RemoteAddr)

	// Read the first message (should be registration)
	var msg protocol.Message
	if err := conn.ReadJSON(&msg); err != nil {
		logger.Error("Failed to read registration message: %v", err)
		conn.Close()
		return
	}

	if msg.Type != protocol.TypeRegister {
		logger.Error("Expected register message, got: %s", msg.Type)
		m.sendError(conn, "INVALID_MESSAGE", "Expected registration message")
		conn.Close()
		return
	}

	registerMsg, err := protocol.DecodeRegister(&msg)
	if err != nil {
		logger.Error("Failed to decode register message: %v", err)
		m.sendError(conn, "DECODE_ERROR", err.Error())
		conn.Close()
		return
	}

	// Generate tunnel ID
	tunnelID := generateTunnelID()

	// Create tunnel connection
	tunnelConn := tunnel.NewConnection(registerMsg.Domain, tunnelID, conn)

	// Register tunnel
	if err := m.registry.Register(registerMsg.Domain, tunnelConn); err != nil {
		logger.Error("Failed to register tunnel: %v", err)
		m.sendError(conn, "REGISTRATION_FAILED", err.Error())
		conn.Close()
		return
	}

	// Send registration confirmation
	registeredMsg, err := protocol.EncodeMessage(protocol.TypeRegistered, &protocol.RegisteredMessage{
		TunnelID:  tunnelID,
		ServerURL: fmt.Sprintf("https://%s", registerMsg.Domain),
	})
	if err != nil {
		logger.Error("Failed to encode registered message: %v", err)
		m.registry.Unregister(registerMsg.Domain)
		conn.Close()
		return
	}

	if err := conn.WriteJSON(registeredMsg); err != nil {
		logger.Error("Failed to send registered message: %v", err)
		m.registry.Unregister(registerMsg.Domain)
		conn.Close()
		return
	}

	logger.Info("Tunnel registered successfully: domain=%s, tunnel_id=%s", registerMsg.Domain, tunnelID)

	// Start listening for messages from the client
	m.handleConnection(tunnelConn)

	// Clean up on disconnect
	m.registry.Unregister(registerMsg.Domain)
	conn.Close()
}

// handleConnection handles messages from a tunnel connection
func (m *Manager) handleConnection(tunnelConn *tunnel.Connection) {
	for {
		msg, err := tunnelConn.ReadMessage()
		if err != nil {
			logger.Info("Tunnel disconnected: domain=%s, tunnel_id=%s, error=%v",
				tunnelConn.Domain(), tunnelConn.TunnelID(), err)
			return
		}

		switch msg.Type {
		case protocol.TypeHTTPResponse:
			m.handleHTTPResponse(msg)
		case protocol.TypePing:
			m.handlePing(tunnelConn)
		default:
			logger.Warn("Unknown message type from client: %s", msg.Type)
		}
	}
}

// handleHTTPResponse handles HTTP response from client
func (m *Manager) handleHTTPResponse(msg *protocol.Message) {
	resp, err := protocol.DecodeHTTPResponse(msg)
	if err != nil {
		logger.Error("Failed to decode HTTP response: %v", err)
		return
	}

	// Find the pending request
	if pending, ok := m.pendingRequests.LoadAndDelete(resp.RequestID); ok {
		pr := pending.(*PendingRequest)
		pr.Timeout.Stop()
		pr.ResponseChan <- resp
		close(pr.ResponseChan)
	} else {
		logger.Warn("Received response for unknown request ID: %s", resp.RequestID)
	}
}

// handlePing handles ping message
func (m *Manager) handlePing(tunnelConn *tunnel.Connection) {
	pongMsg, _ := protocol.EncodeMessage(protocol.TypePong, nil)
	if err := tunnelConn.SendMessage(pongMsg); err != nil {
		logger.Error("Failed to send pong: %v", err)
	}
}

// SendHTTPRequest sends an HTTP request to a tunnel and waits for response
func (m *Manager) SendHTTPRequest(domain string, req *protocol.HTTPRequestMessage) (*protocol.HTTPResponseMessage, error) {
	tunnelConn, ok := m.registry.GetTunnel(domain)
	if !ok {
		return nil, fmt.Errorf("no tunnel found for domain: %s", domain)
	}

	// Create pending request
	responseChan := make(chan *protocol.HTTPResponseMessage, 1)
	timeout := time.NewTimer(30 * time.Second)

	pr := &PendingRequest{
		ResponseChan: responseChan,
		Timeout:      timeout,
	}

	m.pendingRequests.Store(req.RequestID, pr)

	// Send request to client
	tc := tunnelConn.(*tunnel.Connection)
	if err := tc.SendHTTPRequest(req); err != nil {
		m.pendingRequests.Delete(req.RequestID)
		timeout.Stop()
		return nil, fmt.Errorf("failed to send request to tunnel: %w", err)
	}

	// Wait for response or timeout
	select {
	case resp := <-responseChan:
		return resp, nil
	case <-timeout.C:
		m.pendingRequests.Delete(req.RequestID)
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// sendError sends an error message to a connection
func (m *Manager) sendError(conn *websocket.Conn, code, message string) {
	errMsg, _ := protocol.EncodeMessage(protocol.TypeError, &protocol.ErrorMessage{
		Code:    code,
		Message: message,
	})
	conn.WriteJSON(errMsg)
}

// generateTunnelID generates a random tunnel ID
func generateTunnelID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
