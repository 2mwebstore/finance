-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
DROP TABLE IF EXISTS user_managers;
SET @drop_attachments = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'transactions' AND column_name = 'attachment_key') > 0,
    'ALTER TABLE transactions DROP COLUMN attachment_size, DROP COLUMN attachment_type, DROP COLUMN attachment_name, DROP COLUMN attachment_key, MODIFY COLUMN description VARCHAR(1024) NULL',
    'SELECT 1');
PREPARE drop_attachments FROM @drop_attachments;
EXECUTE drop_attachments;
DEALLOCATE PREPARE drop_attachments;
