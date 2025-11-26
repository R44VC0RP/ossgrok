# ossgrok

A self-hostable ngrok alternative for HTTP/HTTPS tunneling. Expose your local development server to the internet through your own domain.

## Features

- **HTTP/HTTPS Tunneling**: Expose local web applications to the internet
- **Custom Domains**: Use your own domains with automatic HTTPS via Let's Encrypt
- **Simple CLI**: Easy-to-use command-line interface
- **Docker Ready**: Deploy server with Docker in minutes
- **Secure**: WebSocket-based control plane with TLS encryption
- **Lightweight**: Built with Go for performance and low resource usage

## Quick Start

### 1. Deploy the Server

**Recommended: Deploy to Railway (Easiest)**

See [RAILWAY_DEPLOY.md](RAILWAY_DEPLOY.md) for detailed Railway deployment instructions.

Quick steps:
1. Push this repo to GitHub
2. Create new Railway project from GitHub repo
3. Set environment variables (see Railway guide)
4. Deploy and get your URL (e.g., `ossgrok-production.up.railway.app`)

**Alternative: Self-hosted VPS**

**Prerequisites:**
- A server with Docker installed
- A domain pointing to your server (e.g., `tunnel.example.com`)

```bash
# Build and run the server
docker build -f deployments/docker/Dockerfile.server -t ossgrok-server .

docker run -d \
  -p 80:80 \
  -p 443:443 \
  -p 4443:4443 \
  -e AUTOCERT_DOMAINS=tunnel.example.com \
  -e AUTOCERT_EMAIL=admin@example.com \
  -v ossgrok-autocert:/var/lib/autocert \
  --name ossgrok-server \
  ossgrok-server
```

**Important:** Ensure your domain's DNS is configured to point to your server before starting. Let's Encrypt requires this for certificate validation.

### 2. Use the Client

**Option A: Docker**

```bash
# Build client image
docker build -f deployments/docker/Dockerfile.client -t ossgrok-client .

# Configure server (one-time)
docker run --rm \
  -v ossgrok-config:/root/.ossgrok \
  ossgrok-client \
  config --server tunnel.example.com

# Create tunnel
docker run --rm \
  --network host \
  -v ossgrok-config:/root/.ossgrok \
  ossgrok-client \
  --url development.exon.dev 3000
```

**Option B: Build from Source**

```bash
# Build client
go build -o ossgrok ./cmd/client

# Configure server (one-time)
./ossgrok config --server tunnel.example.com

# Create tunnel
./ossgrok --url development.exon.dev 3000
```

## Usage

### Configure Server

Run this once to configure the client:

```bash
ossgrok config --server tunnel.example.com
```

This saves the server URL to `~/.ossgrok/config.json`.

### Create a Tunnel

```bash
ossgrok --url DOMAIN PORT
```

**Example:**

```bash
ossgrok --url development.exon.dev 3000
```

This creates a tunnel from `https://development.exon.dev` to `http://localhost:3000`.

### DNS Configuration

For each domain you want to tunnel, create a CNAME record pointing to your server:

```
development.exon.dev.  CNAME  tunnel.example.com.
```

## Architecture

```
Internet → Server (80/443) → Domain Router → Tunnel Registry
                                                    ↓
                                              WebSocket (4443)
                                                    ↓
                                            CLI Client (local)
                                                    ↓
                                            Local App (e.g., :3000)
```

### How It Works

1. Client connects to server via WebSocket and registers a domain
2. Server stores domain → WebSocket connection mapping
3. HTTP request arrives at server on port 80/443
4. Server extracts Host header, looks up tunnel in registry
5. Server sends HTTP request over WebSocket to client
6. Client forwards to local application
7. Client sends response back via WebSocket
8. Server returns response to original HTTP caller

## Server Configuration

### Environment Variables

- `SERVER_HTTP_PORT` (default: `8080`) - HTTP port for ACME challenges (use 8080 for PaaS platforms, 80 for VPS)
- `SERVER_HTTPS_PORT` (default: `8443`) - HTTPS port for tunnel traffic (use 8443 for PaaS platforms, 443 for VPS)
- `SERVER_WS_PORT` (default: `4443`) - WebSocket port for control plane
- `AUTOCERT_DOMAINS` (required) - Comma-separated list of allowed domains
- `AUTOCERT_EMAIL` (optional) - Email for Let's Encrypt notifications
- `AUTOCERT_CACHE_DIR` (default: `/var/lib/autocert`) - Certificate cache directory
- `LOG_LEVEL` (default: `info`) - Log level (debug/info/warn/error)

### Example Docker Run (VPS with root access)

```bash
docker run -d \
  -p 80:8080 \
  -p 443:8443 \
  -p 4443:4443 \
  -e AUTOCERT_DOMAINS=tunnel.example.com \
  -e AUTOCERT_EMAIL=admin@example.com \
  -e LOG_LEVEL=debug \
  -v ossgrok-autocert:/var/lib/autocert \
  --name ossgrok-server \
  ossgrok-server
```

### Deploying to PaaS (Railway, Sevalla, Render, etc.)

When deploying to platforms like Railway, Sevalla, or Render, the platform handles SSL/TLS termination and routing. Configure:

**Environment Variables:**
```
AUTOCERT_DOMAINS=yourdomain.com,*.yourdomain.com
AUTOCERT_EMAIL=your-email@example.com
SERVER_HTTP_PORT=8080
SERVER_HTTPS_PORT=8443
SERVER_WS_PORT=4443
```

**Important Notes:**
- The platform's reverse proxy will handle ports 80/443 and forward to your app's high ports (8080/8443)
- Let's Encrypt will work through the platform's HTTP-01 challenge on port 8080
- Point your DNS to the platform's provided domain
- For Sevalla: Use the "Expose HTTP traffic" setting and configure port 8080

## Development

### Build from Source

```bash
# Install dependencies
go mod download

# Build server
go build -o ossgrok-server ./cmd/server

# Build client
go build -o ossgrok ./cmd/client
```

### Run Locally

**Server:**

```bash
export AUTOCERT_DOMAINS=localhost
export LOG_LEVEL=debug
go run ./cmd/server
```

**Client:**

```bash
go run ./cmd/client config --server localhost
go run ./cmd/client --url test.local 3000
```

## Project Structure

```
ossgrok/
├── cmd/
│   ├── server/          # Server entry point
│   └── client/          # Client entry point
├── internal/
│   ├── protocol/        # WebSocket message protocol
│   ├── server/          # Server components
│   │   ├── registry/    # Tunnel registry
│   │   ├── httphandler/ # HTTP request handler
│   │   ├── wsmanager/   # WebSocket manager
│   │   └── tunnel/      # Tunnel connection
│   └── client/          # Client components
│       ├── config/      # Config management
│       ├── wsclient/    # WebSocket client
│       └── proxy/       # HTTP proxy
├── pkg/
│   └── logger/          # Logging utility
└── deployments/
    └── docker/          # Docker configurations
```

## Security Considerations

- **TLS Encryption**: All traffic uses HTTPS/WSS with Let's Encrypt certificates
- **No Authentication**: Current implementation has no authentication (suitable for private networks)
- **Port Access**: Ensure ports 80, 443, and 4443 are properly firewalled

## Testing Your Server

Before using the client, verify your server is running correctly:

### Quick Test (Bash)

```bash
./test-server.sh ossgrok.sevalla.app
```

### Comprehensive Test (Go)

```bash
go run cmd/test/main.go ossgrok.sevalla.app
```

This will test:
- HTTP endpoint accessibility
- HTTPS endpoint with TLS
- WebSocket control plane connectivity
- WebSocket protocol (registration)

Expected output when server is healthy:
```
✓ HTTP endpoint is accessible
✓ HTTPS endpoint is accessible
✓ WebSocket endpoint is accessible and accepting connections
✓ WebSocket protocol is working correctly

Passed: 4/4 tests
```

## Troubleshooting

### Certificate Issues

If Let's Encrypt fails to issue certificates:
- Ensure DNS is properly configured
- Check that ports 80 and 443 are accessible
- Verify `AUTOCERT_DOMAINS` matches your DNS records

### Connection Issues

If the client can't connect:
- Verify server is running: `docker ps`
- Check server logs: `docker logs ossgrok-server`
- Ensure port 4443 is accessible
- Verify server domain in config matches deployed server

### Tunnel Not Working

If HTTP requests aren't reaching your local app:
- Check client is connected and shows "Tunnel is active"
- Verify DNS CNAME record is configured correctly
- Check local application is running on specified port
- Review server logs for errors

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
