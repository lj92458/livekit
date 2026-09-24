#!/bin/bash

# LiveKit Server with Cloudflare TURN - Quick Start Script
#
# This script helps you quickly start LiveKit with Cloudflare TURN enabled.

set -e

echo "=========================================="
echo "LiveKit + Cloudflare TURN Quick Start"
echo "=========================================="
echo ""

# Check if required environment variables are set
if [ -z "$CF_TURN_KEY_ID" ]; then
    echo "❌ ERROR: CF_TURN_KEY_ID environment variable is not set"
    echo ""
    echo "Please set the following environment variables:"
    echo "  export CF_TURN_KEY_ID='your-cloudflare-turn-key-id'"
    echo "  export CF_TURN_API_TOKEN='your-cloudflare-api-token'"
    echo ""
    echo "Or create a .env file from the example:"
    echo "  cp .env.cloudflare-turn.example .env"
    echo "  # Edit .env with your credentials"
    echo "  source .env"
    echo ""
    exit 1
fi

if [ -z "$CF_TURN_API_TOKEN" ]; then
    echo "❌ ERROR: CF_TURN_API_TOKEN environment variable is not set"
    echo ""
    echo "Please set the following environment variables:"
    echo "  export CF_TURN_KEY_ID='your-cloudflare-turn-key-id'"
    echo "  export CF_TURN_API_TOKEN='your-cloudflare-api-token'"
    echo ""
    exit 1
fi

# Set defaults
LIVEKIT_KEYS=${LIVEKIT_KEYS:-"devkey:secret"}
LIVEKIT_REGION=${LIVEKIT_REGION:-"us-west-1"}
CF_TURN_TTL=${CF_TURN_TTL:-"86400"}

echo "✅ Environment variables:"
echo "   CF_TURN_KEY_ID: ${CF_TURN_KEY_ID:0:10}..."
echo "   CF_TURN_API_TOKEN: ${CF_TURN_API_TOKEN:0:10}..."
echo "   CF_TURN_TTL: ${CF_TURN_TTL}s"
echo "   LIVEKIT_KEYS: ${LIVEKIT_KEYS}"
echo "   LIVEKIT_REGION: ${LIVEKIT_REGION}"
echo ""

# Check if Redis is running
if ! redis-cli ping > /dev/null 2>&1; then
    echo "⚠️  WARNING: Redis is not running"
    echo ""
    echo "You can start Redis with Docker:"
    echo "  docker run -d -p 6379:6379 redis:7-alpine"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Start LiveKit
echo "🚀 Starting LiveKit Server with Cloudflare TURN..."
echo ""

# Build the server (if not already built)
if [ ! -f "./livekit-server" ]; then
    echo "📦 Building LiveKit Server..."
    go build -o livekit-server ./cmd/server
fi

# Start the server
./livekit-server --dev --keys="$LIVEKIT_KEYS" --region="$LIVEKIT_REGION"

echo ""
echo "=========================================="
echo "LiveKit Server Stopped"
echo "=========================================="
