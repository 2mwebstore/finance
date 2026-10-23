-- Files from whoever entered a transaction (submitted) are kept apart from the ones its approver
-- added (approval). Existing files were all added while entering or editing.
SET @add_stage = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'transaction_attachments' AND column_name = 'stage') = 0,
    'ALTER TABLE transaction_attachments ADD COLUMN stage VARCHAR(16) NOT NULL DEFAULT ''submitted'' AFTER size',
    'SELECT 1');
PREPARE add_stage FROM @add_stage;
EXECUTE add_stage;
DEALLOCATE PREPARE add_stage;
