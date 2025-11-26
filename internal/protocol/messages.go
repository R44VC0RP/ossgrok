package protocol

import (
	"encoding/json"
	"fmt"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	TypeRegister     MessageType = "register"
	TypeRegistered   MessageType = "registered"
	TypeHTTPRequest  MessageType = "http_request"
	TypeHTTPResponse MessageType = "http_response"
	TypePing         MessageType = "ping"
	TypePong         MessageType = "pong"
	TypeError        MessageType = "error"
)

// Message is the base message structure
type Message struct {
	Type MessageType `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// RegisterMessage is sent from client to server to register a domain
type RegisterMessage struct {
	Domain          string `json:"domain"`
	ProtocolVersion string `json:"protocol_version"`
}

// RegisteredMessage is sent from server to client after successful registration
type RegisteredMessage struct {
	TunnelID  string `json:"tunnel_id"`
	ServerURL string `json:"server_url"`
}

// HTTPRequestMessage is sent from server to client with HTTP request to proxy
type HTTPRequestMessage struct {
	RequestID string              `json:"request_id"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Headers   map[string][]string `json:"headers"`
	Body      []byte              `json:"body,omitempty"`
}

// HTTPResponseMessage is sent from client to server with HTTP response
type HTTPResponseMessage struct {
	RequestID  string              `json:"request_id"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body,omitempty"`
}

// ErrorMessage is sent when an error occurs
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// EncodeMessage wraps a typed message into a generic Message
func EncodeMessage(msgType MessageType, data interface{}) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message data: %w", err)
	}

	return &Message{
		Type: msgType,
		Data: dataBytes,
	}, nil
}

// DecodeRegister decodes a register message
func DecodeRegister(msg *Message) (*RegisterMessage, error) {
	var register RegisterMessage
	if err := json.Unmarshal(msg.Data, &register); err != nil {
		return nil, fmt.Errorf("failed to decode register message: %w", err)
	}
	return &register, nil
}

// DecodeRegistered decodes a registered message
func DecodeRegistered(msg *Message) (*RegisteredMessage, error) {
	var registered RegisteredMessage
	if err := json.Unmarshal(msg.Data, &registered); err != nil {
		return nil, fmt.Errorf("failed to decode registered message: %w", err)
	}
	return &registered, nil
}

// DecodeHTTPRequest decodes an HTTP request message
func DecodeHTTPRequest(msg *Message) (*HTTPRequestMessage, error) {
	var req HTTPRequestMessage
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("failed to decode HTTP request message: %w", err)
	}
	return &req, nil
}

// DecodeHTTPResponse decodes an HTTP response message
func DecodeHTTPResponse(msg *Message) (*HTTPResponseMessage, error) {
	var resp HTTPResponseMessage
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode HTTP response message: %w", err)
	}
	return &resp, nil
}

// DecodeError decodes an error message
func DecodeError(msg *Message) (*ErrorMessage, error) {
	var errMsg ErrorMessage
	if err := json.Unmarshal(msg.Data, &errMsg); err != nil {
		return nil, fmt.Errorf("failed to decode error message: %w", err)
	}
	return &errMsg, nil
}
