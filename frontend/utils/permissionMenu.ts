import type { Component } from 'vue'
import {
  ArrowLeftRight,
  CalendarCheck,
  FileSpreadsheet,
  Layers,
  Network,
  NotebookPen,
  Boxes,
  Building2,
  Coins,
  DatabaseBackup,
  FileChartColumnIncreasing,
  KeyRound,
  LayoutDashboard,
  Package,
  ReceiptText,
  Settings,
  ShieldCheck,
  ShoppingCart,
  Tags,
  TrendingDown,
  TrendingUp,
  Truck,
  UsersRound,
} from 'lucide-vue-next'
import { P } from '~/utils/permissions'

export interface PermissionMenuPage {
  key: string
  label: string
  icon: Component
  codes: string[]
}

export interface PermissionMenuSection {
  /** The sidebar's caption ('' for the top items, which have none). */
  caption: string
  pages: PermissionMenuPage[]
}

/**
 * Permissions laid out like the sidebar (layouts/default.vue): its sections and pages, with the
 * same labels, icons and order. Each code sits under the page it opens or acts on. Pages that need
 * no permission of their own (Inventory Reports: reports + inventory) are left out.
 */
export const PERMISSION_MENU: PermissionMenuSection[] = [
  { caption: '', pages: [{ key: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, codes: [P.DashboardView] }] },
  {
    caption: 'FINANCIAL ACTIVITY',
    pages: [
      { key: 'transactions', label: 'Transactions', icon: ReceiptText, codes: [P.TransactionsView, P.TransactionsApprove, P.TransactionsReject, P.TransactionsReverse] },
      { key: 'income', label: 'Income', icon: TrendingUp, codes: [P.IncomeView, P.IncomeCreate, P.IncomeEdit, P.IncomeVoid] },
      { key: 'expense', label: 'Expense', icon: TrendingDown, codes: [P.ExpenseView, P.ExpenseCreate, P.ExpenseEdit, P.ExpenseVoid] },
      { key: 'reports', label: 'Reports', icon: FileChartColumnIncreasing, codes: [P.ReportsView, P.ReportsExport] },
      { key: 'categories', label: 'Categories', icon: Tags, codes: [P.CategoriesManage] },
    ],
  },
  {
    caption: 'INVENTORY',
    pages: [
      // Manage Products also opens Expense Categories and Payment Methods.
      { key: 'products', label: 'Products', icon: Package, codes: [P.InventoryView, P.InventoryManage] },
      { key: 'stock', label: 'Stock', icon: Boxes, codes: [P.InventoryAdjust] },
      // Create Sales opens the POS as well as the sale form.
      { key: 'sales', label: 'POS & Sales', icon: ShoppingCart, codes: [P.SalesView, P.SalesCreate, P.SalesEdit, P.SalesVoid] },
      { key: 'purchases', label: 'Purchases', icon: Truck, codes: [P.PurchasesView, P.PurchasesCreate, P.PurchasesEdit, P.PurchasesApprove, P.PurchasesVoid] },
      { key: 'inventory-expenses', label: 'Expenses', icon: ReceiptText, codes: [P.InventoryExpensesView, P.InventoryExpensesCreate, P.InventoryExpensesVoid] },
      { key: 'transfers', label: 'Transfers', icon: ArrowLeftRight, codes: [P.InventoryTransfer] },
    ],
  },
  {
    caption: 'PLATFORMS',
    pages: [
      { key: 'winlose-entry', label: 'Win/Lose Entry', icon: NotebookPen, codes: [P.WinLoseView, P.WinLoseCreate, P.WinLoseEdit] },
      // One permission opens the Weekly Report and the HQ Summary.
      { key: 'winlose-reports', label: 'Win/Lose Reports', icon: FileSpreadsheet, codes: [P.WinLoseReportsView, P.WinLoseReportsConsolidated] },
      { key: 'winlose-settlements', label: 'Weekly Settlements', icon: CalendarCheck, codes: [P.WinLoseSettlementsFinalize] },
      { key: 'platforms', label: 'Platforms', icon: Layers, codes: [P.PlatformsView, P.PlatformsManage] },
      { key: 'platform-assignments', label: 'HQ Platform Assignments', icon: Network, codes: [P.PlatformAssignmentsManage] },
    ],
  },
  {
    caption: 'MANAGE',
    pages: [
      { key: 'branches', label: 'Branches', icon: Building2, codes: [P.BranchesView, P.BranchesCreate, P.BranchesEdit, P.BranchesManage] },
      { key: 'roles', label: 'Roles & Permissions', icon: ShieldCheck, codes: [P.RolesView, P.RolesManage, P.PermissionsManage] },
      { key: 'permission-codes', label: 'Permission Codes', icon: KeyRound, codes: [P.PermissionsView] },
      { key: 'users', label: 'Users', icon: UsersRound, codes: [P.UsersView, P.UsersCreate, P.UsersEdit, P.UsersManage] },
      { key: 'currency', label: 'Currency', icon: Coins, codes: [P.CurrencySettingsManage] },
      { key: 'audit', label: 'Audit Log', icon: ReceiptText, codes: [P.AuditLogsView] },
      { key: 'backups', label: 'Backups', icon: DatabaseBackup, codes: [P.BackupsManage] },
      // Not a sidebar page: app settings (and, with Audit Log, clearing old audit entries).
      { key: 'settings', label: 'Settings', icon: Settings, codes: [P.SettingsManage] },
    ],
  },
]

const OTHER: PermissionMenuPage = { key: 'other', label: 'Other', icon: KeyRound, codes: [] }

export interface MenuGroup<T> {
  caption: string
  pages: (PermissionMenuPage & { items: T[] })[]
}

/**
 * Sorts permissions into the sidebar's sections and pages (in the menu's order, and the menu's
 * order within a page). Codes the menu doesn't know yet go to "Other" at the end. Sections and
 * pages with none of the permissions are dropped.
 */
export function groupByMenu<T extends { code: string }>(permissions: T[]): MenuGroup<T>[] {
  const byCode = new Map(permissions.map(p => [p.code, p]))
  const placed = new Set<string>()
  const groups: MenuGroup<T>[] = PERMISSION_MENU.map(section => ({
    caption: section.caption,
    pages: section.pages.map((page) => {
      const items = page.codes.flatMap(code => byCode.has(code) ? [byCode.get(code)!] : [])
      items.forEach(item => placed.add(item.code))
      return { ...page, items }
    }).filter(page => page.items.length),
  }))
  const rest = permissions.filter(p => !placed.has(p.code))
  if (rest.length) groups.push({ caption: 'OTHER', pages: [{ ...OTHER, codes: rest.map(p => p.code), items: rest }] })
  return groups.filter(section => section.pages.length)
}
