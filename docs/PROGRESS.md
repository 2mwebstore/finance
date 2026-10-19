# Progress Tracker

**Last updated**: 2026-10-17 (Admin UI)

## Admin UI — delivered

| Route | Features |
|-------|----------|
| `/admin/branches` | List + create branch (code, name) |
| `/admin/users` | List + create user (role + branch assignment) |
| `/admin/roles` | List roles, expand permission details |
| `/admin/audit` | Filterable audit log |

### Navigation
- Desktop sidebar shows Admin links when user has permission / is super admin
- Mobile: Admin links under **More → Administration**

### Permission gates (client-side UX)
- `branches.view` / `branches.create`
- `users.view` / `users.create`
- `roles.view`
- `audit-logs.view`
- Server still enforces all authorization

### Files
```
frontend/pages/admin/branches.vue
frontend/pages/admin/users.vue
frontend/pages/admin/roles.vue
frontend/pages/admin/audit.vue
frontend/layouts/default.vue  (updated nav)
frontend/pages/more.vue       (admin section)
frontend/types/api.ts         (Role, AdminUser, AuditEntry)
```
