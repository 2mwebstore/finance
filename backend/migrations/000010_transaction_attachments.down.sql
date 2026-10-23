-- Safe to run before the up file (MySQL initdb runs *.down.sql first).
DROP TABLE IF EXISTS transaction_attachments;
