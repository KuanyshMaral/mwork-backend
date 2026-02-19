package upload

// service_batch.go is intentionally removed.
// Batch upload logic (StageBatch, CommitBatch) was part of the 2-phase staging system.
// In the new local-first architecture, each file is uploaded individually via POST /files.
// Batch uploads from the frontend can simply be done with multiple POST /files requests.
