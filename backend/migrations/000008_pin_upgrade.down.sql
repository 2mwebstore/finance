-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
SET @drop_pin_upgrade = IF(
    (SELECT COUNT(*) FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'pin_upgrade_required') > 0,
    'ALTER TABLE users DROP COLUMN pin_upgrade_required',
    'SELECT 1');
PREPARE drop_pin_upgrade FROM @drop_pin_upgrade;
EXECUTE drop_pin_upgrade;
DEALLOCATE PREPARE drop_pin_upgrade;
