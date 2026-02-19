package storage

// validator.go is intentionally removed.
// AllowedMimeTypes and MaxFileSizes were per-purpose maps tied to the old 2-phase upload system.
// In the new architecture, MIME validation lives in the upload domain's service.go as a flat global whitelist.
// See: internal/domain/upload/service.go
