package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/test/main.go <server-url>")
		fmt.Println("Example: go run cmd/test/main.go ossgrok.sevalla.app")
		os.Exit(1)
	}

	serverURL := os.Args[1]

	fmt.Println("========================================")
	fmt.Println("ossgrok Server Health Check")
	fmt.Println("========================================")
	fmt.Printf("\nTesting server: %s\n\n", serverURL)

	passed := 0
	total := 4

	// Test 1: HTTP endpoint
	fmt.Println("[1/4] Testing HTTP endpoint...")
	if testHTTP(serverURL) {
		fmt.Println("✓ HTTP endpoint is accessible")
		passed++
	} else {
		fmt.Println("✗ HTTP endpoint failed (might be expected if platform redirects)")
	}
	fmt.Println()

	// Test 2: HTTPS endpoint
	fmt.Println("[2/4] Testing HTTPS endpoint...")
	if testHTTPS(serverURL) {
		fmt.Println("✓ HTTPS endpoint is accessible")
		fmt.Println("  (503 Service Unavailable is expected when no tunnel is registered)")
		passed++
	} else {
		fmt.Println("✗ HTTPS endpoint failed")
	}
	fmt.Println()

	// Test 3: WebSocket endpoint
	fmt.Println("[3/4] Testing WebSocket control plane (port 4443)...")
	if testWebSocket(serverURL) {
		fmt.Println("✓ WebSocket endpoint is accessible and accepting connections")
		passed++
	} else {
		fmt.Println("✗ WebSocket endpoint failed")
	}
	fmt.Println()

	// Test 4: WebSocket protocol
	fmt.Println("[4/4] Testing WebSocket protocol (registration)...")
	if testWebSocketProtocol(serverURL) {
		fmt.Println("✓ WebSocket protocol is working correctly")
		passed++
	} else {
		fmt.Println("✗ WebSocket protocol test failed")
	}
	fmt.Println()

	// Summary
	fmt.Println("========================================")
	fmt.Println("Test Summary")
	fmt.Println("========================================")
	fmt.Printf("\nPassed: %d/%d tests\n\n", passed, total)

	if passed == total {
		fmt.Println("✓ Server is healthy and ready to accept tunnels!")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("1. Configure client: ossgrok config --server %s\n", serverURL)
		fmt.Printf("2. Create tunnel: ossgrok --url test.yourdomain.com 3000\n")
	} else {
		fmt.Println("⚠ Some tests failed. Check server configuration and logs.")
	}

	fmt.Println("\n========================================")
}

func testHTTP(serverURL string) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err := client.Get("http://" + serverURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Accept any response (200, 301, 404, 503)
	return resp.StatusCode > 0
}

func testHTTPS(serverURL string) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get("https://" + serverURL)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	fmt.Printf("  Status: %d %s\n", resp.StatusCode, resp.Status)

	// Accept 503 (no tunnel) or 200 (has tunnel) or 404
	return resp.StatusCode == 503 || resp.StatusCode == 200 || resp.StatusCode == 404
}

func testWebSocket(serverURL string) bool {
	wsURL := fmt.Sprintf("wss://%s:4443/tunnel", serverURL)

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	dialer.HandshakeTimeout = 5 * time.Second

	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			fmt.Printf("  Error: %v (Status: %d)\n", err, resp.StatusCode)
		} else {
			fmt.Printf("  Error: %v\n", err)
		}
		return false
	}
	defer conn.Close()

	fmt.Printf("  Connected to: %s\n", wsURL)
	return true
}

func testWebSocketProtocol(serverURL string) bool {
	wsURL := fmt.Sprintf("wss://%s:4443/tunnel", serverURL)

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	dialer.HandshakeTimeout = 5 * time.Second

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("  Error connecting: %v\n", err)
		return false
	}
	defer conn.Close()

	// Send a registration message
	registerMsg := map[string]interface{}{
		"type": "register",
		"data": map[string]interface{}{
			"domain":           "test.example.com",
			"protocol_version": "1.0",
		},
	}

	if err := conn.WriteJSON(registerMsg); err != nil {
		fmt.Printf("  Error sending registration: %v\n", err)
		return false
	}

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var response map[string]interface{}
	if err := conn.ReadJSON(&response); err != nil {
		fmt.Printf("  Error reading response: %v\n", err)
		return false
	}

	msgType, ok := response["type"].(string)
	if !ok {
		fmt.Printf("  Invalid response format\n")
		return false
	}

	fmt.Printf("  Received response type: %s\n", msgType)

	// We expect either "registered" (success) or "error" (domain conflict/other)
	if msgType == "registered" {
		fmt.Printf("  ✓ Registration successful\n")
		return true
	} else if msgType == "error" {
		fmt.Printf("  ⚠ Registration failed (expected if domain is in use)\n")
		return true // Still means the protocol is working
	}

	return false
}
