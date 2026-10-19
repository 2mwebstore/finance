# Database Design (Phase 1 Focus)

## Core Tables (Phase 1)

- `users`
- `roles`
- `permissions`
- `user_roles`
- `role_permissions`
- `branches`
- `user_branches`
- `sessions` / `refresh_tokens`
- `audit_logs`
- `settings` (minimal)

## Money Rules (All Phases)

- Amounts stored as `DECIMAL(19,4)`
- Currency stored as `CHAR(3)` or ENUM (`USD`, `KHR`)
- Never use floating-point types for money
- Indexes on `(branch_id, currency, status, transaction_date)` for reports

## Migration Strategy

- Versioned SQL migrations in `backend/migrations/`
- Applied by migration tool at startup or via CI/CD
- No destructive auto-migrate in production
