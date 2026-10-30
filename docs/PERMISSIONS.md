# Permission Codes Reference

Server-side authorization is enforced on every protected API. The UI uses the same codes for visibility only.

**Total: 57 permissions** across 18 modules.

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
| `inventory.view` | View Inventory | Inventory | See products, stock levels and stock history of your branches |
| `inventory.manage` | Manage Products | Inventory | Create and edit products and product categories (shared by every branch) |
| `inventory.adjust` | Adjust Stock | Inventory | Add, remove or count stock in your branches, with a reason |
| `inventory.transfer` | Transfer Stock | Inventory | Move stock between branches |
| `sales.view` | View Sales | Sales | See sales of your branches |
| `sales.create` | Create Sales | Sales | Record sales; their stock goes out right away |
| `sales.edit` | Edit Sales | Sales | Change a sale; its goods are put back and the new lines taken out of stock |
| `sales.void` | Void Sales | Sales | Void a sale and put its stock back |
| `purchases.view` | View Purchases | Purchases | See purchases of your branches |
| `purchases.create` | Create Purchases | Purchases | Record purchases from suppliers; stock goes in once approved |
| `purchases.approve` | Approve Purchases | Purchases | Approve or reject purchases; approved purchases add stock |
| `purchases.edit` | Edit Purchases | Purchases | Change a pending purchase (an approved one also needs Approve Purchases) |
| `purchases.void` | Void Purchases | Purchases | Void a purchase and take its stock back out |
| `inventory-expenses.view` | View Inventory Expenses | Inventory Expenses | See inventory expenses of your branches |
| `inventory-expenses.create` | Create Inventory Expenses | Inventory Expenses | Record inventory expenses (shipping, packaging, storage…) |
| `inventory-expenses.void` | Void Inventory Expenses | Inventory Expenses | Void an inventory expense |
| `platforms.view` | View Agents | Platforms | See agents and the groups they are in |
| `platforms.manage` | Manage Agents | Platforms | Add and edit agents and their currency |
| `platform-assignments.manage` | Manage Group Agent Assignments | Platforms | Make titled groups of agents with their rates; update, turn off or delete them |
| `winlose.view` | View Win/Lose | Win/Lose | See the weekly win/lose entered for your groups |
| `winlose.create` | Enter Win/Lose | Win/Lose | Enter a week's win/lose and commission for your groups (once a week) |
| `winlose.edit` | Correct Win/Lose | Win/Lose | Change a week already saved, with a reason (kept in the audit log) |
| `winlose-reports.view` | View Win/Lose Reports | Win/Lose Reports | The weekly report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | View Consolidated Win/Lose | Win/Lose Reports | Combined totals across all your groups |
| `winlose-settlements.finalize` | Finalize Weekly Settlements | Win/Lose Reports | Finalize or reopen a group's week; a finalized week can't be changed |

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
| `currency-settings.manage` | Add, change and turn off currencies (rate per 1 USD) and set the default currency (Manage → Currency) |
| `settings.manage` | Change application settings |
| `audit-logs.view` | View system audit logs |

### Inventory, sales & purchases
Inventory is separate from income/expense. Products are shared by every branch; stock is kept per branch at average cost, and everyone works only in their own branches (owner/admin see every branch).

| Code | Description |
|------|-------------|
| `inventory.view` | See products, stock levels and stock history of your branches |
| `inventory.manage` | Create and edit products and product categories (shared by every branch) |
| `inventory.adjust` | Add, remove or count stock in your branches, with a reason |
| `inventory.transfer` | Move stock between branches |
| `sales.view` | See sales of your branches |
| `sales.create` | Record sales; their stock goes out right away |
| `sales.edit` | Change a sale; its goods are put back and the new lines taken out of stock |
| `sales.void` | Void a sale and put its stock back |
| `purchases.view` | See purchases of your branches |
| `purchases.create` | Record purchases from suppliers; stock goes in once approved |
| `purchases.approve` | Approve or reject purchases; approved purchases add stock |
| `purchases.edit` | Change a pending purchase (an approved one also needs Approve Purchases) |
| `purchases.void` | Void a purchase and take its stock back out |
| `inventory-expenses.view` | See inventory expenses of your branches |
| `inventory-expenses.create` | Record inventory expenses (shipping, packaging, storage…) |
| `inventory-expenses.void` | Void an inventory expense |

Inventory expenses use their own categories (type `inv_expense`, managed with `inventory.manage` on Inventory → Expense categories) and come off the sales profit as net profit. Inventory reports (stock value, sales and profit, purchases, expenses, low stock) need both `reports.view` and `inventory.view`. Nobody approves a purchase they created; people with `purchases.approve` record their own purchases as approved unless they send them for approval.

### Platforms & weekly win/lose
In the PLATFORMS menu a platform is called an **agent** and a branch a **group** (the codes keep the old names). Win/lose is entered per group: people enter and see it only for their own groups (owner/admin see every group). An agent can be assigned to many groups and a group can have many agents, each assignment with its own partners (each at their own %) and currency.

| Code | Description |
|------|-------------|
| `platforms.view` | See agents and the groups they are in |
| `platforms.manage` | Add and edit agents and their currency |
| `platform-assignments.manage` | Make titled groups of agents with their rates; update, turn off or delete them |
| `winlose.view` | See the weekly win/lose entered for your groups |
| `winlose.create` | Enter a week's win/lose and commission for your groups (once a week) |
| `winlose.edit` | Change a week already saved, with a reason (kept in the audit log) |
| `winlose-reports.view` | The weekly report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | Combined totals across all your groups |
| `winlose-settlements.finalize` | Finalize or reopen a group's week; a finalized week can't be changed |

Win/lose is entered once a week (Monday to Sunday) per branch and platform. A saved week keeps the partners, partner %, currency and formula version it was worked out with; changing it needs `winlose.edit` and a reason (audit log). A finalized week can't be changed until someone with `winlose-settlements.finalize` reopens it. Reports without one branch chosen (all your branches together) need `winlose-reports.consolidated`. Owner/admin (Manage Branches) get these permissions automatically when they first ship.

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

Project root: [`permissions.catalog.json`](../permissions.catalog.json) (48 entries, versioned).

## Source of truth

- Backend constants: `backend/internal/permissions/permissions.go`
- Frontend constants: `frontend/utils/permissions.ts` (`P` + `PERMISSION_CATALOG`)
- API list: `GET /api/v1/permissions` (requires `roles.view`)
