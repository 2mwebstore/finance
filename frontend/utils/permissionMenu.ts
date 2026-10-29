import type { Component } from 'vue'
import {
  ArrowLeftRight,
  Boxes,
  Building2,
  CalendarRange,
  ChartColumn,
  ChartPie,
  Coins,
  DatabaseBackup,
  FileChartColumnIncreasing,
  Handshake,
  KeyRound,
  Layers,
  LayoutDashboard,
  Network,
  NotebookPen,
  Package,
  ReceiptText,
  ScrollText,
  Settings,
  ShieldCheck,
  ShoppingCart,
  Store,
  Tags,
  TrendingDown,
  TrendingUp,
  Truck,
  UsersRound,
  Wallet,
} from 'lucide-vue-next'
import { P } from '~/utils/permissions'

export interface PermissionMenuPage {
  key: string
  label: string
  icon: Component
  /** The permissions that open the page or act on it (a permission can open several pages). */
  codes: string[]
  /** Anything else worth knowing about getting in. */
  note?: string
}

export interface PermissionMenuSection {
  /** The sidebar's caption ('' for the top items, which have none). */
  caption: string
  pages: PermissionMenuPage[]
}

/**
 * Permissions laid out exactly like the sidebar (layouts/default.vue): the same sections, pages,
 * labels, icons and order. Keep the two in step when the sidebar changes. A permission that opens
 * several pages (View Inventory: Products, Stock, Transfers, Inventory Reports) is listed on each;
 * ticking it on one ticks it on all. Settings has no sidebar page and comes last under Manage.
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
      { key: 'products', label: 'Products', icon: Package, codes: [P.InventoryView, P.InventoryManage] },
      { key: 'stock', label: 'Stock', icon: Boxes, codes: [P.InventoryView, P.InventoryAdjust] },
      { key: 'pos', label: 'POS', icon: Store, codes: [P.SalesCreate] },
      { key: 'sales', label: 'Sales', icon: ShoppingCart, codes: [P.SalesView, P.SalesCreate, P.SalesEdit, P.SalesVoid] },
      { key: 'purchases', label: 'Purchases', icon: Truck, codes: [P.PurchasesView, P.PurchasesCreate, P.PurchasesEdit, P.PurchasesApprove, P.PurchasesVoid] },
      { key: 'inventory-expenses', label: 'Expenses', icon: ReceiptText, codes: [P.InventoryExpensesView, P.InventoryExpensesCreate, P.InventoryExpensesVoid] },
      { key: 'expense-categories', label: 'Expense Categories', icon: Tags, codes: [P.InventoryManage] },
      { key: 'payment-methods', label: 'Payment Methods', icon: Wallet, codes: [P.InventoryManage, P.CategoriesManage], note: 'Either one opens it.' },
      { key: 'transfers', label: 'Transfers', icon: ArrowLeftRight, codes: [P.InventoryView, P.InventoryTransfer] },
      { key: 'inventory-reports', label: 'Inventory Reports', icon: ChartPie, codes: [P.ReportsView, P.InventoryView], note: 'Needs both.' },
    ],
  },
  {
    caption: 'PLATFORMS',
    pages: [
      { key: 'winlose-entry', label: 'Data Entry', icon: NotebookPen, codes: [P.WinLoseView, P.WinLoseCreate, P.WinLoseEdit] },
      { key: 'winlose-weekly', label: 'Weekly Report', icon: CalendarRange, codes: [P.WinLoseReportsView, P.WinLoseReportsConsolidated, P.WinLoseSettlementsFinalize] },
      { key: 'winlose-branch-summary', label: 'Group Summary', icon: ChartColumn, codes: [P.WinLoseReportsView, P.WinLoseReportsConsolidated] },
      { key: 'winlose-partner-statement', label: 'Partner Statement', icon: ScrollText, codes: [P.WinLoseReportsView, P.WinLoseReportsConsolidated] },
      { key: 'platforms', label: 'Agents', icon: Layers, codes: [P.PlatformsView, P.PlatformsManage] },
      { key: 'partners', label: 'Partners', icon: Handshake, codes: [P.PlatformsView, P.PlatformsManage] },
      { key: 'platform-assignments', label: 'Group Agent Assignments', icon: Network, codes: [P.PlatformsView, P.PlatformAssignmentsManage] },
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
      { key: 'backups', label: 'Backups', icon: DatabaseBackup, codes: [P.BackupsManage], note: 'Also needs owner/admin access.' },
      { key: 'settings', label: 'Settings', icon: Settings, codes: [P.SettingsManage], note: 'No sidebar page: app settings, and clearing old audit entries with Audit Log.' },
    ],
  },
]

/** The sidebar pages a permission opens or acts on, by label (for "also on…" hints). */
export function pagesOf(code: string): string[] {
  return PERMISSION_MENU.flatMap(section => section.pages.filter(page => page.codes.includes(code)).map(page => page.label))
}

const OTHER: PermissionMenuPage = { key: 'other', label: 'Other', icon: KeyRound, codes: [] }

export interface MenuGroup<T> {
  caption: string
  pages: (PermissionMenuPage & { items: T[] })[]
}

/**
 * Sorts permissions into the sidebar's sections and pages (in the menu's order, and the menu's
 * order within a page). A permission shared by several pages is on each of them. Codes the menu
 * doesn't know yet go to "Other" at the end. Sections and pages with none of the permissions are
 * dropped.
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
