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
| `platforms.view` | View Agents | Platforms | See the agents, their currency and how many groups have them |
| `platforms.create` | Add Agents | Platforms | Add agents with their currency |
| `platforms.edit` | Update Agents | Platforms | Edit agents; turn them off (with a reason) or on |
| `platforms.delete` | Delete Agents | Platforms | Delete an agent that was never in a group or entered (a used one can only be turned off) |
| `platform-assignments.view` | View Group Agent Assignments | Platforms | See the titled groups with their agents and rates |
| `platform-assignments.create` | Add Group Agent Assignments | Platforms | Make a titled group of agents with their rates (Assign agents) |
| `platform-assignments.edit` | Update Group Agent Assignments | Platforms | Change a group's title, agents and rates; turn it off or on |
| `platform-assignments.delete` | Delete Group Agent Assignments | Platforms | Delete a group no week was entered for (a used one can only be turned off) |
| `winlose.view` | View Data Entry | Win/Lose | See the entries and the W/L and COMM entered for your groups |
| `winlose.create` | Add Data Entry | Win/Lose | Make new entries and enter agents' W/L and COMM for your groups |
| `winlose.edit` | Update Data Entry | Win/Lose | Change an entry's title and amounts already saved, with a reason (kept in the audit log) |
| `winlose.delete` | Delete Data Entry | Win/Lose | Delete an entry and its amounts, with a reason (kept in the audit log) |
| `winlose.react` | React to Data Entry | Win/Lose | Give an entry, or an agent in it, ✅ or ❌, on Data entry and the Report |
| `winlose-reports.view` | View Report | Win/Lose Reports | The Report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | View Consolidated Win/Lose | Win/Lose Reports | Combined totals across all your groups |
| `winlose-reports.rates` | View Rates in Reports | Win/Lose Reports | Show each agent's rate on the Report and the Group Summary; without it the rates are hidden |
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
In the RP menu (once called PLATFORMS) a platform is called an **agent** and a branch a **group** (the codes keep the old names). Win/lose is entered per group: people enter and see it only for their own groups (owner/admin see every group). An agent can be assigned to many groups and a group can have many agents, each assignment with its own partners (each at their own %) and currency.

| Code | Description |
|------|-------------|
| `platforms.view` | See the agents, their currency and how many groups have them |
| `platforms.create` | Add agents with their currency |
| `platforms.edit` | Edit agents; turn them off (with a reason) or on |
| `platforms.delete` | Delete an agent that was never in a group or entered (a used one can only be turned off) |
| `platform-assignments.view` | See the titled groups with their agents and rates |
| `platform-assignments.create` | Make a titled group of agents with their rates (Assign agents) |
| `platform-assignments.edit` | Change a group's title, agents and rates; turn it off or on |
| `platform-assignments.delete` | Delete a group no week was entered for (a used one can only be turned off) |
| `winlose.view` | See the entries and the W/L and COMM entered for your groups |
| `winlose.create` | Make new entries and enter agents' W/L and COMM for your groups |
| `winlose.edit` | Change an entry's title and amounts already saved, with a reason (kept in the audit log) |
| `winlose.delete` | Delete an entry and its amounts, with a reason (kept in the audit log) |
| `winlose.react` | Give an entry, or an agent in it, ✅ or ❌, on Data entry and the Report |
| `winlose-reports.view` | The Report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | Combined totals across all your groups |
| `winlose-reports.rates` | Show each agent's rate on the Report and the Group Summary (hidden without it) |
| `winlose-settlements.finalize` | Finalize or reopen a group's week; a finalized week can't be changed |

Win/lose is entered once a week (Monday to Sunday) per branch and platform. A saved week keeps the partners, partner %, currency and formula version it was worked out with; changing it needs `winlose.edit` and a reason (audit log). A finalized week can't be changed until someone with `winlose-settlements.finalize` reopens it. Reports without one branch chosen (all your branches together) need `winlose-reports.consolidated`. Owner/admin (Manage Branches) get these permissions automatically when they first ship.

Each RP page has its own **view, add, update and delete** (Data entry also **react**). They took
over from Manage Agents (`platforms.manage`) and Manage Group Agent Assignments
(`platform-assignments.manage`), which are gone: when the API first starts with them, every role
that had an old permission gets the ones that replace it (Manage Agents → add, update and delete
agents; Manage Group Agent Assignments → add, update and delete groups; View Agents → view groups;
Correct Win/Lose → delete entries; any Data entry or Report permission → react), and then the old
ones are removed. Nobody loses what they could do; take permissions off on Roles & permissions.

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
