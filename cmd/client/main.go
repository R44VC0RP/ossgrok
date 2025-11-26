package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/R44VC0RP/ossgrok/internal/client/config"
	"github.com/R44VC0RP/ossgrok/internal/client/wsclient"
	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

func main() {
	// Set log level
	logger.SetLevel("info")

	// Check for subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "config":
		handleConfig()
	case "--url":
		handleTunnel()
	default:
		// If first arg starts with a number, treat as port (backward compat)
		if _, err := strconv.Atoi(os.Args[1]); err == nil {
			handleTunnelShorthand()
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", subcommand)
			printUsage()
			os.Exit(1)
		}
	}
}

func handleConfig() {
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	server := configCmd.String("server", "", "Server domain (e.g., tunnel.example.com)")

	configCmd.Parse(os.Args[2:])

	if *server == "" {
		fmt.Fprintf(os.Stderr, "Error: --server flag is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: ossgrok config --server DOMAIN\n")
		fmt.Fprintf(os.Stderr, "Example: ossgrok config --server tunnel.example.com\n")
		os.Exit(1)
	}

	// Save configuration
	cfg := &config.Config{
		Server: *server,
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to save config: %v\n", err)
		os.Exit(1)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("Configuration saved to %s\n", configPath)
	fmt.Printf("Server: %s\n", cfg.Server)
	fmt.Printf("WebSocket URL: %s\n", cfg.GetWebSocketURL())
}

func handleTunnel() {
	tunnelCmd := flag.NewFlagSet("tunnel", flag.ExitOnError)
	url := tunnelCmd.String("url", "", "Public domain for the tunnel")

	tunnelCmd.Parse(os.Args[2:])

	if *url == "" {
		fmt.Fprintf(os.Stderr, "Error: --url flag is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: ossgrok --url DOMAIN PORT\n")
		fmt.Fprintf(os.Stderr, "Example: ossgrok --url development.exon.dev 3000\n")
		os.Exit(1)
	}

	// Get port from remaining args
	args := tunnelCmd.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: PORT argument is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: ossgrok --url DOMAIN PORT\n")
		fmt.Fprintf(os.Stderr, "Example: ossgrok --url development.exon.dev 3000\n")
		os.Exit(1)
	}

	port, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid port number: %s\n", args[0])
		os.Exit(1)
	}

	startTunnel(*url, port)
}

func handleTunnelShorthand() {
	// Parse: ossgrok 3000 (assumes --url flag before)
	fmt.Fprintf(os.Stderr, "Error: Invalid usage\n\n")
	fmt.Fprintf(os.Stderr, "Usage: ossgrok --url DOMAIN PORT\n")
	fmt.Fprintf(os.Stderr, "Example: ossgrok --url development.exon.dev 3000\n")
	os.Exit(1)
}

func startTunnel(domain string, port int) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create WebSocket client
	client := wsclient.New(cfg.GetWebSocketURL(), domain, port)

	// Connect to server
	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect: %v\n", err)
		os.Exit(1)
	}

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start client in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := client.Run(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt or error
	select {
	case <-sigChan:
		logger.Info("\nReceived interrupt signal, shutting down...")
	case err := <-errChan:
		logger.Error("Client error: %v", err)
	}

	// Close client
	client.Close()
	logger.Info("Tunnel closed")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "ossgrok - Self-hosted tunneling service\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  ossgrok config --server DOMAIN    Configure server settings\n")
	fmt.Fprintf(os.Stderr, "  ossgrok --url DOMAIN PORT         Create HTTP tunnel\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  ossgrok config --server tunnel.example.com\n")
	fmt.Fprintf(os.Stderr, "  ossgrok --url development.exon.dev 3000\n")
}
