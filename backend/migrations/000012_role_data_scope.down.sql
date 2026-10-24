-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
DROP TABLE IF EXISTS role_branches;
SET @drop_scope = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'roles' AND column_name = 'data_scope') > 0,
    'ALTER TABLE roles DROP COLUMN data_scope',
    'SELECT 1');
PREPARE drop_scope FROM @drop_scope;
EXECUTE drop_scope;
DEALLOCATE PREPARE drop_scope;
