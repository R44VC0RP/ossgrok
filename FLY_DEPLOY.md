# Deploy ossgrok to Fly.io

Fly.io is the **recommended** platform for deploying ossgrok due to its excellent support for multiple ports and TCP services.

## Prerequisites

- Fly.io account (https://fly.io)
- Fly CLI installed
- A domain name (optional, but recommended)

## Installation

### Install Fly CLI

**macOS/Linux:**
```bash
curl -L https://fly.io/install.sh | sh
```

**Windows (PowerShell):**
```powershell
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

### Login to Fly.io

```bash
fly auth login
```

This will open a browser window for authentication.

## Deployment Steps

### 1. Initialize Fly App (Optional - Already Configured)

The `fly.toml` file is already configured. But if you need to customize:

```bash
fly launch --no-deploy
```

This will ask you:
- **App name**: `ossgrok` (or choose your own)
- **Region**: Choose closest to you (e.g., `iad` for US East)
- **Database**: Skip (not needed)

### 2. Create Volume for Autocert Cache

```bash
fly volumes create autocert_cache --region iad --size 1
```

Replace `iad` with your chosen region.

### 3. Set Environment Variables (Secrets)

```bash
fly secrets set AUTOCERT_DOMAINS=ehook.dev,*.ehook.dev
fly secrets set AUTOCERT_EMAIL=your-email@example.com
```

**Important:** Replace with your actual domain and email!

### 4. Deploy

```bash
fly deploy
```

This will:
- Build the Docker image
- Deploy to Fly.io
- Expose ports 80, 443, 8443, and 4443
- Assign you a domain like `ossgrok.fly.dev`

### 5. Verify Deployment

Check status:
```bash
fly status
```

View logs:
```bash
fly logs
```

You should see:
```
[INFO] Starting ossgrok server...
[INFO] Configured domains: [ehook.dev *.ehook.dev]
[INFO] ossgrok server is running!
```

### 6. Get Your Fly.io URL

```bash
fly info
```

Look for the hostname (e.g., `ossgrok.fly.dev`)

### 7. Test Deployment

```bash
# Quick test
./test-server.sh ossgrok.fly.dev

# Comprehensive test
go run cmd/test/main.go ossgrok.fly.dev
```

Expected output:
```
✓ HTTP endpoint is accessible
✓ HTTPS endpoint is accessible
✓ WebSocket endpoint is accessible and accepting connections
✓ WebSocket protocol is working correctly

Passed: 4/4 tests
```

### 8. Configure DNS

#### Using Fly.io's Domain (Easiest)

```bash
ossgrok config --server ossgrok.fly.dev
ossgrok --url dev.ehook.dev 3000
```

Then point your DNS:
```
dev.ehook.dev.  CNAME  ossgrok.fly.dev.
*.ehook.dev.    CNAME  ossgrok.fly.dev.
```

#### Using Custom Domain

1. **Add custom domain to Fly:**
   ```bash
   fly certs add tunnel.ehook.dev
   ```

2. **Follow DNS instructions** (Fly will show you what records to add)

3. **Verify certificate:**
   ```bash
   fly certs show tunnel.ehook.dev
   ```

4. **Update wildcard DNS:**
   ```
   *.ehook.dev.  CNAME  tunnel.ehook.dev.
   ```

### 9. Configure Client

```bash
# Configure (one-time)
ossgrok config --server ossgrok.fly.dev

# Create a tunnel
ossgrok --url dev.ehook.dev 3000
```

## Configuration Details

### Port Mapping

The `fly.toml` configures three services:

1. **HTTP Service (Ports 80/443)**
   - External: 80 (HTTP) → Internal: 8080
   - External: 443 (HTTPS) → Internal: 8080
   - For ACME challenges and HTTP redirects

2. **HTTPS Tunnel Traffic (Port 8443)**
   - External: 8443 → Internal: 8443
   - For actual tunnel HTTPS traffic

3. **WebSocket Control Plane (Port 4443)**
   - External: 4443 → Internal: 4443
   - For tunnel registration and control messages

### Environment Variables

Set via `fly secrets`:
- `AUTOCERT_DOMAINS` - Your domain(s), comma-separated
- `AUTOCERT_EMAIL` - Email for Let's Encrypt notifications

Set in `fly.toml`:
- `SERVER_HTTP_PORT=8080`
- `SERVER_HTTPS_PORT=8443`
- `SERVER_WS_PORT=4443`
- `LOG_LEVEL=info`

### Persistent Storage

The `/var/lib/autocert` directory is mounted from a Fly volume to persist Let's Encrypt certificates between deployments.

## Scaling & Performance

### Scale Instances

```bash
# Add more instances
fly scale count 2

# Scale to specific regions
fly regions add lax sea
```

### Update Resources

```bash
# Scale VM size
fly scale vm shared-cpu-1x

# Scale memory
fly scale memory 512
```

## Updating

To deploy updates:

```bash
git pull
fly deploy
```

Fly.io will:
- Build new Docker image
- Deploy with zero downtime
- Keep certificates and volumes intact

## Monitoring

### View Logs

```bash
# Stream logs
fly logs

# Recent logs
fly logs --limit 200
```

### Check Status

```bash
fly status
fly info
```

### SSH into Instance

```bash
fly ssh console
```

## Troubleshooting

### Deployment Fails

Check build logs:
```bash
fly logs --limit 500
```

Common issues:
- Incorrect Dockerfile path → Check `fly.toml` build section
- Missing secrets → Run `fly secrets list`

### Can't Connect to Port 4443

Verify the service is exposed:
```bash
fly status
```

Should show services on ports 80, 443, 8443, and 4443.

### Certificate Issues

Check certificate status:
```bash
fly certs show yourdomain.com
```

Verify DNS is pointing to Fly:
```bash
dig yourdomain.com
```

### AUTOCERT Not Working

Ensure:
1. DNS points to Fly.io domain
2. Port 80 is accessible (for ACME challenge)
3. Domain is in `AUTOCERT_DOMAINS` secret
4. Wait up to 10 minutes for certificate issuance

Check logs:
```bash
fly logs | grep -i autocert
```

## Cost

Fly.io pricing:
- **Free tier**: $5/month credit (enough for hobby use)
- **Shared CPU**: ~$3-5/month
- **Volume storage**: $0.15/GB/month (1GB = $0.15)

Total estimated cost for personal use: **$0-3/month**

## Alternative Regions

Available regions:
```bash
fly platform regions
```

Change region:
```bash
fly regions set iad lax  # US East + West
```

## Cleanup

To destroy the app:
```bash
fly apps destroy ossgrok
```

To destroy volume:
```bash
fly volumes destroy autocert_cache
```
