# Demo Wallet Usage Flow

`PAYMENT_MODE=demo` enables internal virtual balance charging for paid features through `DemoPaymentProvider`.

## Typical flow

1. User tops up demo balance:
   - `POST /api/v1/demo/wallet/topup`
2. A paid feature attempts charge through `PaymentProvider.Charge(...)`.
   - In `demo` mode this delegates to wallet spend.
3. If feature operation fails after charge, feature may issue refund:
   - `POST /api/v1/demo/wallet/refund`
4. User checks balance:
   - `GET /api/v1/demo/wallet/balance`

## Sample curl commands

```bash
curl -X POST http://localhost:8080/api/v1/demo/wallet/topup \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1000, "reference_id": "topup_seed_001"}'
```

```bash
curl -X POST http://localhost:8080/api/v1/demo/wallet/spend \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"amount": 250, "reference_id": "feature_purchase_123"}'
```

```bash
curl -X POST http://localhost:8080/api/v1/demo/wallet/refund \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"amount": 250, "reference_id": "feature_purchase_123_refund"}'
```

```bash
curl http://localhost:8080/api/v1/demo/wallet/balance \
  -H "Authorization: Bearer <JWT>"
```


## Automated endpoint test (recommended)

This repository now includes an integration test that calls all demo wallet endpoints with JWT auth:

- `GET /api/v1/demo/wallet/balance`
- `POST /api/v1/demo/wallet/topup`
- `POST /api/v1/demo/wallet/spend`
- `POST /api/v1/demo/wallet/refund`

It also verifies:
- idempotent retry (`same reference_id + same amount`),
- conflict (`same reference_id + different amount`),
- unauthorized access without JWT.

### Setup

1. Start PostgreSQL (Docker) and wait until it is healthy:
   ```bash
   docker compose up -d postgres
   docker compose ps
   ```

2. Apply migrations (without requiring local `make`/`migrate` install):
   ```bash
   docker compose run --rm migrate
   ```

3. Run endpoint integration test:

   **Linux/macOS (bash):**
   ```bash
   ./scripts/test_demo_wallet_endpoints.sh
   ```

   **Windows PowerShell:**
   ```powershell
   .\scripts\test_demo_wallet_endpoints.ps1
   ```


4. Verify DB credentials from host (important):

   ```powershell
   docker compose exec postgres psql -U mwork -d mwork_dev -c "SELECT current_user, current_database();"
   ```

   If this command fails with role/db errors, reset volume and recreate:

   ```powershell
   docker compose down -v
   docker compose up -d postgres
   docker compose run --rm migrate
   ```

> Note: `docker-compose.yml` publishes PostgreSQL on host port `5432`, so local `go test` from host can connect to `localhost:5432`.

### Direct go test alternative

```bash
go test ./internal/domain/wallet -run TestWalletEndpointsIntegration -v
```


### Troubleshooting (Windows / Docker)

If you get an error like `pq: role "mwork" does not exist`, your existing Docker volume was initialized earlier with different credentials.

Reset Postgres volume and recreate DB with the current credentials from `docker-compose.yml`:

```powershell
docker compose down -v
docker compose up -d postgres
docker compose run --rm migrate
```

Then run tests again:

```powershell
.\scripts\test_demo_wallet_endpoints.ps1
```

You can also override connection explicitly before running tests:

```powershell
$env:DATABASE_URL = "postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable"
```

