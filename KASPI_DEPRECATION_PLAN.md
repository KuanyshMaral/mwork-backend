# Kaspi Deprecation & Removal Plan

**Status:** ✅ Migration Complete (Feb 9, 2026)  
**Timeline:** 3-Month Deprecation Period

---

## 📋 Current Kaspi Code Inventory

### Files That Still Exist

| File | Size | Status | Action |
|------|------|--------|--------|
| `internal/pkg/kaspi/client.go` | ~180 lines | **DEAD CODE** | Remove in v2.0 |
| `internal/pkg/kaspi/webhook.go` | ~120 lines | **DEAD CODE** | Remove in v2.0 |
| `migrations/000023_add_kaspi_order_id.up/down.sql` | - | **HISTORIC** | Keep indefinitely |
| `migrations/000033_create_kaspi_events.up/down.sql` | - | **HISTORIC** | Keep indefinitely |

### Data That Still Exists

| Data | Location | Status | Action |
|------|----------|--------|--------|
| `KaspiOrderID` field | `Payment` struct | Deprecated ✋ | Keep for backward compatibility |
| `kaspi_order_id` column | `payments` table | Deprecated ✋ | Keep for historical queries |
| `kaspi_events` table | Database | Legacy | Archive/keep as read-only |

---

## 🗓️ Deprecation Timeline

### **Phase 1: NOW - Immediate (Week 1)**
**Goal:** Prevent new Kaspi usage

- ✅ Delete `internal/pkg/kaspi/` package (2 files)
- ✅ Add deprecation comments to `KaspiOrderID` field:
  ```go
  // DEPRECATED: Replaced by RoboKassaInvID. Kept for historical data only.
  // Do NOT use for new payments. This field will be removed in v2.0 (May 2026)
  KaspiOrderID string `db:"kaspi_order_id" json:"kaspi_order_id,omitempty"`
  ```
- ✅ Remove any Kaspi-specific code from active handlers
- ✅ Document in CHANGELOG

**Release:** v1.5.0

---

### **Phase 2: 4-8 Weeks After Migration**
**Goal:** Monitor & validate RoboKassa stability

**Actions:**
- Monitor payment success rates
- Validate all old Kaspi payments are still queryable
- Confirm no regressions

**Keep:** All fields, all migrations, all historical data

**Release:** v1.5.x (patch releases only)

---

### **Phase 3: 8-12 Weeks After (May 15, 2026)**
**Goal:** Final cleanup before v2.0

**Actions:**
- Remove `KaspiOrderID` from Payment struct
- Drop `kaspi_order_id` column (migration 000050)
- Drop `kaspi_events` table if unused (migration 000051)
- Remove migration files 000023, 000033 from codebase (keep docs/archive/)

**Breaking Change:** v2.0.0 (Major version bump required)

**Migration required for users:**
```bash
# Before upgrade to v2.0
SELECT COUNT(*) FROM payments WHERE kaspi_order_id IS NOT NULL;
# Export/archive if needed
```

---

## 🛠️ Recommended Actions for Today

### **Immediate (30 min):** Delete Dead Code

Delete these 2 files - they're no longer imported or used:
- `internal/pkg/kaspi/client.go` → 🗑️ DELETE
- `internal/pkg/kaspi/webhook.go` → 🗑️ DELETE
- Delete `internal/pkg/kaspi/` directory

### **Short-term (optional):** Archive for Reference

If you want to keep Kaspi code for team reference:
- Create `docs/deprecated/kaspi-integration/` folder
- Copy Kaspi files there with README explaining deprecation
- Link from main README

### **Documentation:** Update Files

Add to `README.md`:
```markdown
## Payment Providers

### ✅ RoboKassa (Active)
Currently used for all new payments since Feb 2026.

### ⚠️ Kaspi (Deprecated)
Support ended Feb 9, 2026. Migrated to RoboKassa.
- Historical payment data preserved in `payments.kaspi_order_id` column
- Scheduled for complete removal in v2.0 (May 2026)
```

---

## 💾 What NOT to Delete (Keep Forever)

```
migrations/000023_add_kaspi_order_id.up.sql      ← KEEP (history)
migrations/000023_add_kaspi_order_id.down.sql    ← KEEP (history)
migrations/000033_create_kaspi_events.up.sql     ← KEEP (history)
migrations/000033_create_kaspi_events.down.sql   ← KEEP (history)
```

**Why:** These migrations represent your database evolution. Never delete them, even when features are removed.

---

## 📊 Migration Checklist

- ✅ RoboKassa payment creation working
- ✅ RoboKassa webhooks processed
- ✅ All payments routable via provider factory
- ⏳ **TODO:** Delete `internal/pkg/kaspi/` package
- ⏳ **TODO:** Add deprecation comments to `KaspiOrderID`
- ⏳ **TODO:** Update README with deprecation notice
- ⏳ **TODO:** Update CHANGELOG for v1.5.0

---

## 🔍 Verification Query

Before any final cleanup, verify historical data:

```sql
-- How many Kaspi payments exist?
SELECT COUNT(*) as kaspi_payments FROM payments WHERE provider = 'kaspi' OR kaspi_order_id IS NOT NULL;

-- Check distribution by status
SELECT status, COUNT(*) FROM payments WHERE kaspi_order_id IS NOT NULL GROUP BY status;

-- Export for archive if needed
COPY (SELECT * FROM payments WHERE kaspi_order_id IS NOT NULL) TO '/tmp/kaspi_backup.csv' WITH CSV HEADER;
```

---

## 📝 Version Tags

| Version | Date | Action |
|---------|------|--------|
| v1.4.x | Before Feb 9 | Old Kaspi system |
| **v1.5.0** | Feb 9, 2026 | RoboKassa live, Kaspi deprecated |
| v1.5.1+ | Feb-May 2026 | Monitoring period |
| **v2.0.0** | May 15, 2026 | Kaspi fully removed |

---

## ⚠️ Risks & Mitigation

| Risk | Mitigation |
|------|-----------|
| Can't query old Kaspi data | Keep `kaspi_order_id` column indefinitely |
| Breaking change in v2.0 | Announce 3 months in advance, document migration |
| Accidental Kaspi reference | Delete `internal/pkg/kaspi/` now to prevent usage |
| Edge cases missed | Monitor logs during Phase 2 for "kaspi" references |

---

**Next Step:** Should I delete `internal/pkg/kaspi/` now? 👀
