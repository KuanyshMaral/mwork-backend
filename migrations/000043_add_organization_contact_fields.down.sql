ALTER TABLE organizations
    DROP COLUMN IF EXISTS contact_person,
    DROP COLUMN IF EXISTS contact_phone,
    DROP COLUMN IF EXISTS contact_telegram,
    DROP COLUMN IF EXISTS contact_whatsapp;
