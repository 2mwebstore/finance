-- Safe to run when the column does not exist yet (MySQL initdb runs *.down.sql before *.up.sql).
SET @drop_branch_categories = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'categories' AND column_name = 'branch_id') > 0,
    'ALTER TABLE categories DROP KEY idx_categories_branch, DROP COLUMN branch_id',
    'SELECT 1');
PREPARE drop_branch_categories FROM @drop_branch_categories;
EXECUTE drop_branch_categories;
DEALLOCATE PREPARE drop_branch_categories;
