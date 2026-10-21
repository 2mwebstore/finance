# Demo seed data

When `APP_ENV=development` (or `test`), the API runs `SeedDemoData` after permissions/categories.

## Enable / force

- Runs automatically in development if branch `PP-01` does not exist.
- Set `SEED_DEMO=true` for a forced demo seed. This restores the documented `246810` PIN for existing demo accounts as well as creating missing demo data.

## Demo users (PIN: `246810`)

| Username | Role | Branches |
|----------|------|----------|
| `manager1` | owner (all permissions) | PP-01 |
| `cashier1` | cashier | PP-01 |
| `accountant1` | accountant | PP-01, SR-01, BT-01 |
| `viewer1` | viewer | PP-01 |
| `manager_sr` | manager | SR-01 |

Super Admin is **not** created by demo seed — use `POST /api/v1/auth/setup` with `SETUP_TOKEN`.

To repair demo logins in an existing development database, restart the API once with `SEED_DEMO=true` in its environment. Remove the variable after the seed completes so demo PINs do not get reset on every restart.

Demo accounts created before PINs became 6 digits still sign in with `2468`; the app then asks for a new 6-digit PIN. A forced seed sets them back to `246810`.

## Demo content

- **Branches:** Phnom Penh HQ (`PP-01`), Siem Reap (`SR-01`), Battambang (`BT-01`)
- **Opening balances:** USD + KHR per branch (as of 2026-10-01)
- **Transactions:** ~12 DEMO-tagged income/expense rows (USD & KHR), including one **pending**
- **Cash reconciliation:** sample USD short for PP-01 today

## Reset tip

Drop/recreate the database, then restart the API with `APP_ENV=development`.
