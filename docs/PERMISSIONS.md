# Permission Codes Reference

Server-side authorization is enforced on every protected API. The UI uses the same codes for visibility only.

**Total: 30 permissions** across 10 modules.

## Full table

| Code | Name | Module | Description |
|------|------|--------|-------------|
| `dashboard.view` | View Dashboard | Dashboard | Access the main dashboard |
| `income.view` | View Income | Income | View income transactions |
| `income.create` | Create Income | Income | Create income transactions |
| `income.edit` | Edit Income | Income | Edit income transactions |
| `income.void` | Void Income | Income | Void income transactions |
| `expense.view` | View Expenses | Expenses | View expense transactions |
| `expense.create` | Create Expenses | Expenses | Create expense transactions |
| `expense.edit` | Edit Expenses | Expenses | Edit expense transactions |
| `expense.void` | Void Expenses | Expenses | Void expense transactions |
| `transactions.view` | View Transactions | Transactions | View all transactions |
| `transactions.approve` | Approve Transactions | Transactions | Approve pending transactions |
| `transactions.reject` | Reject Transactions | Transactions | Reject pending transactions |
| `transactions.reverse` | Reverse Transactions | Transactions | Reverse finalized transactions |
| `reports.view` | View Reports | Reports | View financial reports |
| `reports.export` | Export Reports | Reports | Export reports to CSV/Excel/PDF |
| `branches.view` | View Branches | Branches | View branch list and details |
| `branches.create` | Create Branches | Branches | Create new branches |
| `branches.edit` | Edit Branches | Branches | Edit branch details |
| `branches.manage` | Manage Branches | Branches | Full branch management including assignments |
| `users.view` | View Users | Users | View user list and profiles |
| `users.create` | Create Users | Users | Create new users |
| `users.edit` | Edit Users | Users | Edit user profiles |
| `users.manage` | Manage Users | Users | Manage user roles, passwords, and account activation |
| `roles.view` | View Roles | Roles | View roles and their permissions |
| `roles.manage` | Manage Roles | Roles | Create, edit, and delete non-system roles |
| `permissions.manage` | Manage Permissions | Roles | Assign permissions to roles |
| `permissions.view` | View Permission Codes | Roles | See the Permission Codes page with every permission in the system |
| `categories.manage` | Manage Categories | Settings | Create, edit, and deactivate income/expense categories |
| `currency-settings.manage` | Manage Currency Settings | Settings | Configure currency options |
| `audit-logs.view` | View Audit Logs | Audit | View system audit logs |
| `settings.manage` | Manage Settings | Settings | Change application settings |

## By module

### Dashboard
| Code | Description |
|------|-------------|
| `dashboard.view` | Access the main dashboard |

### Income
| Code | Description |
|------|-------------|
| `income.view` | View income transactions |
| `income.create` | Create income transactions |
| `income.edit` | Edit income transactions |
| `income.void` | Void income transactions |

### Expenses
| Code | Description |
|------|-------------|
| `expense.view` | View expense transactions |
| `expense.create` | Create expense transactions |
| `expense.edit` | Edit expense transactions |
| `expense.void` | Void expense transactions |

### Transactions
| Code | Description |
|------|-------------|
| `transactions.view` | View all transactions |
| `transactions.approve` | Approve pending transactions |
| `transactions.reject` | Reject pending transactions |
| `transactions.reverse` | Reverse finalized transactions |

### Reports
| Code | Description |
|------|-------------|
| `reports.view` | View financial reports |
| `reports.export` | Export reports to CSV/Excel/PDF |

### Branches
| Code | Description |
|------|-------------|
| `branches.view` | View branch list and details |
| `branches.create` | Create new branches |
| `branches.edit` | Edit branch details |
| `branches.manage` | Full branch management including assignments |

### Users
| Code | Description |
|------|-------------|
| `users.view` | View user list and profiles |
| `users.create` | Create new users |
| `users.edit` | Edit user profiles |
| `users.manage` | Manage user roles, passwords, and account activation |

### Roles
| Code | Description |
|------|-------------|
| `roles.view` | View roles and their permissions |
| `roles.manage` | Create, edit, and delete non-system roles |
| `permissions.manage` | Assign permissions to roles |
| `permissions.view` | Show the Permission Codes page (menu item and page); without it the page is hidden |

### Settings & Audit
| Code | Description |
|------|-------------|
| `categories.manage` | Create, edit, and deactivate income/expense categories |
| `currency-settings.manage` | Configure currency options |
| `settings.manage` | Change application settings |
| `audit-logs.view` | View system audit logs |

## Default system roles (summary)

| Role | Typical access |
|------|----------------|
| **super_admin** | All permissions (flag `is_super_admin`; not only role codes) |
| **admin** | Broad manage: users, branches, roles, reports, transactions, settings |
| **manager** | Branch operations, approvals, reports, limited admin |
| **accountant** | Income/expense full cycle, reports, categories |
| **cashier** | Create income/expense, view own-branch data |
| **viewer** | Read-only dashboard, transactions, reports |

Exact role → permission mappings are seeded in `backend/internal/permissions` and returned by `GET /api/v1/roles`.

## Machine-readable catalog

Project root: [`permissions.catalog.json`](../permissions.catalog.json) (30 entries, versioned).

## Source of truth

- Backend constants: `backend/internal/permissions/permissions.go`
- Frontend constants: `frontend/utils/permissions.ts` (`P` + `PERMISSION_CATALOG`)
- API list: `GET /api/v1/permissions` (requires `roles.view`)
