# 🔄 ROBOKASSA MIGRATION ANALYSIS REPORT

## 📊 Executive Summary

**Status:** ✅ Ready for Integration  
**Risk Level:** LOW-MEDIUM  
**Current Progress:** 70% (robokassa-integration folder provided)  
**Estimated Time to Complete:** 3-4 hours  

---

## 📋 CURRENT STATE ANALYSIS

### Files Using Kaspi (Need Replacement)

**Configuration Layer:**
- ✅ `internal/config/config.go` 
  - Contains: `KaspiBaseURL`, `KaspiMerchantID`, `KaspiSecretKey`
  - Status: NEEDS UPDATE - Add RoboKassa config fields

**Payment Domain:**
- ✅ `internal/domain/payment/handler.go` (231 lines)
  - Contains: `HandleKaspiWebhook()` - processes Kaspi webhooks
  - Imports: `"github.com/mwork/mwork-api/internal/pkg/kaspi"`
  - Status: NEEDS REWRITE - Replace webhook handling with provider-agnostic approach

- ✅ `internal/domain/payment/service.go` (TBD)
  - Status: LIKELY NEEDS UPDATE - Payment service methods

- ✅ `internal/domain/payment/repository.go` (TBD)
  - Status: LIKELY NEEDS UPDATE - Database queries for kaspi_order_id

**Subscription Domain:**
- ✅ `internal/domain/subscription/handler.go` (292 lines)
  - Contains: `KaspiClient` interface, `KaspiPaymentRequest`, `KaspiPaymentResponse`
  - Status: NEEDS REWRITE - Refactor to use provider factory pattern

- ✅ `internal/domain/subscription/service.go` (TBD)
  - Status: LIKELY NEEDS UPDATE

**Entry Point:**
- ✅ `cmd/api/main.go` (636 lines)
  - Line 44: Imports `"github.com/mwork/mwork-api/internal/pkg/kaspi"`
  - Lines 168-172: Creates Kaspi client directly
  - Lines 542-560: `subscriptionKaspiAdapter` struct
  - Status: NEEDS MAJOR REFACTOR - Replace with provider factory pattern

**Package Layer:**
- ✅ `internal/pkg/kaspi/` (DEPRECATED)
  - Files: `client.go`, `webhook.go`
  - Status: Can be removed after migration (or kept for backward compatibility)

**Database**
- ✅ `migrations/000023_add_kaspi_order_id.down.sql` (DEPRECATED)
- ✅ `migrations/000033_create_kaspi_events.up/down.sql` (Legacy)
- ✅ `migrations/000048_replace_kaspi_with_robokassa.up/down.sql` (Already exists!)
- ✅ `migrations/000049_create_robokassa_sequence.up/down.sql` (Needs to be applied)

---

## 🎯 ROBOKASSA INTEGRATION FILES (Ready to Use)

From `robokassa-integration/` folder:

### ✅ READY TO COPY:

1. **Provider Abstraction Layer:**
   ```
   internal/pkg/payment/provider.go
   internal/pkg/payment/robokassa_provider.go
   ```

2. **RoboKassa Client:**
   ```
   internal/pkg/robokassa/client.go
   internal/pkg/robokassa/webhook.go
   internal/pkg/robokassa/client_test.go
   internal/pkg/robokassa/webhook_test.go
   ```

3. **Updated Domain Files:**
   ```
   internal/domain/payment/handler_new.go
   internal/domain/payment/service_robokassa.go
   internal/domain/payment/repository_robokassa.go
   internal/domain/subscription/handler_robokassa.go
   ```

4. **Migrations:**
   ```
   migrations/000049_create_robokassa_sequence.up.sql
   migrations/000049_create_robokassa_sequence.down.sql
   ```

5. **Reference Implementation:**
   ```
   cmd/api/main_robokassa_setup.go
   ```

---

## 🔧 DETAILED MIGRATION PLAN

### PHASE 1: Configuration Updates ⏱️ 15 min

#### 1.1 Update `internal/config/config.go`

**ADD these fields to Config struct:**
```go
// RoboKassa Payment
RoboKassaMerchantLogin string
RoboKassaPassword1     string
RoboKassaPassword2     string
RoboKassaTestMode      bool

// Optional: Keep for migration period
FrontendURL string
BackendURL  string
```

**ADD environment variable loading:**
```go
RoboKassaMerchantLogin: getEnv("ROBOKASSA_MERCHANT_LOGIN", ""),
RoboKassaPassword1:     getEnv("ROBOKASSA_PASSWORD1", ""),
RoboKassaPassword2:     getEnv("ROBOKASSA_PASSWORD2", ""),
RoboKassaTestMode:      envBool("ROBOKASSA_TEST_MODE", false),
FrontendURL:            getEnv("FRONTEND_URL", "http://localhost:3000"),
BackendURL:             getEnv("BACKEND_URL", "http://localhost:8080"),
```

**Status:** ⏳ TODO

---

### PHASE 2: Copy New Files ⏱️ 5 min

**From robokassa-integration:**

```bash
# 1. Create directories
mkdir -p internal/pkg/payment
mkdir -p internal/pkg/robokassa

# 2. Copy payment provider abstraction
cp robokassa-integration/internal/pkg/payment/provider.go \
   internal/pkg/payment/provider.go

cp robokassa-integration/internal/pkg/payment/robokassa_provider.go \
   internal/pkg/payment/robokassa_provider.go

# 3. Copy RoboKassa client
cp robokassa-integration/internal/pkg/robokassa/*.go \
   internal/pkg/robokassa/

# 4. Copy updated domain files
cp robokassa-integration/internal/domain/payment/handler_new.go \
   internal/domain/payment/

cp robokassa-integration/internal/domain/payment/service_robokassa.go \
   internal/domain/payment/

cp robokassa-integration/internal/domain/payment/repository_robokassa.go \
   internal/domain/payment/

# 5. Copy subscription updates
cp robokassa-integration/internal/domain/subscription/handler_robokassa.go \
   internal/domain/subscription/
```

**Status:** ⏳ TODO

---

### PHASE 3: Update Payment Domain ⏱️ 45 min

#### 3.1 Analyze Current `payment/handler.go`
- Remove `kaspi` import
- Remove `KaspiWebhookPayload` struct
- Remove `HandleKaspiWebhook()` method
- Update `NewHandler()` signature

#### 3.2 Analyze Current `payment/service.go`
- Check for kaspi-specific methods
- May need: `UpdatePaymentByRoboKassaInvID()` instead of `UpdatePaymentByKaspiOrderID()`
- Update payment logic to be provider-agnostic

#### 3.3 Analyze Current `payment/repository.go`
- Replace `kaspi_order_id` references with `robokassa_inv_id`
- Update database column names in queries

**Status:** ⏳ TODO

---

### PHASE 4: Update Subscription Domain ⏱️ 45 min

#### 4.1 Analyze Current `subscription/handler.go`
- Remove `KaspiClient` interface
- Remove `KaspiPaymentRequest` and `KaspiPaymentResponse` structs
- Update `NewHandler()` signature
- Change from direct client to provider factory pattern

#### 4.2 Update Subscription Routes
- Remove Kaspi-specific payment initiation
- Implement generic provider-based payment flow

#### 4.3 Check `subscription/service.go`
- Look for kaspi-specific logic
- Make provider-agnostic

**Status:** ⏳ TODO

---

### PHASE 5: Update Main Entry Point ⏱️ 30 min

#### 5.1 Update `cmd/api/main.go`

**REMOVE:**
```go
import (
	"github.com/mwork/mwork-api/internal/pkg/kaspi"  // ❌ DELETE
)

// ❌ DELETE these lines (168-175):
subscriptionKaspiClient := &subscriptionKaspiAdapter{client: kaspi.NewClient(kaspi.Config{
	BaseURL:    cfg.KaspiBaseURL,
	MerchantID: cfg.KaspiMerchantID,
	SecretKey:  cfg.KaspiSecretKey,
})}

// ❌ DELETE the subscriptionKaspiAdapter struct (lines 542-560)
```

**ADD:**
```go
import (
	paymentpkg "github.com/mwork/mwork-api/internal/pkg/payment"
	"github.com/mwork/mwork-api/internal/pkg/robokassa"
)

// ADD provider factory setup function:
func setupPaymentProviders(cfg *config.Config) *paymentpkg.ProviderFactory {
	factory := paymentpkg.NewProviderFactory()

	// Register RoboKassa provider
	roboConfig := robokassa.Config{
		MerchantLogin: cfg.RoboKassaMerchantLogin,
		Password1:     cfg.RoboKassaPassword1,
		Password2:     cfg.RoboKassaPassword2,
		TestMode:      cfg.RoboKassaTestMode,
	}
	roboProvider := paymentpkg.NewRoboKassaProvider(roboConfig)
	factory.Register("robokassa", roboProvider)

	return factory
}

// In main():
providerFactory := setupPaymentProviders(cfg)

// Update handler initialization:
paymentHandler := payment.NewHandler(paymentService, providerFactory)
subscriptionHandler := subscription.NewHandler(
	subscriptionService,
	paymentService,
	providerFactory,
	&subscription.Config{
		FrontendURL: cfg.FrontendURL,
		BackendURL:  cfg.BackendURL,
	},
)
```

**Status:** ⏳ TODO

---

### PHASE 6: Database Migrations ⏱️ 5 min

**Verify migrations exist and apply them:**

```sql
-- Migration 000048: Replace kaspi_order_id with robokassa_inv_id ✅
-- Migration 000049: Create robokassa_sequence ✅

-- Run migrations:
migrate -path ./migrations -database "postgres://..." up
```

**Status:** ⏳ TODO

---

### PHASE 7: Update .env Configuration ⏱️ 5 min

**Add to .env file:**
```env
# RoboKassa Configuration
ROBOKASSA_MERCHANT_LOGIN=your_merchant_login
ROBOKASSA_PASSWORD1=your_password_1
ROBOKASSA_PASSWORD2=your_password_2
ROBOKASSA_TEST_MODE=true

# URLs for redirects and webhooks
FRONTEND_URL=http://localhost:3000
BACKEND_URL=http://localhost:8080
```

**Or update existing .env.example:**
See `robokassa-integration/.env.robokassa.example`

**Status:** ⏳ TODO

---

### PHASE 8: Testing & Validation ⏱️ 30 min

- [ ] Code compilation check
- [ ] Unit tests for RoboKassa provider
- [ ] Integration tests for payment flow
- [ ] Webhook signature verification tests
- [ ] Test database migrations

**Status:** ⏳ TODO

---

## ⚠️ CRITICAL CHANGES TO VERIFY

### 1. **Provider Interface Implementation**
All files using payment provider must implement:
```go
type PaymentProvider interface {
    Name() string
    CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
    VerifyWebhook(signature, body string) bool
    ParseWebhook(r *http.Request) (*WebhookPayload, error)
}
```

### 2. **Payment Handler Signature Change**
**OLD:** `NewHandler(service *Service, kaspiSecret string)`  
**NEW:** `NewHandler(service *Service, factory *ProviderFactory)`

### 3. **Subscription Handler Signature Change**
**OLD:** `NewHandler(service *Service, paymentService PaymentService, kaspiClient KaspiClient, ...)`  
**NEW:** `NewHandler(service *Service, paymentService PaymentService, factory *ProviderFactory, ...)`

### 4. **Database Column Changes**
- ❌ `kaspi_order_id` → ✅ `robokassa_inv_id`
- ✅ `provider` column EXISTS (verify value)

### 5. **API Differences**
| Aspect | Kaspi | RoboKassa |
|--------|-------|-----------|
| Payment Flow | API call → redirect | Redirect URL generation |
| Webhook Format | JSON POST | Form-encoded POST |
| Signature | X-Signature header | MD5 hash in PaymentStatus param |
| Invoice ID | Order ID (string) | Sequence BIGINT |
| Idempotency | Query-based | ResultURL-based |

---

## ✅ CHECKLIST FOR COMPLETION

### Pre-Migration
- [ ] Backup current database
- [ ] Review migration files (000048, 000049)
- [ ] Prepare RoboKassa merchant credentials

### Code Changes
- [ ] Update `internal/config/config.go`
- [ ] Copy files from robokassa-integration
- [ ] Refactor `internal/domain/payment/handler.go`
- [ ] Refactor `internal/domain/payment/service.go`
- [ ] Refactor `internal/domain/payment/repository.go`
- [ ] Refactor `internal/domain/subscription/handler.go`
- [ ] Update `cmd/api/main.go`
- [ ] Remove Kaspi adapter
- [ ] Update imports (remove kaspi imports)

### Testing
- [ ] Build project successfully
- [ ] Run unit tests
- [ ] Integration tests for payment flow
- [ ] Webhook handler tests
- [ ] Test error handling

### Deployment
- [ ] Run database migrations
- [ ] Set RoboKassa environment variables
- [ ] Deploy to staging
- [ ] Full integration test
- [ ] Deploy to production
- [ ] Monitor payment tracking

### Post-Migration (Optional)
- [ ] Remove legacy `internal/pkg/kaspi/` directory
- [ ] Remove Kaspi-related migrations deprecation notes
- [ ] Update documentation

---

## 🚨 KNOWN ISSUES TO ADDRESS

### 1. **Payment Method References**
Search for direct Kaspi references:
```bash
grep -r "kaspi" --include="*.go" internal/
grep -r "Kaspi" --include="*.go" internal/
grep -r "KASPI" --include="*.go" internal/
```

### 2. **Database Query Updates**
Files likely needing attention:
- `internal/domain/payment/repository.go` - Column references
- `internal/domain/payment/service.go` - Business logic
- Any payment event processing

### 3. **Event Tracking**
- Old `kaspi_events` table may be deprecated
- Verify if needed for audit trail
- Plan for data migration if keeping historical data

---

## 📚 REFERENCE DOCUMENTATION

From robokassa-integration folder:
- **docs/ROBOKASSA_README.md** - Complete architecture overview
- **docs/ROBOKASSA_MIGRATION_GUIDE.md** - Step-by-step migration guide
- **cmd/api/main_robokassa_setup.go** - Working example of provider setup

---

## 🎯 NEXT STEPS (Senior Developer Approval)

1. **APPROVAL NEEDED:** Review this analysis
2. **REQUEST:** Confirm migration strategy (all-at-once vs phased)
3. **CONFIRMATION:** Verify RoboKassa credentials available
4. **EXECUTION:** Begin PHASE 1 (configuration updates)

