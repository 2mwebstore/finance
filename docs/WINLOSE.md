# Platforms & weekly win/lose

Once a week, each branch enters every platform's win/lose and commission; the API works out the
**Total** (the spreadsheet's Jame-PS) and the reports add it up by branch, platform, weeks and
currency.

**Branches are the "HQs"** of the original spec (HQ001, HQ002…): Manage → Branches creates, edits,
activates and deactivates them, assigns users, and gives branch-level access. People see and enter
data only for the branches they are assigned to; owner/admin (Manage Branches) see every branch.
This is enforced in the API, not just the UI. In the code and database an assignment is still an
`hq_platform_assignment` and reports return `hqs`.

**Names in the app:** in the PLATFORMS menu a platform is called an **agent** and a branch a
**group** (Manage → Branches still says branch). The code, database, API and page URLs keep
`platform` and `branch` (e.g. `/admin/platforms` is the Agents page, `/winlose/entry` is Data Entry).

## Data model (migration `000013_platforms`)

| Table | What it holds |
|---|---|
| `platforms` | An agent: code (shown as **User**; unique and required, e.g. `LCMMJB`, `12K_YMLLJB_V1`), name (optional), default currency, active. `default_rate` is still the % for assignments without partners, but the app no longer shows or sets it (new agents get 0). |
| `partners` | Name (unique), phone, note, default partner % (`default_rate`), active. |
| `hq_platform_assignments` | A platform given to a branch with its own partner % (`commission_rate`, used when it has no partners), currency, first and last day. Many platforms per branch and many branches per platform. |
| `assignment_groups` | A **titled group** of agents (the Group Agent Assignments page) in one branch: title, currency (empty = each agent's default), first day, active (on/off). Its agents are the `hq_platform_assignments` with its `group_id`. |
| `assignment_partners` | An assignment's partners, each with their own % (`commission_rate`). Any number per assignment; a partner at most once. |
| `weekly_platform_results` | One row per week (Monday–Sunday), branch and platform: win/lose, commission, Total (`jame_ps`), **plus the partner %, currency and formula version used**. |
| `weekly_partner_results` | Each partner's share of a saved week: their % and their Total (`jame_ps`), as they were when the week was entered. |
| `weekly_entries` | Data Entry's entries: a title for one group's (branch's) week, who made it and when. One per group and week; its amounts are that group's `weekly_platform_results` for the week. |
| `weekly_settlements` | A finalized (or reopened) week for one branch. |
| `weekly_settlement_totals` | The finalized week's totals per currency. |

- The **partner %** is a percent: `35` means 35% (`DECIMAL(7,4)`; stored as `commission_rate` / `default_rate`). Amounts are `DECIMAL(19,4)` in the
  assignment's currency; USD and KHR are never added together.
- In the database and API the Total is `jame_ps` (the spreadsheet's name); the app shows it as
  "Total". `adjusted_amount` (win/lose + commission) is kept with each row but not shown.
- **Partners:** an assignment can have several partners, each with their own %. Assigning gives
  every chosen partner their default %; the % can be changed per assignment afterwards (the
  **Partners** button). The partners' % may add up to at most 100%. With no partners, the Total is
  at the assignment's own % (the platform's default when assigned). Changing an assignment's partners
  applies to new weeks; saved weeks keep the partners and % they were entered with
  (`weekly_partner_results`). A partner on an active assignment can't be turned off.
- **Upgrading from the one-partner test build:** that build had a `partner_id` column on
  `hq_platform_assignments` and `weekly_platform_results`. When the API creates the two new tables it
  copies those partners into them once (with their %); the old columns are then no longer used and
  are left in place.
- **One active assignment per branch and platform:** `active_key` is 1 while active and NULL once
  ended, with a unique key on `(branch_id, platform_id, active_key)`. The service also refuses an
  assignment whose days would overlap an earlier one for the same branch and platform.
- **One entry per week:** unique key on `(week_start, branch_id, platform_id)`. Picking any day of
  a week opens that week (its Monday).
- A platform the branch had on any day of a week is on that week's sheet (assigned on a Wednesday,
  or moved mid-week: the week is on both branches' sheets).
- **History is kept:** ending or moving an assignment sets its last day and keeps the row (a move
  takes the partners and % along); weeks already entered keep their own partners, %, currency and
  formula version when the partners, a % or the formula change.
- Foreign keys to `branches`, `users`, `platforms`, `hq_platform_assignments` and
  `weekly_settlements`; indexes on week and branch. The tables take the collation of `branches.id`
  (the SQL migrations use `utf8mb4_unicode_ci`; a database built by AutoMigrate on MySQL 8 uses
  `utf8mb4_0900_ai_ci`), since a foreign key needs the same collation on both sides.
- `LCMMJB` and `12K_YMLLJB_V1` are added when the `platforms` table is first created; add the other
  12K_YMLLJB agents on the Agents page.
- **Titled groups:** Assign agents makes a group: a title, its agents, partners (at their default %),
  currency and first day; all its agents get the same partners. An agent can be in one group at a
  time. **Update** changes the title, agents (one taken out is ended if weeks were entered for it,
  else removed; one put in starts today), partners and their %, and currency. **Active** off takes
  the group's agents off Data Entry (weeks already saved stay; they stay in the group). **Delete**
  works only while no week has been entered for the group (its assignments go for good); after
  that, turn it off. When `assignment_groups` is first created, every active assignment becomes a
  group of its own titled with its agent's code (once); ended ones stay without a group.
- **Branches are kept behind the scenes:** no PLATFORMS page asks for or shows a branch. Each person
  works in their own group: their default branch (Manage → Users), else their only branch; without
  either the API refuses with a message. New entries, assignments and settlements go there, and
  editing an entry keeps its group. The reports cover the own group, or every group in the
  person's scope for those with View Consolidated Win/Lose, added up by agent (groups aren't shown
  apart). The API still takes `branch_id` where noted below, for other tools.
- When `weekly_entries` is first created, every week already saved gets an entry titled
  "Week of …" (once).

The same SQL is in `backend/internal/database/platforms.go`, which the API runs on start
(`EnsureRequiredTables`): it only creates missing tables and adds missing foreign keys (a foreign key
that can't be added is logged and skipped so the API still starts). A test keeps the two in step.

> **Earlier test builds** created a `daily_platform_results` table. Daily entry is gone; that table is
> no longer used and can be dropped by hand once you have checked it is empty
> (`SELECT COUNT(*) FROM daily_platform_results;`). The API never drops it.

## Calculation

All of it is in `backend/internal/winlose` (one place, on the server). The entry page only takes
Win/Lose and Commission; the Total and each partner's share are worked out when the week is saved
and shown in the reports. (`POST /winlose/entry/preview` still returns them without saving.)

Formula **v1** — read from the spreadsheet, **still to be confirmed by the business**:

```
Total = ROUND((win_lose + commission) × partner %, 0)     -- half away from zero, like Excel's ROUND
```

With several partners, each partner's share is worked out at their own %, and the platform's Total
is the sum of the shares (each share is rounded on its own). Example: win/lose 1,000 and commission
200, Sok Dara at 40% and Chan Lina at 10%: shares 480 and 120, Total 600 (shown at 50% in all).

To change it: add a new version in `internal/winlose/calc.go` (keep the old one), then set the app
setting `winlose.formula` to the new version (`PUT /api/v1/settings`, needs `settings.manage`).
New weeks use the new version; saved weeks keep theirs (and a correction to a saved week is worked
out with that week's own version, partners and %). `GET /winlose/formula` reports a version as not
`confirmed` until it is marked `Confirmed`.

## Permissions

| Code | Allows |
|---|---|
| `platforms.view` | See platforms, partners and branch assignments |
| `platforms.manage` | Add/edit platforms and partners, default partner % and currency |
| `platform-assignments.manage` | Assign platforms to branches (many to many at once, with partners); change partners and their %, end or move assignments |
| `winlose.view` | See the weekly entries for your branches |
| `winlose.create` | Enter new weeks |
| `winlose.edit` | Correct a saved week (reason required; old and new figures go to the audit log) |
| `winlose-reports.view` | Weekly report and branch summary — one branch at a time |
| `winlose-reports.consolidated` | All your branches together |
| `winlose-settlements.finalize` | Finalize or reopen a branch's week (reason required to reopen) |

Owner/admin roles (those with Manage Branches) get all nine when the API first starts with this
version. Default roles for new installs: manager (view, enter, correct, reports), accountant (view,
enter, reports), viewer (view, reports). Give other roles access on Roles & permissions.

## API (`/api/v1`)

| Method & path | Permission |
|---|---|
| `GET /platforms` · `POST /platforms` · `PATCH /platforms/:id` | view / manage |
| `GET /partners` · `POST /partners` · `PATCH /partners/:id` `{name, phone?, note?, default_rate, is_active?}` | view (also assignments, reports) / `platforms.manage` |
| `GET /platform-groups?branch_id?&platform_id&status=all\|active\|off` (no `branch_id` = your own branch) · `POST /platform-groups` `{title, platform_ids[], partner_ids[]?, currency?, starts_on?}` · `PATCH /platform-groups/:id` `{title, platform_ids[], partners: [{partner_id, commission_rate}], commission_rate?, currency?}` · `POST /platform-groups/:id/active` `{is_active}` · `DELETE /platform-groups/:id` (only with no weeks entered) | `platforms.view` or assignments / assignments |
| `GET /platform-assignments?branch_id?&platform_id&status=active\|ended\|all` (no `branch_id` = your own group) | `platforms.view` or assignments |
| `POST /platform-assignments` `{platform_ids[], branch_ids[]?, partner_ids[]?, currency?, starts_on?}` — every platform to every branch (none = your own group) with the same partners (each at their default %), all or nothing; empty currency = each platform's default | assignments |
| `PATCH /platform-assignments/:id` `{partners: [{partner_id, commission_rate}], commission_rate?, currency}` — replaces the partners; `commission_rate` is the assignment's own % (used with no partners; empty keeps it) | assignments |
| `POST /platform-assignments/:id/end` `{ends_on?, reason?}` · `/:id/move` `{branch_id, on?, reason?}` (the app no longer moves) | assignments |
| `GET /winlose/formula` | any win/lose permission |
| `GET /winlose/entries?search` — the entries of your groups, newest first (title, week, created at) | view / create / edit |
| `GET /winlose/entry?week_start&branch_id?` (any day of the week; no `branch_id` = your own group) — includes the week's `entry` | view / create / edit |
| `POST /winlose/entry/preview` · `PUT /winlose/entry` `{week_start, branch_id?, title, rows[], reason?}` | create / edit |
| `GET /winlose/reports?date_from&date_to&branch_id?&partner_id?&by_week?` — weeks starting in the range (from the Monday of `date_from`); lines per agent and currency across the groups covered, with each partner's share (`partners`), totals per agent, partner and currency; with `partner_id` only that partner's weeks, the Total being their share; `by_week=true` gives a line per week (partner statement) | reports (no `branch_id` = every group in scope with consolidated, else your own group) |
| `GET /winlose/settlements?week_start&branch_id?` · `POST /winlose/settlements` `{week_start, branch_id?}` · `POST /winlose/settlements/:id/reopen` (no `branch_id` = your own group) | reports / finalize |

Rules the API enforces: a week that hasn't started can't be entered; a finalized week can't be
changed; a week can be finalized from its Sunday; ends and moves can't hide weeks already entered.

## Pages

- PLATFORMS → **Data Entry** (`/winlose/entry`: a list of entries with Created at, Title and the
  actions **View** (a window with the entry's details and saved amounts) and **Update**; **New
  entry** / Update opens the week, a title and every agent's Win/Lose and Commission),
  **Weekly Report** (by agent, with the own group's weekly settlement), **Group Summary**
  (`/winlose/reports/branch-summary`; totals, by agent and by partner), **Partner Statement** (one
  partner's weeks across agents), **Agents** (`/admin/platforms`), **Partners**, **Group Agent
  Assignments** (`/admin/platform-assignments`: your branch's titled groups of agents, each with
  an **Active** switch, **Update** and **Delete**; **Assign agents** makes a new group).
  The amounts are formatted money inputs (a minus for a loss); a new title or new amounts need
  Enter Win/Lose, changing a saved entry's title or amounts Correct Win/Lose (amounts also a
  reason). The Weekly Report shows each line's Total and partner shares, and it
  and the Group Summary filter by partner and total by partner.
- Manage → **Branches** manages the branches themselves. New branches get their code automatically
  (`BR-001`, `BR-002`…, one more than the highest BR-number ever used); codes can't be typed or
  changed in the app. Codes made before (e.g. `PP-01`) stay.
- Reports export **CSV** (UTF-8 with BOM, opens in Excel), like the other reports.

## Deploying

No new environment variables.

1. Back up the database (Manage → Backups).
2. Either apply `backend/migrations/000013_platforms.up.sql` (safe to run more than once), or let the
   API create the tables on start. Both only add tables, keys and the two platforms; nothing existing
   is changed.
3. Deploy the API and frontend as usual. On start the API adds the nine permissions and gives them
   to owner/admin. It also refreshes the permissions' names and descriptions to the catalog's
   (e.g. "View Platforms" becomes "View Agents").
4. On Agents: add the other 12K_YMLLJB agents (User required, name optional).
5. On Partners: add your partners with their default %.
6. On Group Agent Assignments: assign agents to groups, choosing their partners; use
   **Partners** on a row to change a partner's % for that assignment.
7. **Confirm the formula** with real figures before relying on the reports (see Calculation).

`000013_platforms.down.sql` drops the eight tables and every result in them — only for rolling back a
fresh install.

## Tests

- `go test ./internal/winlose/` — the formula and rounding.
- `go test ./internal/database/` — migration 000013 matches the startup SQL.
- Database tests (many-to-many assignments, duplicate prevention, history on end/move, partner %
  snapshots, several partners per assignment and their shares, corrections, branch access, week ranges, currency separation, settlements) run against a
  scratch MySQL database whose name contains `test`:

  ```
  TEST_MYSQL_DSN='user:pw@tcp(127.0.0.1:3306)/finance_test?charset=utf8mb4&parseTime=True&loc=Asia%2FPhnom_Penh' \
    go test ./internal/services/ -run 'Platform|WinLose|Weekly|Assignment'
  ```

  Without `TEST_MYSQL_DSN` they are skipped.
