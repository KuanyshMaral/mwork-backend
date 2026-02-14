$ErrorActionPreference = 'Stop'

Write-Host "[wallet-e2e] running wallet endpoint integration tests..." -ForegroundColor Cyan
go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v

if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = 'postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable'
}
Write-Host "[wallet-e2e] DATABASE_URL=$env:DATABASE_URL" -ForegroundColor DarkCyan
