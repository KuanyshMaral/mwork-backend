# 🔍 ROBOKASSA INTEGRATION - TECHNICAL VALIDATION REPORT

**Analysis Date:** 2026-02-09  
**Status:** ✅ READY FOR INTEGRATION  
**Severity of Changes:** MEDIUM (significant refactoring required)  

---

## 📊 DETAILED FILE-BY-FILE ANALYSIS

### ✅ CONFIGURATION LAYER

#### File: `internal/config/config.go`
**Current State:** Has Kaspi config only  
**Required Changes:**

```go
// ADD to Config struct:
type Config struct {
    // ... existing fields ...
    
    // RoboKassa Payment Configuration
    RoboKassaMerchantLogin string
    RoboKassaPassword1     string
    RoboKassaPassword2     string
    RoboKassaTestMode      bool
    FrontendURL            string  // For payment redirects
    BackendURL             string  // For webhook callbacks
}

// ADD to Load() function:
RoboKassaMerchantLogin: getEnv("ROBOKASSA_MERCHANT_LOGIN", ""),
RoboKassaPassword1:     getEnv("ROBOKASSA_PASSWORD1", ""),
RoboKassaPassword2:     getEnv("ROBOKASSA_PASSWORD2", ""),
RoboKassaTestMode:      envBool("ROBOKASSA_TEST_MODE", false),
FrontendURL:            getEnv("FRONTEND_URL", "http://localhost:3000"),
BackendURL:             getEnv("BACKEND_URL", "http://localhost:8080"),
```

**Action:** UPDATE ⏳

---

### 🔴 PAYMENT DOMAIN

#### File: `internal/domain/payment/entity.go`
**Current State:** Has `KaspiOrderID` field  
**Required Changes:**

```go
// CHANGE from:
type Payment struct {
    // ... other fields ...
    KaspiOrderID   string          `db:"kaspi_order_id" json:"kaspi_order_id,omitempty"`
    // ...
}

// CHANGE to:
type Payment struct {
    // ... other fields ...
    RoboKassaInvID *int64          `db:"robokassa_inv_id" json:"robokassa_inv_id,omitempty"`
    ExternalID     sql.NullString  `db:"external_id" json:"external_id,omitempty"`
    // ...
}

// UPDATE Provider constants:
const (
    ProviderRoboKassa Provider = "robokassa"  // NEW
    ProviderKaspi     Provider = "kaspi"      // DEPRECATED
    ProviderCard      Provider = "card"
    ProviderManual    Provider = "manual"
)
```

**Action:** UPDATE ⏳

---

#### File: `internal/domain/payment/handler.go`
**Current State:** Kaspi webhook handler (231 lines)  
**Required Changes:**

| Line | Current | Action |
|------|---------|--------|
| 1-10 | Imports kaspi | REMOVE kaspi import |
| 18-24 | `Handler struct` | CHANGE signature |
| 25-30 | `NewHandler()` func | CHANGE signature |
| 55-95 | `HandleKaspiWebhook()` | DELETE entire method |
| 55-80 | `KaspiWebhookPayload` struct | DELETE struct |
| 65-95 | Kaspi signature verification | REPLACE with provider-agnostic |

**New Handler Signature:**
```go
type Handler struct {
    service           *Service
    providerFactory   *payment.ProviderFactory  // NEW
}

func NewHandler(service *Service, factory *payment.ProviderFactory) *Handler {
    return &Handler{
        service:         service,
        providerFactory: factory,
    }
}
```

**New Webhook Handler (provider-agnostic):**
```go
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    provider := chi.URLParam(r, "provider")  // robokassa, kaspi, etc.
    
    providerImpl := h.providerFactory.Get(provider)
    if providerImpl == nil {
        response.Error(w, http.StatusNotFound, "PROVIDER_NOT_FOUND", "unknown provider")
        return
    }
    
    payload, err := providerImpl.ParseWebhook(r)
    if err != nil {
        response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
        return
    }
    
    // Update payment based on payload...
}
```

**Action:** REWRITE ⏳

---

#### File: `internal/domain/payment/service.go`
**Current State:** 254 lines, missing RoboKassa-specific methods  
**Required Changes:**

```go
// ADD new methods from robokassa-integration/service_robokassa.go:

// UpdatePaymentByInvoiceID - for RoboKassa
func (s *Service) UpdatePaymentByInvoiceID(ctx context.Context, invoiceID int64, status string) error { }

// UpdatePaymentByOrderID - generic
func (s *Service) UpdatePaymentByOrderID(ctx context.Context, orderID string, status string) error { }

// GenerateInvoiceID - for RoboKassa
func (s *Service) GenerateInvoiceID(ctx context.Context) (int64, error) { }

// CreatePaymentWithInvoiceID - RoboKassa variant
func (s *Service) CreatePaymentWithInvoiceID(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, provider Provider, invoiceID int64) (*Payment, error) { }

// REMOVE (deprecated):
// - Any Kaspi-specific methods like UpdatePaymentByKaspiOrderID
```

**Action:** UPDATE + ADD ⏳

---

#### File: `internal/domain/payment/repository.go`
**Current State:** 158 lines, uses `kaspi_order_id`  
**Required Changes:**

```go
// CHANGE Interface:
type Repository interface {
    // ... existing methods ...
    GetByInvoiceID(ctx context.Context, invoiceID int64) (*Payment, error)        // NEW
    GetNextInvoiceID(ctx context.Context) (int64, error)                          // NEW
    CreateWithInvoiceID(ctx context.Context, p *Payment) error                    // NEW
    UpdateByInvoiceID(ctx context.Context, invoiceID int64, status Status) error  // NEW
    
    // REMOVE these:
    // ConfirmPayment(ctx context.Context, kaspiOrderID string) error
}

// CHANGE methods:
func (r *repository) CreatePendingPayment(ctx context.Context, payment *Payment) error {
    // Change FROM: kaspi_order_id column
    // Change TO:  robokassa_inv_id column
    
    // Change FROM: INSERT ... kaspi_order_id, $4 ...
    // Change TO:  INSERT ... robokassa_inv_id, $4 ...
}

func (r *repository) ConfirmPayment(ctx context.Context, kaspiOrderID string) error {
    // DEPRECATED - Use UpdateByInvoiceID instead
}
```

**Action:** UPDATE ⏳

---

### 🔴 SUBSCRIPTION DOMAIN

#### File: `internal/domain/subscription/handler.go`
**Current State:** 292 lines, uses `KaspiClient` directly  
**Required Changes:**

```go
// OLD Handler struct:
type Handler struct {
    service        *Service
    paymentService PaymentService
    kaspiClient    KaspiClient                // ❌ DELETE
    config         *Config
}

// NEW Handler struct:
type Handler struct {
    service        *Service
    paymentService PaymentService
    providerFactory *payment.ProviderFactory  // ✅ NEW
    config         *Config
}

// OLD NewHandler signature:
func NewHandler(service *Service, paymentService PaymentService, kaspiClient KaspiClient, config *Config) *Handler

// NEW NewHandler signature:
func NewHandler(service *Service, paymentService PaymentService, factory *payment.ProviderFactory, config *Config) *Handler
```

**Remove These Interfaces (from handler.go):**
```go
// ❌ DELETE:
type KaspiClient interface {
    CreatePayment(ctx context.Context, req KaspiPaymentRequest) (*KaspiPaymentResponse, error)
}

type KaspiPaymentRequest struct { }
type KaspiPaymentResponse struct { }
```

**Update Payment Return Logic:**
```go
// OLD: Direct kaspiClient call
// newPaymentResp, err := h.kaspiClient.CreatePayment(ctx, req)

// NEW: Provider factory pattern
provider := h.providerFactory.Get("robokassa")
if provider == nil {
    response.Error(w, http.StatusInternalServerError, "PROVIDER_ERROR", "robokassa not configured")
    return
}
paymentResp, err := provider.CreatePayment(ctx, payment.PaymentRequest{...})
```

**Action:** REWRITE ⏳

---

#### File: `internal/domain/subscription/service.go` & `dto.go`
**Current State:** Unknown structure  
**Required Changes:**
- Remove any Kaspi-specific logic
- Update DTO to remove KaspiOrderID references
- Ensure provider-agnostic implementation

**Action:** REVIEW + UPDATE ⏳

---

### 🔴 MAIN ENTRY POINT

#### File: `cmd/api/main.go`
**Current State:** 636 lines, direct Kaspi initialization  
**Required Changes:**

**REMOVE (Lines 44, 168-175, 542-560):**
```go
// ❌ Line 44:
import (
    "github.com/mwork/mwork-api/internal/pkg/kaspi"
)

// ❌ Lines 168-175:
subscriptionKaspiClient := &subscriptionKaspiAdapter{client: kaspi.NewClient(kaspi.Config{
    BaseURL:    cfg.KaspiBaseURL,
    MerchantID: cfg.KaspiMerchantID,
    SecretKey:  cfg.KaspiSecretKey,
})}

// ❌ Lines 542-560:
type subscriptionKaspiAdapter struct {
    client *kaspi.Client
}

func (a *subscriptionKaspiAdapter) CreatePayment(ctx context.Context, req subscription.KaspiPaymentRequest) (*subscription.KaspiPaymentResponse, error) { }
```

**ADD (After database imports):**
```go
import (
    paymentpkg "github.com/mwork/mwork-api/internal/pkg/payment"
    "github.com/mwork/mwork-api/internal/pkg/robokassa"
)

// Add this function before main():
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
```

**CHANGE (Where handlers are initialized):**
```go
// OLD:
subscriptionKaspiClient := &subscriptionKaspiAdapter{...}
paymentHandler := payment.NewHandler(paymentService, cfg.KaspiSecretKey)

// NEW:
providerFactory := setupPaymentProviders(cfg)
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

**Action:** REWRITE ⏳

---

### 📦 PACKAGE LAYER

#### File: `internal/pkg/kaspi/` (DEPRECATED)
**Current State:** Client and webhook packages  
**Required Changes:**

**Option 1 (Safe):** Keep for backward compatibility, mark as DEPRECATED  
**Option 2 (Clean):** Remove after full migration to RoboKassa

**Action:** KEEP FOR NOW (will remove in Phase 2) ⏳

---

### 🗄️ DATABASE LAYER

#### Migrations Applied
- ✅ `000048_replace_kaspi_with_robokassa.up.sql` - Replaces kaspi_order_id with robokassa_inv_id
- ✅ `000049_create_robokassa_sequence.up.sql` - Creates robokassa_invoice_seq

**Required SQL Verification:**
```sql
-- Check Payment table structure
\d payments

-- Expected columns:
-- robokassa_inv_id BIGINT (after migration)
-- provider VARCHAR (supports 'robokassa', 'kaspi')
-- external_id VARCHAR (optional)

-- Check sequence
SELECT nextval('robokassa_invoice_seq');
```

**Action:** APPLY MIGRATIONS ⏳

---

### 📥 FILES TO COPY FROM `robokassa-integration/`

| Source | Destination | Purpose |
|--------|-------------|---------|
| `internal/pkg/payment/provider.go` | Same | Provider interface |
| `internal/pkg/payment/robokassa_provider.go` | Same | RoboKassa adapter |
| `internal/pkg/robokassa/client.go` | Same | RoboKassa client |
| `internal/pkg/robokassa/webhook.go` | Same | Webhook handler |
| `internal/pkg/robokassa/client_test.go` | Same | Unit tests |
| `internal/pkg/robokassa/webhook_test.go` | Same | Webhook tests |

**Already Exist (Don't copy):**
- The sample files like `handler_new.go` - use as reference only

**Action:** COPY FILES ⏳

---

## 🎯 IMPLEMENTATION ORDER (Critical Path)

### ✅ Block 1: Foundation (MUST do first)
1. Update `internal/config/config.go` - Config fields
2. Copy payment provider abstraction (`internal/pkg/payment/`)
3. Copy RoboKassa client (`internal/pkg/robokassa/`)
4. Update `internal/domain/payment/entity.go` - Database fields

### ✅ Block 2: Core Domain Logic (No dependencies)
5. Update `internal/domain/payment/repository.go` - New methods
6. Update `internal/domain/payment/service.go` - New methods + old methods removal
7. Update `internal/domain/payment/handler.go` - Webhook handlers

### ✅ Block 3: Subscription (Depends on Block 2)
8. Update `internal/domain/subscription/handler.go` - Remove KaspiClient
9. Update `internal/domain/subscription/service.go` - If needed

### ✅ Block 4: Integration (Depends on Blocks 1-3)
10. Update `cmd/api/main.go` - Provider factory + adapters
11. Apply database migrations

### ✅ Block 5: Testing & Deployment
12. Run tests and validation
13. Deploy with RoboKassa environment variables

---

## ⚠️ HIGH-RISK AREAS

### 🔴 CRITICAL - Must Handle Correctly

1. **Database Migration Safety**
   - `kaspi_order_id` → `robokassa_inv_id` conversion
   - Index management
   - Data loss prevention
   - Migration rollback plan

2. **Webhook Signature Verification**
   - RoboKassa uses different signature method than Kaspi
   - MD5 hash vs HMAC
   - Form-encoded vs JSON payload
   - Test thoroughly before production

3. **Invoice ID Generation**
   - RoboKassa requires unique numeric IDs
   - Database sequence collision handling
   - Fallback logic if sequence fails

4. **Payment Flow Changes**
   - Kaspi: API call → Payment page
   - RoboKassa: URL generation → Redirect
   - Return URL vs Webhook handling
   - State management differences

### 🟡 MEDIUM - Watch Closely

5. **Backward Compatibility**
   - Old payments with kaspi_order_id
   - Payment history queries
   - Reporting and analytics
   - Consider keeping kaspi_events table for audit

6. **Configuration Management**
   - Multiple environment variables
   - Test vs Production credentials
   - Fallback configurations
   - Documentation updates

---

## 📋 VALIDATION CHECKLIST

### Code Compilation ⏳
- [ ] All imports resolve correctly
- [ ] No undefined types or methods
- [ ] Go build succeeds without errors
- [ ] Go vet passes

### Type Safety ⏳
- [ ] All Handler signatures updated consistently
- [ ] Service methods match repository interface
- [ ] Payment entity fields consistent across codebase
- [ ] No orphaned Kaspi references

### Database ⏳
- [ ] Migration 000048 runs successfully
- [ ] Migration 000049 runs successfully
- [ ] Sequence created: robokassa_invoice_seq
- [ ] Column renamed/added: robokassa_inv_id
- [ ] Rollback migrations tested

### Functional ⏳
- [ ] Payment creation works
- [ ] Invoice ID generation works
- [ ] Webhook signature verification works
- [ ] Payment status updates work
- [ ] No double-charging scenarios

### Integration ⏳
- [ ] Payment → Subscription flow works
- [ ] Payment → Credit flow works
- [ ] Webhook handling (if applicable)
- [ ] Error handling consistent

---

## 📊 EFFORT ESTIMATE

| Task | Hours | Risk |
|------|-------|------|
| Configuration Updates | 0.25 | LOW |
| Copy New Files | 0.25 | LOW |
| Entity & Repository Updates | 1.5 | MEDIUM |
| Payment Handler Refactor | 1.5 | MEDIUM |
| Payment Service Updates | 1.0 | MEDIUM |
| Subscription Handler Refactor | 1.5 | MEDIUM |
| Main.go Refactor | 1.0 | HIGH |
| Testing & Validation | 1.5 | HIGH |
| Database Migrations | 0.5 | HIGH |
| **TOTAL** | **9 hours** | |

---

## 🚀 SUCCESS CRITERIA

✅ Project compiles without errors  
✅ All tests pass  
✅ Database migrations applied successfully  
✅ Payment creation generates correct invoice IDs  
✅ Webhook signatures verify correctly  
✅ Payment status updates work end-to-end  
✅ Subscription activation on payment works  
✅ No Kaspi imports in active code paths  
✅ RoboKassa provider registered and available  
✅ Environment variables loaded correctly  

