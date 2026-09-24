# LiveKit 编译脚本 (Windows PowerShell)
# 解决路径空格和引号问题

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Building LiveKit Server" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# 设置工作目录
$workspace = "e:\workspace\livekit"
Set-Location $workspace

Write-Host "Working directory: $workspace" -ForegroundColor Yellow
Write-Host ""

# 检查 go 命令是否可用
try {
    $goVersion = go version
    Write-Host "Go version: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go not found in PATH" -ForegroundColor Red
    Write-Host "Please install Go: https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# 编译 LiveKit Server
Write-Host "Building LiveKit Server..." -ForegroundColor Cyan
$output = ".\livekit-server.exe"

try {
    go build -o $output ./cmd/server
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build successful!" -ForegroundColor Green
        Write-Host "Output: $output" -ForegroundColor Yellow

        # 显示文件信息
        $fileInfo = Get-Item $output
        Write-Host ""
        Write-Host "File details:" -ForegroundColor Cyan
        Write-Host "  Size: $($fileInfo.Length / 1MB) MB" -ForegroundColor White
        Write-Host "  Created: $($fileInfo.CreationTime)" -ForegroundColor White
        Write-Host "  Location: $($fileInfo.FullName)" -ForegroundColor White
    } else {
        Write-Host "❌ Build failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ Build failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Build Complete" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
