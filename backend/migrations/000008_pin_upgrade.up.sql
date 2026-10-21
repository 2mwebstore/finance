-- PINs grow from 4 to 6 digits. Old PINs keep working; users who still have one are asked to
-- choose a new PIN. Every PIN stored before this migration is a 4-digit one.
ALTER TABLE users
    ADD COLUMN pin_upgrade_required BOOLEAN NOT NULL DEFAULT FALSE AFTER password_hash;
UPDATE users SET pin_upgrade_required = TRUE;
