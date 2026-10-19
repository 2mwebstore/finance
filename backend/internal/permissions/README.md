# Permissions package

Canonical RBAC permission codes for the Income & Expense API.

## Files

| File | Purpose |
|------|---------|
| `permissions.go` | Constants, `AllPermissions()`, `DefaultRoles()` |
| `codes.go` | `Codes()`, `IsValid()`, `CodeList`, `Count` |
| `permissions_test.go` | Uniqueness & role reference tests |

## Usage

```go
import "github.com/cambodia-finance/income-expense/internal/permissions"

// In middleware / handlers
middleware.RequireAnyPermission(db, permissions.IncomeCreate, permissions.ExpenseCreate)

// Validation
if !permissions.IsValid(code) { ... }

// Seeding (called from database.SeedPermissionsAndRoles)
for _, p := range permissions.AllPermissions() { ... }
```

## Count

**30** permission codes. Keep in sync with:

- `permissions.catalog.json` (repo root)
- `frontend/utils/permissions.ts`
- `docs/PERMISSIONS.md`
