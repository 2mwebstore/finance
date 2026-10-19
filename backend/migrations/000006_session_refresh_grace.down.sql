-- Safe to run when the columns do not exist yet (MySQL initdb runs *.down.sql before *.up.sql).
SET @drop_refresh_grace = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'sessions' AND column_name = 'previous_token_hash') > 0,
    'ALTER TABLE sessions DROP KEY idx_sessions_previous_token_hash, DROP COLUMN rotated_at, DROP COLUMN previous_token_hash',
    'SELECT 1');
PREPARE drop_refresh_grace FROM @drop_refresh_grace;
EXECUTE drop_refresh_grace;
DEALLOCATE PREPARE drop_refresh_grace;
