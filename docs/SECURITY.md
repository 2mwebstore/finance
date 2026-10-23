# Security Model

## Authentication

- Login: username **or** email + PIN of 4 to 6 digits (people may choose 6 for extra safety)
- Password hashing: Argon2id (preferred) or bcrypt cost ≥ 12
- Access token: short-lived JWT (15 min default), stored in memory on client
- Refresh token: rotating, stored in HttpOnly + Secure + SameSite=Strict cookie
- Session revocation: both single-session and “logout all”
- Failed login tracking + rate limiting
- First-run Super Admin provisioning only (disabled after success)

## Authorization (RBAC + Branch)

- Permissions are granular strings (e.g. `income.create`, `branches.manage`)
- Roles aggregate permissions
- Users have one or more roles
- Users have explicit branch assignments (`user_branches`)
- Global access only when a high-privilege role/permission is present
- **Every** protected endpoint checks:
  1. Valid session / token
  2. Required permission(s)
  3. Branch access for the requested resource

## Critical Rules

- Never trust client-supplied user ID, role, or branch ID as proof of authority.
- Never allow a user to grant themselves extra permissions.
- Never allow the last Super Admin to be demoted/deleted without replacement.
- Audit all administrative and financial mutations.
- Do not log passwords, tokens, or full card numbers.

## Phase 1 Test Cases (Must Pass)

1. Anonymous → 401 on protected routes
2. Expired / revoked session → 401
3. Cashier cannot manage roles
4. Viewer cannot create transactions
5. User cannot self-escalate permissions
6. Branch A user cannot read Branch B data
7. Changing branch ID in request does not bypass access
8. Unauthorized attachment / export denied
9. Admin actions produce audit records

## Permission codes

Full catalog of the **30** RBAC permission codes (code, name, module, description):

→ **[PERMISSIONS.md](./PERMISSIONS.md)**

Server enforcement lives in `backend/internal/permissions` and middleware; the UI only uses the same codes for visibility.
