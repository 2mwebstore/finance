# Role-based UI permissions

See **[docs/PERMISSIONS.md](../docs/PERMISSIONS.md)** for the full **30-code** reference table.

## Quick usage

```ts
import { P } from '~/utils/permissions'
const { can } = usePermissions()
can(P.IncomeCreate, P.ExpenseCreate)
```

```vue
<Can perm="reports.export">…</Can>
<Can :any="['income.create','expense.create']">…</Can>
```

Catalog array: `PERMISSION_CATALOG` in `utils/permissions.ts`.

## Pinia store + API sync

- Store: `stores/permissions.ts` (`usePermissionsStore`)
- Plugin: `plugins/permissions.client.ts` — on login, sets user grants from `/auth/me` and syncs catalog via `GET /permissions` when `roles.view` / super admin
- `usePermissions()` reads grants and catalog from the store
- Admin page **Permission codes** shows sync status and “Sync from API”

```ts
const store = usePermissionsStore()
store.userCodes          // granted codes
store.catalog            // full list (API or static)
await store.syncCatalogFromApi(api)
```

## Branch scope

`hasFullAccess` (from `usePermissions()`) is true for Super Admin and anyone with `branches.manage` (the owner and admin roles). Everyone else works at branch level, and the API enforces it:

- **Categories:** shared categories (`branch_id: null`) appear in every branch and only full-access users may create or edit them. Branch managers create and edit categories for their own branches. An expense can use a shared category or one from its own branch.
- **Users:** branch managers see users in their branches only, and can't manage owners/admins or anyone holding a permission they lack. They can only assign roles whose permissions they hold, and only their own branches.
- **Roles & permissions:** anyone with `roles.manage` can add roles. The owner/admin (full access) can change any role; others can give a role only permissions they have themselves, and can change or delete only non-system roles within those limits whose users are all in their branches.
- **Branch data:** everyone sees only the branches they are assigned to; data is never shared between branches. Only full access (`branches.manage`, owner/admin) sees every branch.
- **Branch Telegram:** people with `branches.edit` or `branches.manage` set up the Telegram chat (and optionally the branch's own bot) for the branches they see, and choose which transaction updates are sent (new, approved, rejected, voided, reversed). Without an own bot the default bot from the Backups page is used. Each branch's updates go only to its own chat; the bot token is never sent back to the browser.
- **Audit log:** branch managers with `audit-logs.view` see events from their branches and their branch users. Cleanup (`POST /audit-logs/cleanup`, keep 1–3 months) needs full access plus `settings.manage`.

## Managers (who approves whose transactions)

On **Users → Edit** (or when creating a user), pick **Managers**. When a user has managers, only those
managers, or the owner/admin/Super Admin, can approve or reject that user's pending transactions.
Users without managers keep the old rule: anyone with `transactions.approve` / `transactions.reject`
and access to the branch. Nobody can approve their own transaction or choose their own managers.

A manager must have approve or reject permission and work in one of the user's branches (owners/admins
always qualify). The transactions list returns `can_approve` / `can_reject` per row, so Approve/Reject
buttons only appear for people the API will accept.

Users without `transactions.approve` can't post approved transactions: whatever they send, the API saves
their new transactions as **pending**, and the Add form shows "Sent for approval" instead of the switch.
