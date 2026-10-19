# Architecture

## High-Level

```
┌─────────────────┐     HTTPS      ┌──────────────────┐
│  Nuxt 3 SPA     │ ◄────────────► │  Nginx / Reverse │
│  (TypeScript)   │                │  Proxy           │
└─────────────────┘                └────────┬─────────┘
                                            │
                                   ┌────────▼─────────┐
                                   │  Go + Gin API    │
                                   │  /api/v1/*       │
                                   └────────┬─────────┘
                                            │
                                   ┌────────▼─────────┐
                                   │  MySQL 8         │
                                   │  (DECIMAL money) │
                                   └──────────────────┘
```

## Backend Layers

1. **Controllers** — HTTP request/response, input binding, status codes
2. **Services** — Business rules, authorization decisions, orchestration
3. **Repositories** — Data access (GORM + explicit SQL where needed)
4. **Models** — Domain entities
5. **Middleware** — Auth (JWT + session), RBAC, branch scope, rate limiting, logging
6. **Audit** — Centralized event recording

## Frontend Layers

- Pages (file-based routing)
- Components (UI + domain)
- Composables / Stores (state)
- Typed API client (centralized)
- Middleware (route guards — UX only, never security boundary)

## Key Design Decisions

- **Money**: MySQL `DECIMAL(19,4)` + `shopspring/decimal` in Go. Never `float64`.
- **Currencies**: Every transaction carries its own currency. Aggregates always separated (USD vs KHR).
- **Authorization**: Dual enforcement — middleware for route-level + service-level for resource ownership.
- **Branch isolation**: Explicit `user_branches` mapping. Super Admin / Admin may have global scope if granted.
- **Tokens**: Short-lived access JWT (memory) + rotating refresh tokens in HttpOnly Secure cookies.
- **Migrations**: Versioned SQL files (golang-migrate or equivalent). No auto-migrate in production.
- **Audit**: Immutable append-only log for security and financial events.

## Directory Layout (Target)

See master prompt. Adapt only when justified by real constraints.

## Authorization

Role-based access control uses fine-grained permission codes. See **[PERMISSIONS.md](./PERMISSIONS.md)** for the complete table.
