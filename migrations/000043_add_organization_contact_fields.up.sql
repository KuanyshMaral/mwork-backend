-- Add contact fields to organizations for employer verification flow

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS contact_person VARCHAR(200),
    ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS contact_telegram VARCHAR(100),
    ADD COLUMN IF NOT EXISTS contact_whatsapp VARCHAR(100);
