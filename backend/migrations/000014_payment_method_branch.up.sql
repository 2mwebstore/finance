-- A payment method is shared by every branch (branch_id NULL, as all existing ones stay) or
-- belongs to one branch, which alone sees and uses it.
SET @add_branch = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payment_methods' AND column_name = 'branch_id') = 0,
    'ALTER TABLE payment_methods ADD COLUMN branch_id CHAR(36) NULL, ADD INDEX idx_payment_methods_branch_id (branch_id)',
    'SELECT 1');
PREPARE add_branch FROM @add_branch;
EXECUTE add_branch;
DEALLOCATE PREPARE add_branch;
