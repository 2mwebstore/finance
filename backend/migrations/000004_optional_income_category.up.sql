-- Income records do not require a category. Expense category validation stays in the service.
ALTER TABLE transactions
    MODIFY COLUMN category_id CHAR(36) NULL;
