-- Create sequence for generating RoboKassa invoice IDs
-- Starts at 1000 to avoid conflicts with existing IDs

CREATE SEQUENCE IF NOT EXISTS robokassa_invoice_seq
    START WITH 1000
    INCREMENT BY 1
    NO MAXVALUE
    NO MINVALUE
    CACHE 1;

-- Grant usage on sequence to application user
-- GRANT USAGE, SELECT ON SEQUENCE robokassa_invoice_seq TO your_app_user;

-- Note: The robokassa_inv_id column was already added in migration 000048
-- This migration only adds the sequence for ID generation
