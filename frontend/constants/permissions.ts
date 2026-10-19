/**
 * Frontend permission codes — keep in sync with:
 * - backend/internal/permissions/permissions.go
 * - permissions.catalog.json (project root)
 * - docs/PERMISSIONS.md
 *
 * Re-exports the canonical definitions from utils/permissions.ts
 */
export {
  P,
  PERMISSION_CATALOG,
  ROUTE_PERMISSIONS,
  type PermissionCode,
} from '~/utils/permissions'

/** Flat list of all 30 codes (for checks / display). */
export const PERMISSION_CODES = [
  'dashboard.view',
  'income.view',
  'income.create',
  'income.edit',
  'income.void',
  'expense.view',
  'expense.create',
  'expense.edit',
  'expense.void',
  'transactions.view',
  'transactions.approve',
  'transactions.reject',
  'transactions.reverse',
  'reports.view',
  'reports.export',
  'branches.view',
  'branches.create',
  'branches.edit',
  'branches.manage',
  'users.view',
  'users.create',
  'users.edit',
  'users.manage',
  'roles.view',
  'roles.manage',
  'permissions.manage',
  'categories.manage',
  'currency-settings.manage',
  'audit-logs.view',
  'settings.manage',
] as const

export type PermissionCodeLiteral = (typeof PERMISSION_CODES)[number]
