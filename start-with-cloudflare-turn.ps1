# LiveKit Server with Cloudflare TURN - Quick Start Script (PowerShell)
#
# This script helps you quickly start LiveKit with Cloudflare TURN enabled.

$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "LiveKit + Cloudflare TURN Quick Start" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Check if required environment variables are set
$cfTurnKeyId = $env:CF_TURN_KEY_ID
$cfTurnApiToken = $env:CF_TURN_API_TOKEN

if ([string]::IsNullOrEmpty($cfTurnKeyId)) {
    Write-Host "❌ ERROR: CF_TURN_KEY_ID environment variable is not set" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please set the following environment variables:" -ForegroundColor Yellow
    Write-Host '  $env:CF_TURN_KEY_ID = "your-cloudflare-turn-key-id"' -ForegroundColor White
    Write-Host '  $env:CF_TURN_API_TOKEN = "your-cloudflare-api-token"' -ForegroundColor White
    Write-Host ""
    Write-Host "Or create a .env file from the example:" -ForegroundColor Yellow
    Write-Host "  cp .env.cloudflare-turn.example .env" -ForegroundColor White
    Write-Host "  # Edit .env with your credentials" -ForegroundColor White
    Write-Host "  # Then run: Get-Content .env | ForEach-Object { \$var = \$_ -split '='; [Environment]::SetEnvironmentVariable(\$var[0], \$var[1]) }" -ForegroundColor White
    Write-Host ""
    exit 1
}

if ([string]::IsNullOrEmpty($cfTurnApiToken)) {
    Write-Host "❌ ERROR: CF_TURN_API_TOKEN environment variable is not set" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please set the following environment variables:" -ForegroundColor Yellow
    Write-Host '  $env:CF_TURN_KEY_ID = "your-cloudflare-turn-key-id"' -ForegroundColor White
    Write-Host '  $env:CF_TURN_API_TOKEN = "your-cloudflare-api-token"' -ForegroundColor White
    Write-Host ""
    exit 1
}

# Set defaults
$livekitKeys = if ($env:LIVEKIT_KEYS) { $env:LIVEKIT_KEYS } else { "devkey:secret" }
$livekitRegion = if ($env:LIVEKIT_REGION) { $env:LIVEKIT_REGION } else { "us-west-1" }
$cfTurnTTL = if ($env:CF_TURN_TTL) { $env:CF_TURN_TTL } else { "86400" }

Write-Host "✅ Environment variables:" -ForegroundColor Green
Write-Host "   CF_TURN_KEY_ID: $($cfTurnKeyId.Substring(0, [Math]::Min(10, $cfTurnKeyId.Length)))..." -ForegroundColor White
Write-Host "   CF_TURN_API_TOKEN: $($cfTurnApiToken.Substring(0, [Math]::Min(10, $cfTurnApiToken.Length)))..." -ForegroundColor White
Write-Host "   CF_TURN_TTL: ${cfTurnTTL}s" -ForegroundColor White
Write-Host "   LIVEKIT_KEYS: ${livekitKeys}" -ForegroundColor White
Write-Host "   LIVEKIT_REGION: ${livekitRegion}" -ForegroundColor White
Write-Host ""

# Check if Redis is running
try {
    $redisResult = & redis-cli ping 2>&1
    if ($LASTEXITCODE -eq 0 -and $redisResult -eq "PONG") {
        Write-Host "✅ Redis is running" -ForegroundColor Green
    } else {
        throw "Redis not responding"
    }
} catch {
    Write-Host "⚠️  WARNING: Redis is not running" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "You can start Redis with Docker:" -ForegroundColor Yellow
    Write-Host "  docker run -d -p 6379:6379 redis:7-alpine" -ForegroundColor White
    Write-Host ""
    $response = Read-Host "Continue anyway? (y/n)"
    if ($response -ne "y" -and $response -ne "Y") {
        exit 1
    }
}

# Start LiveKit
Write-Host "🚀 Starting LiveKit Server with Cloudflare TURN..." -ForegroundColor Cyan
Write-Host ""

# Build the server (if not already built)
if (-not (Test-Path "./livekit-server.exe")) {
    Write-Host "📦 Building LiveKit Server..." -ForegroundColor Cyan
    go build -o livekit-server.exe ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed" -ForegroundColor Red
        exit 1
    }
}

# Start the server
& "./livekit-server.exe" --dev --keys=$livekitKeys --region=$livekitRegion

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "LiveKit Server Stopped" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
