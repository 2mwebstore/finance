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
| `winlose.react` | React to Data Entry | Win/Lose | Give an entry, or a group or agent in it (at 0 too), ✅ or ❌, on Data entry, the Report and Group summary |
| `winlose-reports.view` | View Report | Win/Lose Reports | The Report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | View Consolidated Win/Lose | Win/Lose Reports | Combined totals across all your groups |
| `winlose-reports.rates` | View Rates in Reports | Win/Lose Reports | Show each agent's rate on the Report and the Group Summary; without it the rates are hidden |

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

**An inactive branch** (Delete on Branches makes it inactive; nothing is removed) keeps its data,
and its data stays in the reports: it is still in the branch pickers, after the active ones and
marked **(inactive)**, for everyone who could see it before. Someone whose every branch is inactive
can't sign in (403 "your branch is inactive, so you can't sign in") and is signed out when the last
one goes inactive; someone who also works in an active branch signs in as before. The owner and
admins are never blocked. **Restore** on Branches makes it active again, and its people can sign in again.

**A branch picked in the top bar** (people with several branches) limits every page to that branch's
own data: lists and reports carry it as `branch_id`. Things that belong to no branch are left out
too: Branches lists only that branch, Roles only its own roles (no built-in roles: `GET /roles?branch_id=`
returns the branch's roles only), and Categories only its own categories (the shared ones still
appear in its forms). **All branches** shows everything again.

### Users
| Code | Description |
|------|-------------|
| `users.view` | View user list and profiles |
| `users.create` | Create new users |
| `users.edit` | Edit user profiles |
| `users.manage` | Manage user roles, passwords, and account activation |

**The user form** (Add user / Edit) has **Name**, **Branch**, **Username** and **Info** (optional,
anything about the person). The username is filled in from the Name and the branch, each in small
letters without spaces: Name `Dara` in Phnom Penh is `dara@phnompenh`, and that is what they sign
in with. No email is asked: the API stores the username as the email (`email` is optional in
`POST /users`; signing in takes the username or the email). On Edit the username follows the Name
and the branch once either is changed (someone in several branches goes by their default branch,
else the first), and the person then signs in with the new one; an email that was the old username
moves with it. A username can't be anyone else's username or email. People made before keep their
username until their Name or branch is changed. The top bar shows a person in one branch by name
with their username under it (others: their role).

The **Role** list in the user form shows the roles of the person's branches only; the built-in
system roles (owner, admin, manager, accountant, cashier, viewer) are not offered there, and
**Super Admin** is offered only to the Super Admin. Someone who already holds a system role keeps it
on Edit. To give a branch a role like a system one, the Super Admin clones it (below).

### Roles
| Code | Description |
|------|-------------|
| `roles.view` | View roles and their permissions |
| `roles.manage` | Create, edit, and delete non-system roles |
| `permissions.manage` | Assign permissions to roles |
| `permissions.view` | Show the Permission Codes page (menu item and page); without it the page is hidden |

**The Roles page** never lists the Super Admin role (`GET /roles` leaves it out; the user form asks
for it with `include_super_admin=true`, which only the Super Admin gets). The Super Admin sees the
built-in roles there, and a branch's own roles by choosing it in **Roles for** (or in the top bar).
Others see every role they may, or one branch's.

**Clone** (Roles & permissions, the Super Admin only): copies any role, system or a branch's, to the
branches picked. Each branch gets its own role with the same name, description and permissions
(code `<role>-<branch>`, e.g. `cashier-phnompenh`; `-2`, `-3`… if that is taken), which can then be
given to its people and changed on its own. A branch that already has a role of that name is
skipped. The Super Admin role itself can't be cloned.

**Delete** (a role nobody has) removes it from the database for good, with its permissions; it can't
be brought back, and its code can be used again. A role someone still has can't be deleted (take it
off them first), nor can a built-in role. API: `POST /api/v1/roles/:id/clone`
`{"branch_ids": [...]}` → `201 {"created": [roles], "skipped": [{id, name}]}`.

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
| `winlose.react` | Give an entry, or a group or agent in it (at 0 too), ✅ or ❌, on Data entry, the Report and Group summary |
| `winlose-reports.view` | The Report and the group summary for your groups, one group at a time |
| `winlose-reports.consolidated` | Combined totals across all your groups |
| `winlose-reports.rates` | Show each agent's rate on the Report and the Group Summary (hidden without it) |

Win/lose is entered once a week (Monday to Sunday) per branch and platform. A saved week keeps the partners, partner % and currency it was worked out with and follows the current formula and its agents' current rates; changing it needs `winlose.edit` and a reason (audit log). There is no finalizing: every week can be changed. Finalize Weekly Settlements (`winlose-settlements.finalize`) is gone: the API takes it off every role when it starts. Reports without one branch chosen (all your branches together) need `winlose-reports.consolidated`. Owner/admin (Manage Branches) get these permissions automatically when they first ship.

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
