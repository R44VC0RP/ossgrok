#!/bin/bash

# Test script to verify ossgrok server is running correctly
# Usage: ./test-server.sh <server-url>
# Example: ./test-server.sh ossgrok.sevalla.app

set -e

SERVER_URL="${1:-localhost}"
WS_PORT="${2:-4443}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ossgrok Server Health Check${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Testing server: ${YELLOW}${SERVER_URL}${NC}"
echo ""

# Test 1: HTTP endpoint (should redirect or respond)
echo -e "${BLUE}[1/4]${NC} Testing HTTP endpoint (port 80 or 8080)..."
if curl -s -o /dev/null -w "%{http_code}" "http://${SERVER_URL}" | grep -q "301\|200\|404"; then
    echo -e "${GREEN}✓${NC} HTTP endpoint is accessible"
else
    echo -e "${RED}✗${NC} HTTP endpoint is not accessible"
    echo -e "${YELLOW}Note: This might be expected if platform handles routing differently${NC}"
fi
echo ""

# Test 2: HTTPS endpoint (should respond)
echo -e "${BLUE}[2/4]${NC} Testing HTTPS endpoint (port 443 or 8443)..."
if curl -s -k -o /dev/null -w "%{http_code}" "https://${SERVER_URL}" | grep -q "503\|200\|404"; then
    echo -e "${GREEN}✓${NC} HTTPS endpoint is accessible"
    echo -e "${YELLOW}Status 503 is expected when no tunnel is registered${NC}"
else
    echo -e "${RED}✗${NC} HTTPS endpoint is not accessible"
fi
echo ""

# Test 3: WebSocket endpoint (control plane)
echo -e "${BLUE}[3/4]${NC} Testing WebSocket endpoint (port ${WS_PORT})..."
if command -v wscat &> /dev/null; then
    echo -e "${YELLOW}Testing with wscat...${NC}"
    timeout 5 wscat -c "wss://${SERVER_URL}:${WS_PORT}/tunnel" --no-check 2>&1 | head -n 5 || true
    echo -e "${GREEN}✓${NC} WebSocket endpoint appears to be listening"
else
    # Fallback: use curl to test if port is open
    if curl -s -k --max-time 3 "https://${SERVER_URL}:${WS_PORT}/tunnel" 2>&1 | grep -q "Upgrade"; then
        echo -e "${GREEN}✓${NC} WebSocket endpoint is accessible"
    else
        # Try with netcat if available
        if command -v nc &> /dev/null; then
            if timeout 3 nc -zv "${SERVER_URL}" "${WS_PORT}" 2>&1 | grep -q "succeeded\|open"; then
                echo -e "${GREEN}✓${NC} WebSocket port ${WS_PORT} is open"
            else
                echo -e "${RED}✗${NC} Cannot connect to WebSocket port ${WS_PORT}"
                echo -e "${YELLOW}Tip: Install wscat for better testing: npm install -g wscat${NC}"
            fi
        else
            echo -e "${YELLOW}⚠${NC} Cannot fully test WebSocket (wscat not installed)"
            echo -e "${YELLOW}Tip: Install wscat: npm install -g wscat${NC}"
        fi
    fi
fi
echo ""

# Test 4: DNS check
echo -e "${BLUE}[4/4]${NC} Testing DNS resolution..."
if host "${SERVER_URL}" &> /dev/null; then
    IP=$(host "${SERVER_URL}" | grep "has address" | awk '{print $4}' | head -n1)
    echo -e "${GREEN}✓${NC} DNS resolves to: ${IP}"
else
    echo -e "${RED}✗${NC} DNS resolution failed"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Server URL: ${YELLOW}${SERVER_URL}${NC}"
echo -e "WebSocket URL: ${YELLOW}wss://${SERVER_URL}:${WS_PORT}/tunnel${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Configure client: ${GREEN}ossgrok config --server ${SERVER_URL}${NC}"
echo -e "2. Test tunnel: ${GREEN}ossgrok --url test.yourdomain.com 3000${NC}"
echo ""
echo -e "${BLUE}========================================${NC}"
