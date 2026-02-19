-- 000070_add_attachments.up.sql
-- Phase 3: Create the polymorphic attachments table.
-- This replaces the `photos` concept with a generic 1:N relationship mechanism.
-- Business labels (what the file IS) live here; the upload record is just data.

CREATE TABLE attachments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id   UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,

    -- Polymorphic target: what entity does this file belong to?
    target_id   UUID        NOT NULL,
    target_type VARCHAR(50) NOT NULL CHECK (target_type IN (
        'model_portfolio',   -- Model portfolio photos
        'casting_gallery',   -- Casting images/gallery
        'org_document',      -- Organization verification docs
        'chat_attachment'    -- Chat file attachments (future)
    )),

    -- Ordering within a collection
    sort_order  INT NOT NULL DEFAULT 0,

    -- Flexible metadata: captions, project names, custom fields per target_type
    -- Use typed structs in Go service layer, not raw maps.
    metadata    JSONB NOT NULL DEFAULT '{}',

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Enforce uniqueness: same file can't be attached twice to same target
    UNIQUE (upload_id, target_id, target_type)
);

-- Core lookup: "give me all files for this entity"
CREATE INDEX idx_attachments_target ON attachments(target_type, target_id, sort_order);
-- Reverse lookup: "is this file attached anywhere?"
CREATE INDEX idx_attachments_upload_id ON attachments(upload_id);

COMMENT ON TABLE attachments IS
    'Polymorphic 1:N file→entity relationships. Upload module provides raw files; '
    'attachments provide business labels (what role a file plays).';
COMMENT ON COLUMN attachments.target_type IS
    'VARCHAR with CHECK constraint instead of ENUM for easy future extension.';
COMMENT ON COLUMN attachments.metadata IS
    'Per-target-type structured data. Go layer uses typed structs; DB stores JSON.';
