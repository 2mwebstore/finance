-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
SET @drop_stage = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'transaction_attachments' AND column_name = 'stage') > 0,
    'ALTER TABLE transaction_attachments DROP COLUMN stage',
    'SELECT 1');
PREPARE drop_stage FROM @drop_stage;
EXECUTE drop_stage;
DEALLOCATE PREPARE drop_stage;
