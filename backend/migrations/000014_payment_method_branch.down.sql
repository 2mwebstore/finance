-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
SET @drop_branch = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payment_methods' AND column_name = 'branch_id') > 0,
    'ALTER TABLE payment_methods DROP INDEX idx_payment_methods_branch_id, DROP COLUMN branch_id',
    'SELECT 1');
PREPARE drop_branch FROM @drop_branch;
EXECUTE drop_branch;
DEALLOCATE PREPARE drop_branch;
