-- Keep the previous refresh token hash briefly after rotation (parallel refresh grace period).
ALTER TABLE sessions
    ADD COLUMN previous_token_hash VARCHAR(255) NULL AFTER refresh_token_hash,
    ADD COLUMN rotated_at DATETIME(3) NULL AFTER previous_token_hash,
    ADD KEY idx_sessions_previous_token_hash (previous_token_hash);
