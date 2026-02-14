#!/usr/bin/env bash
set -euo pipefail

# Runs automated integration tests for demo wallet HTTP endpoints.
# Requires local PostgreSQL with the project schema and migrations applied.

echo "[wallet-e2e] running wallet endpoint integration tests..."
export DATABASE_URL="${DATABASE_URL:-postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable}"
echo "[wallet-e2e] DATABASE_URL=$DATABASE_URL"
go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v
