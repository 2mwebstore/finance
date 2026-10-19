-- Restore a matching category before making the column required again.
UPDATE transactions t
JOIN categories c ON c.type = t.type
SET t.category_id = c.id
WHERE t.category_id IS NULL;

ALTER TABLE transactions
    MODIFY COLUMN category_id CHAR(36) NOT NULL;
