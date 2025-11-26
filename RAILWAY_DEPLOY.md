# Deploy ossgrok to Railway

Railway is the recommended platform for deploying ossgrok due to its excellent support for custom ports and straightforward configuration.

## Prerequisites

- Railway account (https://railway.app)
- Your GitHub repository connected to Railway
- A domain name (optional, but recommended)

## Deployment Steps

### 1. Create New Project

1. Go to https://railway.app
2. Click "New Project"
3. Select "Deploy from GitHub repo"
4. Choose `ossgrok` repository

### 2. Configure Build Settings

Railway should auto-detect the Dockerfile. If not:

1. Go to **Settings** → **Build**
2. Set:
   - **Builder**: Dockerfile
   - **Dockerfile Path**: `deployments/docker/Dockerfile.server`
   - **Docker Context**: `.` (root directory)

### 3. Set Environment Variables

Go to **Variables** tab and add:

```
AUTOCERT_DOMAINS=yourdomain.com,*.yourdomain.com
AUTOCERT_EMAIL=your-email@example.com
SERVER_HTTP_PORT=8080
SERVER_HTTPS_PORT=8443
SERVER_WS_PORT=4443
LOG_LEVEL=info
```

**Important:** Replace `yourdomain.com` with your actual domain!

### 4. Expose Ports

Railway will automatically expose port 8080 as the public HTTP port. However, we need multiple ports:

1. Go to **Settings** → **Networking**
2. You should see Railway has assigned a domain like `ossgrok-production.up.railway.app`
3. Railway will handle port mapping automatically

**Note:** Railway maps:
- External port 80/443 → Internal port 8080/8443
- External port 4443 → Internal port 4443

### 5. Configure DNS

#### Option A: Use Railway's Domain (Easiest)

Use Railway's provided domain (e.g., `ossgrok-production.up.railway.app`):

```bash
ossgrok config --server ossgrok-production.up.railway.app
```

#### Option B: Use Custom Domain

1. In Railway, go to **Settings** → **Networking** → **Custom Domain**
2. Add your domain (e.g., `tunnel.yourdomain.com`)
3. Railway will provide DNS instructions (usually a CNAME record)
4. Update your DNS:
   ```
   tunnel.yourdomain.com.  CNAME  ossgrok-production.up.railway.app.
   ```
5. Wait for DNS propagation (can take up to 24 hours, usually minutes)

### 6. Configure Wildcard Subdomains (Optional but Recommended)

If you want to support dynamic subdomains (e.g., `dev.yourdomain.com`, `staging.yourdomain.com`):

1. In `AUTOCERT_DOMAINS`, use: `yourdomain.com,*.yourdomain.com`
2. Add wildcard DNS record:
   ```
   *.yourdomain.com.  CNAME  tunnel.yourdomain.com.
   ```

   Or if using Railway's domain directly:
   ```
   *.yourdomain.com.  CNAME  ossgrok-production.up.railway.app.
   ```

### 7. Deploy

1. Click **Deploy** or push to GitHub (auto-deploys)
2. Watch the build logs
3. Wait for "Deployment successful"

### 8. Test Your Deployment

```bash
# Quick test
./test-server.sh ossgrok-production.up.railway.app

# Comprehensive test
go run cmd/test/main.go ossgrok-production.up.railway.app
```

Expected output:
```
✓ HTTP endpoint is accessible
✓ HTTPS endpoint is accessible
✓ WebSocket endpoint is accessible and accepting connections
✓ WebSocket protocol is working correctly

Passed: 4/4 tests
```

### 9. Configure Client

```bash
# Configure (one-time)
ossgrok config --server ossgrok-production.up.railway.app

# Create a tunnel
ossgrok --url dev.yourdomain.com 3000
```

## Troubleshooting

### Build Fails

- Check **Build Logs** in Railway dashboard
- Verify Dockerfile path is correct
- Ensure all environment variables are set

### Can't Connect to WebSocket (Port 4443)

Railway might not expose port 4443 by default. Two solutions:

**Solution 1: Use Railway's Port Mapping**
Railway should automatically map it, but you can verify in Settings → Networking

**Solution 2: Use a TCP Proxy Service**
If Railway doesn't support port 4443, you may need to:
- Use a different platform (Fly.io, DigitalOcean App Platform)
- Deploy to a VPS with Docker

### Certificate Issues

- Ensure DNS is pointing to Railway before deployment
- Check `AUTOCERT_DOMAINS` matches your DNS records
- Let's Encrypt needs port 80 accessible for HTTP-01 challenge

### 503 Service Unavailable

This is **expected** when no tunnel is registered! It means the server is running correctly.

## Cost

Railway offers:
- **Free tier**: $5/month of usage included
- **Hobby plan**: $5/month for moderate usage
- This application should fit comfortably in the free tier for personal use

## Alternative: Deploy to Fly.io

If Railway doesn't work for your needs, Fly.io is another excellent option that supports multiple ports:

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Launch app
fly launch --dockerfile deployments/docker/Dockerfile.server

# Set environment variables
fly secrets set AUTOCERT_DOMAINS=yourdomain.com,*.yourdomain.com
fly secrets set AUTOCERT_EMAIL=your-email@example.com
fly secrets set SERVER_HTTP_PORT=8080
fly secrets set SERVER_HTTPS_PORT=8443
fly secrets set SERVER_WS_PORT=4443

# Deploy
fly deploy
```

Fly.io automatically handles multiple ports and is very developer-friendly.
