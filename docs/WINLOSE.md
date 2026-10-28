# Platforms & weekly win/lose

Once a week, each HQ enters every platform's win/lose and commission; the API works out the
**Total** (the spreadsheet's Jame-PS) and the reports add it up by HQ, platform, weeks and currency.

**An HQ is a branch.** HQ001, HQ002… are branches (Manage → Branches), so creating, editing,
activating and deactivating HQs, assigning users to them, and HQ-level access all reuse the
existing branch features. People see and enter data only for the branches they are assigned to;
owner/admin (Manage Branches) see every branch. This is enforced in the API, not just the UI.

## Data model (migration `000013_platforms`)

| Table | What it holds |
|---|---|
| `platforms` | Code (unique, e.g. `LCMMJB`, `12K_YMLLJB_V1`), name, default rate, default currency, active. |
| `hq_platform_assignments` | A platform given to an HQ (branch) with its own rate and currency, first and last day. Many platforms per HQ and many HQs per platform. |
| `weekly_platform_results` | One row per week (Monday–Sunday), HQ and platform: win/lose, commission, Total (`jame_ps`), **plus the rate, currency and formula version used**. |
| `weekly_settlements` | A finalized (or reopened) week for one HQ. |
| `weekly_settlement_totals` | The finalized week's totals per currency. |

- Rates are **percents**: `35` means 35% (`DECIMAL(7,4)`). Amounts are `DECIMAL(19,4)` in the
  assignment's currency; USD and KHR are never added together.
- In the database and API the Total is `jame_ps` (the spreadsheet's name); the app shows it as
  "Total". `adjusted_amount` (win/lose + commission) is kept with each row but not shown.
- **One active assignment per HQ and platform:** `active_key` is 1 while active and NULL once
  ended, with a unique key on `(branch_id, platform_id, active_key)`. The service also refuses an
  assignment whose days would overlap an earlier one for the same HQ and platform.
- **One entry per week:** unique key on `(week_start, branch_id, platform_id)`. Picking any day of
  a week opens that week (its Monday).
- A platform the HQ had on any day of a week is on that week's sheet (assigned on a Wednesday, or
  moved mid-week: the week is on both HQs' sheets).
- **History is kept:** ending or moving an assignment sets its last day and keeps the row; weeks
  already entered keep their own rate, currency and formula version when a rate or the formula
  changes.
- Foreign keys to `branches`, `users`, `platforms`, `hq_platform_assignments` and
  `weekly_settlements`; indexes on week and HQ. The tables take the collation of `branches.id`
  (the SQL migrations use `utf8mb4_unicode_ci`; a database built by AutoMigrate on MySQL 8 uses
  `utf8mb4_0900_ai_ci`), since a foreign key needs the same collation on both sides.
- `LCMMJB` and `12K_YMLLJB_V1` are added when the `platforms` table is first created, with rate 0:
  set their rates on the Platforms page and add the other 12K_YMLLJB platforms there.

The same SQL is in `backend/internal/database/platforms.go`, which the API runs on start
(`EnsureRequiredTables`): it only creates missing tables and adds missing foreign keys (a foreign key
that can't be added is logged and skipped so the API still starts). A test keeps the two in step.

> **Earlier test builds** created a `daily_platform_results` table. Daily entry is gone; that table is
> no longer used and can be dropped by hand once you have checked it is empty
> (`SELECT COUNT(*) FROM daily_platform_results;`). The API never drops it.

## Calculation

All of it is in `backend/internal/winlose` (one place; the entry page's live Total comes from the
API's preview endpoint, not a copy in the browser).

Formula **v1** — read from the spreadsheet, **still to be confirmed by the business**:

```
Total = ROUND((win_lose + commission) × rate%, 0)     -- half away from zero, like Excel's ROUND
```

To change it: add a new version in `internal/winlose/calc.go` (keep the old one), then set the app
setting `winlose.formula` to the new version (`PUT /api/v1/settings`, needs `settings.manage`).
New weeks use the new version; saved weeks keep theirs (and a correction to a saved week is worked
out with that week's own version and rate). The entry page shows "formula still to be confirmed"
until a version is marked `Confirmed`.

## Permissions

| Code | Allows |
|---|---|
| `platforms.view` | See platforms and HQ assignments |
| `platforms.manage` | Add/edit platforms, default rate and currency |
| `platform-assignments.manage` | Assign platforms to HQs (many to many at once); change rate, end or move assignments |
| `winlose.view` | See the weekly entries for your HQs |
| `winlose.create` | Enter new weeks |
| `winlose.edit` | Correct a saved week (reason required; old and new figures go to the audit log) |
| `winlose-reports.view` | Weekly report and HQ summary — one HQ at a time |
| `winlose-reports.consolidated` | All your HQs together (Main HQ view) |
| `winlose-settlements.finalize` | Finalize or reopen an HQ's week (reason required to reopen) |

Owner/admin roles (those with Manage Branches) get all nine when the API first starts with this
version. Default roles for new installs: manager (view, enter, correct, reports), accountant (view,
enter, reports), viewer (view, reports). Give other roles access on Roles & permissions.

## API (`/api/v1`)

| Method & path | Permission |
|---|---|
| `GET /platforms` · `POST /platforms` · `PATCH /platforms/:id` | view / manage |
| `GET /platform-assignments?branch_id&platform_id&status=active\|ended\|all` | `platforms.view` or assignments |
| `POST /platform-assignments` `{platform_ids[], branch_ids[], commission_rate?, currency?, starts_on?}` — every platform to every HQ, all or nothing; empty rate/currency = each platform's default | assignments |
| `PATCH /platform-assignments/:id` `{commission_rate, currency}` | assignments |
| `POST /platform-assignments/:id/end` `{ends_on?, reason?}` · `/:id/move` `{branch_id, on?, reason?}` | assignments |
| `GET /winlose/formula` | any win/lose permission |
| `GET /winlose/entry?week_start&branch_id` (any day of the week) | view / create / edit |
| `POST /winlose/entry/preview` · `PUT /winlose/entry` `{week_start, branch_id, rows[], reason?}` | create / edit |
| `GET /winlose/reports?date_from&date_to&branch_id?` — weeks starting in the range (from the Monday of `date_from`) | reports (no `branch_id` = all HQs, needs consolidated) |
| `GET /winlose/settlements?week_start&branch_id?` · `POST /winlose/settlements` · `POST /winlose/settlements/:id/reopen` | reports / finalize |

Rules the API enforces: a week that hasn't started can't be entered; a finalized week can't be
changed; a week can be finalized from its Sunday; ends and moves can't hide weeks already entered.

## Pages

- Platforms → **Win/Lose Entry** (one week at a time, ‹ › to move between weeks),
  **Weekly Report** (with weekly settlements), **HQ Summary** (consolidated and by-platform totals
  for Main HQ), **Platforms**, **HQ Platform Assignments** (Assign platforms: several platforms to
  several HQs at once)
- Manage → **Branches** is HQ management (unchanged)
- Reports export **CSV** (UTF-8 with BOM, opens in Excel), like the other reports.

## Deploying

No new environment variables.

1. Back up the database (Manage → Backups).
2. Either apply `backend/migrations/000013_platforms.up.sql` (safe to run more than once), or let the
   API create the tables on start. Both only add tables, keys and the two platforms; nothing existing
   is changed.
3. Deploy the API and frontend as usual. On start the API adds the nine permissions and gives them
   to owner/admin.
4. On Platforms: set the rates of `LCMMJB` and `12K_YMLLJB_V1`, add the other 12K_YMLLJB platforms.
5. On HQ Platform Assignments: assign platforms to HQs.
6. **Confirm the formula** with real figures before relying on the reports (see Calculation).

`000013_platforms.down.sql` drops the five tables and every result in them — only for rolling back a
fresh install.

## Tests

- `go test ./internal/winlose/` — the formula and rounding.
- `go test ./internal/database/` — migration 000013 matches the startup SQL.
- Database tests (many-to-many assignments, duplicate prevention, history on end/move, rate
  snapshots, corrections, HQ access, week ranges, currency separation, settlements) run against a
  scratch MySQL database whose name contains `test`:

  ```
  TEST_MYSQL_DSN='user:pw@tcp(127.0.0.1:3306)/finance_test?charset=utf8mb4&parseTime=True&loc=Asia%2FPhnom_Penh' \
    go test ./internal/services/ -run 'Platform|WinLose|Weekly|Assignment'
  ```

  Without `TEST_MYSQL_DSN` they are skipped.
