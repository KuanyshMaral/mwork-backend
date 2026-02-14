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
