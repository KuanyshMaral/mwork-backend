-- Rollback: Remove employer leads table

DROP INDEX IF EXISTS idx_leads_created;
DROP INDEX IF EXISTS idx_leads_follow_up;
DROP INDEX IF EXISTS idx_leads_assigned;
DROP INDEX IF EXISTS idx_leads_priority;
DROP INDEX IF EXISTS idx_leads_email;
DROP INDEX IF EXISTS idx_leads_status;

DROP TABLE IF EXISTS employer_leads;

DROP TYPE IF EXISTS lead_status;
