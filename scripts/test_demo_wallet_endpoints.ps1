$ErrorActionPreference = 'Stop'

if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = 'postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable'
}

Write-Host "[wallet-e2e] running wallet endpoint integration tests..." -ForegroundColor Cyan
Write-Host "[wallet-e2e] DATABASE_URL=$env:DATABASE_URL" -ForegroundColor DarkCyan

$portCheck = Test-NetConnection -ComputerName localhost -Port 5432 -WarningAction SilentlyContinue
if (-not $portCheck.TcpTestSucceeded) {
  Write-Host "[wallet-e2e] ERROR: localhost:5432 is not reachable." -ForegroundColor Red
  Write-Host "Run: docker compose up -d postgres" -ForegroundColor Yellow
  Write-Host "Then: docker compose run --rm migrate" -ForegroundColor Yellow
  exit 1
}

go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v
