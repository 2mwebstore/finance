# Implementation Plan — Income & Expense Management System

**Project**: Production-ready Income & Expense Management System for Cambodia  
**Stack**: Go + Gin + MySQL 8 (backend) | Nuxt 3 + Vue 3 + TypeScript + Tailwind + shadcn-vue (frontend)  
**Currencies**: USD ($) and KHR (៛) — never mixed  
**Design**: Blue (#2563EB) + Black, mobile-first  

## Current Status

- Repository was empty at start.
- Initializing full project structure.
- Phase 1 (Security Foundation) is the current focus.

## Phases Overview

| Phase | Name | Priority | Status |
|-------|------|----------|--------|
| 1 | Security Foundation & Access Control | Highest | In Progress |
| 2 | Branches, Users & Administration | High | Pending |
| 3 | Financial Data & Currency Foundation | High | Pending |
| 4 | Income & Expense Modules | High | Pending |
| 5 | Dashboard & Reports | High | Pending |
| 6 | Cash Control & Approvals | Medium | Pending |
| 7 | Audit Logs & Notifications | Medium | Pending |
| 8 | Database & API Contracts | Continuous | Pending |
| 9 | Frontend Implementation | Continuous | Pending |
| 10 | Testing & QA | Continuous | Pending |
| 11 | Production Deployment | Final | Pending |

## Phase 1 Acceptance Criteria (Release Gate)

- [ ] Secure login / logout / refresh-token rotation
- [ ] HttpOnly Secure cookies for refresh tokens (preferred)
- [ ] Argon2id or strong bcrypt password hashing
- [ ] First-run Super Admin provisioning (no hardcoded credentials)
- [ ] Configurable roles + permissions
- [ ] Default roles: Super Admin, Admin, Manager, Accountant, Cashier, Viewer
- [ ] Branch-level authorization enforced on every protected endpoint
- [ ] Server-side permission checks on every protected operation
- [ ] Automated security tests pass (anonymous access, role escalation, branch isolation, etc.)
- [ ] Migrations versioned and applied cleanly
- [ ] No floating-point money handling anywhere in foundation

## Development Rules (Non-negotiable)

1. Never skip backend authorization.
2. Never trust frontend permissions.
3. Never mix USD and KHR without explicit conversion.
4. Never use float for money.
5. Never silently delete finalized financial records.
6. Never hardcode production credentials.
7. Never claim a feature works without tests.
8. Never leave primary action buttons non-functional.
9. Never advance phase while critical tests fail.
10. Never expose data belonging to unauthorized user/branch.

## Next Immediate Steps

1. Create backend Go module + Gin skeleton.
2. Create database models + first migrations (users, roles, permissions, branches, sessions).
3. Implement auth service (login, refresh, logout, password hashing).
4. Implement RBAC middleware + branch access policy.
5. Seed default roles & permissions.
6. Secure first-run admin setup.
7. Write and run Phase 1 security tests.
8. Document progress.
