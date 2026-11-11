# Platforms & weekly win/lose

Once a week, each branch enters every platform's win/lose and commission; the API works out the
**Total** (the spreadsheet's Jame-PS) and the reports add it up by branch, platform, weeks and
currency.

**Branches are the "HQs"** of the original spec (HQ001, HQ002…): Manage → Branches creates, edits,
activates and deactivates them, assigns users, and gives branch-level access. People see and enter
data only for the branches they are assigned to; owner/admin (Manage Branches) see every branch.
This is enforced in the API, not just the UI. In the code and database an assignment is still an
`hq_platform_assignment` and reports return `hqs`.

**Names in the app:** in the RP menu (once called PLATFORMS) a platform is called an **agent** and a branch a
**group** (Manage → Branches still says branch). The code, database, API and page URLs keep
`platform` and `branch` (e.g. `/admin/platforms` is the Agents page, `/winlose/entry` is Data Entry).

## Data model (migration `000013_platforms`)

| Table | What it holds |
|---|---|
| `platforms` | An agent: code (shown as **User**; unique and required, e.g. `LCMMJB`, `12K_YMLLJB_V1`), name (optional), default currency, active, **branch** (`branch_id`; empty = a shared agent, see Branch scope). `default_rate` is still the % for assignments without partners, but the app no longer shows or sets it (new agents get 0). |
| `partners` | Name (unique), phone, note, default partner % (`default_rate`), active. |
| `hq_platform_assignments` | A platform given to a branch with its own partner % (`commission_rate`, used when it has no partners), currency, first and last day. Many platforms per branch and many branches per platform. |
| `assignment_groups` | A **titled group** of agents (the Group Agent Assignments page) in one branch: title, currency (empty = each agent's default), first day, active (on/off). Its agents are the `hq_platform_assignments` with its `group_id`; an agent can be in several groups, at its own rate in each. |
| `assignment_partners` | An assignment's partners, each with their own % (`commission_rate`). Any number per assignment; a partner at most once. |
| `weekly_platform_results` | One row per entry and agent (`entry_id`, `platform_id`): win/lose, commission, Total (`jame_ps`: its groups' Totals added up, 0 for an agent in no group), **plus the rates, currency and formula version used**. `assignment_id` is only set on weeks saved before entries could hold any agent. |
| `weekly_group_results` | A result's part in each group the agent was in that week: the group, the assignment, its rate, the agent's type (`calc_type`) and the Total at it. An agent in two groups counts in both. |
| `weekly_partner_results` | Each partner's share of a saved week: their % and their Total (`jame_ps`), as they were when the week was entered. |
| `weekly_entries` | Data Entry's entries: a title for one branch's week, who made it and when. A week can have several entries; each one's amounts are the `weekly_platform_results` with its `entry_id`. |
| `weekly_entry_reactions` | People's ✅ (`yes`) or ❌ (`no`) on an entry: one per person and entry (`uq_weekly_entry_reaction`), removed with the entry. |
| `weekly_result_reactions` | The same on each agent in an entry (its `weekly_platform_results` row): one per person and result (`uq_weekly_result_reaction`); a correction keeps them, deleting the entry removes them. |
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
- **One active assignment per group and agent:** `active_key` is 1 while active and NULL once
  ended, with a unique key on `(group_id, platform_id, active_key)`; an agent can be in several
  groups of a branch. The service also refuses an assignment whose days would overlap an earlier
  one for the same group and agent (with no group: branch and agent).
- **One result per entry and agent:** unique key on `(entry_id, platform_id)`. **New entry** always
  makes a new entry (its first save creates it, the saves after change only it), so a week can
  have several entries; reports add them all up. Picking any day of a week opens that week (its
  Monday). Databases built before this get the new keys (and lose the old `uq_hq_platform_active`,
  `uq_weekly_platform_result` and `uq_weekly_entry`) on the next start; each saved result is put on
  its week's entry and gets its group's share in `weekly_group_results`.
- Data Entry lists **every agent that is on**, once, whatever its groups. A result counts in every
  group (switched on) the agent was in on any day of the week, at that group's rate; an assignment
  made before groups counts only when the agent is in no group that week.
- **History is kept:** ending or moving an assignment sets its last day and keeps the row (a move
  takes the partners and % along); weeks already entered keep their own partners, %, currency and
  formula version when the partners, a % or the formula change.
- Foreign keys to `branches`, `users`, `platforms`, `hq_platform_assignments` and
  `weekly_settlements`; indexes on week and branch. The tables take the collation of `branches.id`
  (the SQL migrations use `utf8mb4_unicode_ci`; a database built by AutoMigrate on MySQL 8 uses
  `utf8mb4_0900_ai_ci`), since a foreign key needs the same collation on both sides.
- `LCMMJB` and `12K_YMLLJB_V1` are added when the `platforms` table is first created; add the other
  12K_YMLLJB agents on the Agents page. **Turn off** asks for a reason (at least 3 characters; the
  audit log keeps it with `platform.save`) and is refused while an active group still has the agent;
  **Turn on** needs none.
- **Titled groups:** Assign agents (its own page) makes a group: a title, its agents each with its
  own **rate** (the Total is worked out at it), and its first day. There is no currency choice:
  each agent's amounts are in its own currency (Agents page). An agent can be in
  several groups, at its own rate in each. **Update** (its own page) changes the title, agents (one taken out is ended if
  weeks were entered for it, else removed; one put in starts today) and their rates,
  and takes off any partners a group had from before (weeks already saved keep their shares). **Active** off: new
  results no longer count in the group (weeks already saved stay; its agents stay in it). **Delete**
  works only while no week has been entered for the group (its assignments go for good); after
  that, turn it off. When `assignment_groups` is first created, every active assignment becomes a
  group of its own titled with its agent's code (once); ended ones stay without a group.
- **Choosing the branch:** a person who works in **one** branch never sees a branch choice: the API
  works in it for them. With **several** (owners/admins: all), Data Entry, Report, Group Summary and
  Group Agent Assignments each have a **Branch** select, chosen once and kept for the four pages
  (and Assign agents / a new entry use it for the new group or entry, with their own Branch select). The
  choice is **All branches** (every branch in the person's scope; for the Report only with View
  Consolidated Win/Lose) or one branch. A group, entry, settlement or report is always one branch's;
  the API checks every `branch_id` against the person's scope. With no `branch_id` the API works in the
  person's own group (their default branch, else their only one), as before.
- When `weekly_entries` is first created, every week already saved gets an entry titled
  "Week of …" (once).

The same SQL is in `backend/internal/database/platforms.go`, which the API runs on start
(`EnsureRequiredTables`): it only creates missing tables and adds missing foreign keys (a foreign key
that can't be added is logged and skipped so the API still starts). A test keeps the two in step.

> **Earlier test builds** created a `daily_platform_results` table. Daily entry is gone; that table is
> no longer used and can be dropped by hand once you have checked it is empty
> (`SELECT COUNT(*) FROM daily_platform_results;`). The API never drops it.

## Branch scope (data kept apart by branch)

Everything in RP belongs to a branch, and a person works only in the branches they are assigned to
(owners/admins, with Manage Branches, in every branch). The API enforces it; the pages just follow.

| What | Rule |
|---|---|
| Groups, assignments, entries, results, settlements, reports | In one branch; a person sees and changes only their branches' (`branch_id` on each row). |
| **Agents** | Belong to a branch (`platforms.branch_id`). A person sees their branches' agents and the **shared** ones (no branch); an owner/admin sees all. A new agent goes to the branch chosen, else the person's own group; one of another branch is refused (403). |
| Changing or deleting an agent | Only in a branch the person works in. A shared agent only by an owner/admin. Another branch's agent is "not found" (404). |
| Moving an agent to another branch | Owner/admin only, and only while no other branch's group has had it. |
| Putting agents in a group | Only agents of the group's branch and shared ones (`… belongs to another branch` otherwise). |
| Data Entry | Lists only the entry's branch's agents and the shared ones. |

Agents made before this: one used by groups of only one branch went to that branch (once, when
`platforms.branch_id` was added); one used in several branches' groups, or none, stays **shared**
until an owner/admin gives it a branch (Agents → Edit → Branch). The agent's code stays unique across
all branches. `GET /platforms` takes `branch_id` (one branch's agents) and `assignable=true` (the
agents that can go into the person's own group); each row has `branch_id` and `branch_code`. A person
with one branch sees no branch choice anywhere: it is picked for them.

## Calculation

All of it is in `backend/internal/winlose` (one place, on the server). The entry page only takes
Win/Lose and Commission; the Total and each partner's share are worked out when the week is saved
and shown in the reports. (`POST /winlose/entry/preview` still returns them without saving.)

Formula **v4** (the default, the business's rule) adds the commission to the win/lose, and the
agent's rate in a group is the agent's own share: the Total is at what is left of 100%.

```
Total = ROUND((win_lose + commission) × (100% − rate), 0)     -- half away from zero, like Excel's ROUND
e.g.    (1,000 + 100) × (100% − 60%) = 440                    -- an agent at 60%: the Total is at 40%
```

So an agent at 0% leaves the whole amount, and at 100% nothing. An agent in two groups has a Total
in each (at each group's rate); its own Total is their sum.

**Agent type.** Assign agents asks for a type for each agent, kept on its assignment in the group (`hq_platform_assignments.calc_type`, default `PO`), so agents in one group can differ:

| Type | Total (v4) |
|---|---|
| **PO** | `(W/L + Comm) × (100% − Rate)`, the formula above |
| **GI** | `−((W/L + Comm) × Rate)`, e.g. `−((1,000 + 100) × 60%) = −660` |

Both are rounded like above (`winlose.CalculateKind`); GI uses the same adjusted amount as the
version (v3 and before: `W/L − Comm`). A saved week keeps the type it was worked out with
(`weekly_group_results.calc_type`), so changing an agent's type applies to new entries, not to
saved ones. The Report shows the rate a GI Total is at with a minus (−60%). Agents assigned before
this are PO.

Earlier versions, kept only by finalized weeks saved with them: **v3** `(win_lose − commission) × (100% − rate)`
(GI: `−((win_lose − commission) × rate)`), **v2** `(win_lose − commission) × rate` and **v1**
`(win_lose + commission) × rate`, the first one, read from the spreadsheet. Weeks not finalized
follow the current version: starting the API after the change works them out again with it.

Groups from before agents had their own rates can still have partners. Under v1 and v2 each
partner's share is worked out at their own %, and the platform's Total is the sum of the shares
(each rounded on its own). Example: win/lose 1,000 and commission 200, Sok Dara at 40% and Chan Lina
at 10%: shares 480 and 120, Total 600 (shown at 50% in all). Under v3 each partner's share is still
their %, and the Total is what is left after all of them: (1,000 − 200) × (100% − 50%) = 400.

To change it: add a new version in `internal/winlose/calc.go` (keep the old one), then set the app
setting `winlose.formula` to the new version (`PUT /api/v1/settings`, needs `settings.manage`).
Weeks that aren't finalized follow the current version: the API works them out again with it when
it starts (`RecalculateOpenWeeks`, at the rates they were saved with) and when a week is reopened,
and a correction uses it. Finalized weeks keep the version they were settled with.
`GET /winlose/formula` reports a version as not `confirmed` until it is marked `Confirmed`.

On the **Report** (formerly the Weekly Report; `/winlose/reports/weekly`), tapping a group in **By group** shows only that group: each agent with the
rate its Total is at (an agent at 60%: **40%**) and its Total. Tapping an agent shows its part in
each of its groups the same way. Both details also show **By week**: each data entry in the range
(week-1, week-2, …) with its W/L, COMM and Total for the group or agent. Tapping one opens that
entry of the detail: a group's agents in it, or the agent's groups in it, with the entry's title
opening its page. Back returns to the dates before. Each entry has **✅ / ❌** reactions, shown like Telegram's: a pill for each one given (yours
filled blue) with the people's initials (a number above three), and a 😊+ button that opens a bar
to add one; hovering (or tapping) a pill shows who. A person has one per entry: the same again
takes it back, the other changes it (`POST /winlose/entries/:id/reactions` with
`{"reaction": "yes" | "no"}`, needs React to Data Entry; kept in `weekly_entry_reactions`, removed
with the entry; the audit log has `winlose.entry.react`). (`GET /winlose/reports` with
`by_entry=true` adds `entries`, each with its own `lines`, totals and reactions; `entry_id` limits
the report to one entry. The report's `weeks` still has each calendar week's totals.)

The Report and the Group summary list every switched-on group and agent, at 0 when it has no
results in the dates: the agents Data entry lists (the branch's own and the shared ones, not only
in switched-off groups), each in its own currency, and the switched-on groups (a group with no
results in its agents' currencies; one with results in one currency is at 0 in its other agents'
currencies, so a group with KHR results and USD agents is in the USD table too). A group's detail lists all its agents (each at its rate there),
an agent's all its groups. The totals stay what was entered. (`GET /winlose/reports` with
`everyone=true`; not with `partner_id`, `by_week` or `entry_id`.) **By group** is one table on
both: a row per group (A to Z ignoring case, agents in no group last) with its amounts in each of
its agents' currencies in the same row, USD first (`$1,000.00 / -៛500,000`; on phones one under the
other; USD and KHR are never added together). By agent, By week and a group's or an entry's
agents are a table per currency, USD first, each A to Z (ignoring case), and By agent and the
agents' tables end with the currency's Total. Export CSV lists the USD agents, then the KHR ones. On Group summary, the
**Image** of an entry or a group has the same tables, with the agent column titled **AG**.
The group form (Assign agents) shows beside each rate what the Total is worked out at (60% → 40%).

## Permissions

| Code | Allows |
|---|---|
Each RP page has its own view, add, update and delete (Data entry also react):

| Page | View | Add | Update | Delete | Other |
|---|---|---|---|---|---|
| Agents | `platforms.view` | `platforms.create` | `platforms.edit` (also turn off/on) | `platforms.delete` (never-used agents) | |
| Group Agent Assignments | `platform-assignments.view` | `platform-assignments.create` | `platform-assignments.edit` (also turn off/on) | `platform-assignments.delete` (groups with no weeks) | |
| Data Entry | `winlose.view` | `winlose.create` | `winlose.edit` (reason) | `winlose.delete` (reason) | `winlose.react` (✅ / ❌) |
| Report · Group Summary | `winlose-reports.view` | | | | `winlose-reports.consolidated` (all groups), `winlose-reports.rates` (show the rates; without it the API leaves them out of `GET /winlose/reports` and the Rate columns are hidden), `winlose-settlements.finalize` (finalize/reopen) |

Owner/admin roles (those with Manage Branches) get all of them when the API first starts with them.
They replaced `platforms.manage` and `platform-assignments.manage`: each role that had an old
permission gets the ones that took over from it, then the old ones are removed (see PERMISSIONS.md).
Default roles for new installs: manager (view agents and groups; view, add, update, delete and react
on Data entry; the Report), accountant (view agents and groups; view, add and react on Data entry;
the Report), viewer (view only). Give other roles access on Roles & permissions.

## API (`/api/v1`)

| Method & path | Permission |
|---|---|
| `GET /platforms?include_inactive&search&page&page_size` · `POST /platforms` · `PATCH /platforms/:id` (also turning off, with `reason`) · `DELETE /platforms/:id` (never-used agents) | view / create / edit / delete |
| `GET /partners` · `POST /partners` · `PATCH /partners/:id` `{name, phone?, note?, default_rate, is_active?}` | view (also groups, reports) / `platforms.create` / `platforms.edit` |
| `GET /platform-groups?branch_id?&search&platform_id&status=all\|active\|off&page&page_size` (no `branch_id` = your own branch; `branch_id=all` = every branch in your scope, each group with its `branch_code`; `search` = words in the title) · `GET /platform-groups/:id` · `POST /platform-groups` `{title, branch_id?, agents: [{platform_id, commission_rate, calc_type?: PO\|GI}], currency?, starts_on?}` · `PATCH /platform-groups/:id` `{title, agents: [{platform_id, commission_rate, calc_type?}], currency?}` · `POST /platform-groups/:id/active` `{is_active}` · `DELETE /platform-groups/:id` (only with no weeks entered) | `platform-assignments.view` (the list also with reports) / create / edit (also active) / delete |
| `GET /platform-assignments?branch_id?&platform_id&status=active\|ended\|all` (no `branch_id` = your own group) | `platform-assignments.*` |
| `POST /platform-assignments` `{platform_ids[], branch_ids[]?, partner_ids[]?, currency?, starts_on?}` — every platform to every branch (none = your own group) with the same partners (each at their default %), all or nothing; empty currency = each platform's default | `platform-assignments.create` |
| `PATCH /platform-assignments/:id` `{partners: [{partner_id, commission_rate}], commission_rate?, currency}` — replaces the partners; `commission_rate` is the assignment's own % (used with no partners; empty keeps it) | `platform-assignments.edit` |
| `POST /platform-assignments/:id/end` `{ends_on?, reason?}` · `/:id/move` `{branch_id, on?, reason?}` (the app no longer moves) | `platform-assignments.edit` |
| `GET /winlose/formula` | any win/lose permission |
| `DELETE /winlose/entries/:id` `{reason}` — the entry and its week's amounts, for good (the audit log keeps them); not in a finalized week | `winlose.delete` |
| `POST /winlose/entries/:id/reactions` `{reaction: yes\|no}` — your ✅ / ❌ on an entry (the same again takes it back) · `POST /winlose/results/:id/reactions` — the same on an agent in an entry (`result_id` of a saved row of `GET /winlose/entry`, which also has its `reactions`) | `winlose.react` |
| `GET /winlose/entries?branch_id?&search&created_from&created_to&page&page_size` — the entries of your groups, newest first (title, week, created at and by); the created days are the business's days | view / create / edit / delete |
| `GET /winlose/entry?week_start&branch_id?` (any day of the week; no `branch_id` = your own group) — includes the week's `entry` | view / create / edit |
| `POST /winlose/entry/preview` · `PUT /winlose/entry` `{week_start, branch_id?, title, rows[], reason?}` | create / edit |
| `GET /winlose/reports?date_from&date_to&branch_id?&group_id?&partner_id?&by_week?` — `group_id`: only that titled group's agents; `groups`: totals per group (no group last) — weeks starting in the range (from the Monday of `date_from`); lines per agent and currency across the groups covered, with each partner's share (`partners`), totals per agent, partner and currency; with `partner_id` only that partner's weeks, the Total being their share; `by_week=true` gives a line per week (partner statement) | reports (no `branch_id` = every group in scope with consolidated, else your own group) |
| `GET /winlose/settlements?week_start&branch_id?` · `POST /winlose/settlements` `{week_start, branch_id?}` · `POST /winlose/settlements/:id/reopen` (no `branch_id` = your own group) | reports / finalize |

**Paging:** the three lists above (agents, groups, entries) return every row unless `page` or `page_size` is sent; then they return one page, `{items, total, page, page_size}` (`page_size` 1–100, 20 when missing). The Agents, Group Agent Assignments and Data Entry pages show no pager: they load 30 rows, then ask for the next page by themselves as the list is scrolled near its end (`useInfiniteList` and `InfiniteLoader` in the frontend), until `total` rows are shown. A new search or filter starts again from the top; a save or delete fetches the pages already shown again. Agent pickers and the Report's group filter still load every row.

Rules the API enforces: a week that hasn't started can't be entered; a finalized week can't be
changed; a week can be finalized from its Sunday; ends and moves can't hide weeks already entered.

## Pages

- RP → **Data Entry** (`/winlose/entry`: a list of entries with Created at (the day and who
  made it), Title with its **✅ / ❌** (who reacted on hover) and the actions **View** (its own page: every agent with its saved W/L and COMM,
  each saved agent with its own ✅ / ❌, as on the form),
  **Update** and **Delete** (with a reason; needs Delete Data Entry), searchable by title and
  filtered by the days they were created (this month at first); **New entry** is a new entry for
  this week (Monday to Sunday): a title and every agent's W/L and COMM, each agent saved on its own;
  New entry and Update show the agents like the Report: a **USD** table first, then a **KHR**
  table, each **A to Z** (at first) or **Z to A** and its No column counting from 1 in that
  order; they find an agent by search, and show only the agents of one currency
  (**All**, **USD**, **KHR**) or whose code starts with one of the starts they have (up to the first
  `_`: **All** at first, then e.g. **12K_**, **SBK_**, **SBU_**), each button with how many; what
  was typed in the others stays. On a phone, W/L and COMM open the app's own big keypad instead of
  the phone's keyboard: 1–9, 0, a point (not for KHR), **−** (a loss, and back), delete and
  **Done**; an amount already in the box is replaced by the first key),
  **Report** (this month at first; by group, then by agent, and the own branch's weekly
  settlement for a whole week),
  **Group Summary** (`/winlose/reports/branch-summary`; this month at first; totals, by group, then by agent),
  **Agents** (`/admin/platforms`; **Add agent** (`/admin/platforms/new`) and **Update**
  (`/admin/platforms/:id`, which loads the agent with `GET /platforms/:id`) are pages), **Group Agent
  Assignments** (`/admin/platform-assignments`: your branch's titled groups of agents with their
  rates, searchable by title, each with an **Active** switch, **Update** and **Delete**; **Assign
  agents** (`/admin/platform-assignments/new`) and Update (`/admin/platform-assignments/:id`) are
  pages).
  The amounts are formatted money inputs (a minus for a loss); a new title or new amounts need
  Enter Win/Lose, changing a saved entry's title or amounts Correct Win/Lose (amounts also a
  reason). The Report and the Group Summary filter by group (Group Agent Assignments) and
  by date: Today, Yesterday, This week, Last week, This month, Last month or a custom range (weeks
  run Monday to Sunday); Data Entry's list has the same date choices plus All dates.
- **Partners** (and the Partner Statement) are no longer in the app: a group's agents each have a
  rate instead. The partner tables, their data and the `/partners` API stay, and weeks saved with
  partner shares keep them.
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
