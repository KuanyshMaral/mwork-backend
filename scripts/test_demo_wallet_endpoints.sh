#!/usr/bin/env bash
set -euo pipefail

# Runs automated integration tests for demo wallet HTTP endpoints.
# Requires local PostgreSQL with the project schema and migrations applied.

echo "[wallet-e2e] running wallet endpoint integration tests..."
go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v
