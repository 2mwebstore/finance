-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
-- Removes the platform tables and every win/lose result in them: only for rolling back a fresh install.
DROP TABLE IF EXISTS weekly_settlement_totals;
DROP TABLE IF EXISTS weekly_settlements;
DROP TABLE IF EXISTS weekly_partner_results;
DROP TABLE IF EXISTS weekly_platform_results;
DROP TABLE IF EXISTS assignment_partners;
DROP TABLE IF EXISTS hq_platform_assignments;
DROP TABLE IF EXISTS platforms;
DROP TABLE IF EXISTS partners;
