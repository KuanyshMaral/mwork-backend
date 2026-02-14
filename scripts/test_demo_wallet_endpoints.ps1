$ErrorActionPreference = 'Stop'

Write-Host "[wallet-e2e] running wallet endpoint integration tests..." -ForegroundColor Cyan
go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v
