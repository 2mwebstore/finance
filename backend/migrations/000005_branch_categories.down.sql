ALTER TABLE categories
    DROP KEY idx_categories_branch,
    DROP COLUMN branch_id;
