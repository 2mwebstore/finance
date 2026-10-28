export interface User {
  id: string
  username: string
  email: string
  full_name: string
  is_active: boolean
  is_super_admin: boolean
  roles: string[]
  permissions: string[]
  /** Branches the user is assigned to. */
  branches: { id: string; code: string; name: string }[]
  /** Owner/admin access (Manage Branches): every branch; otherwise only scope_branches (their own). */
  all_branches?: boolean
  /** Active branches whose data the user may see and work in (use these for branch pickers). */
  scope_branches?: { id: string; code: string; name: string }[]
  last_login_at?: string
}

/** Which branches' data a role's users see. */
/** Role data scope: always "assigned" (own branches; data is not shared between branches). */
export type DataScope = 'assigned'

export interface LoginResponse {
  access_token: string
  expires_at: string
  user: {
    id: string
    username: string
    email: string
    full_name: string
    is_super_admin: boolean
    roles: string[]
  }
}

export interface TransactionAttachment {
  id: string
  name: string
  type: string
  size: number
  /** submitted: added by whoever entered the transaction; approval: added by its approver. */
  stage: 'submitted' | 'approval'
}

/** How many files each stage of a transaction can have (API: services.MaxAttachments). */
export const MAX_ATTACHMENTS = 2

/** All files in display order: the submitter's first, then the approver's. */
export function orderedAttachments(files: TransactionAttachment[] | undefined) {
  return [...filesOfStage(files, 'submitted'), ...filesOfStage(files, 'approval')]
}

/** Files of one stage (files from before stages existed count as submitted). */
export function filesOfStage(files: TransactionAttachment[] | undefined, stage: TransactionAttachment['stage']) {
  return (files || []).filter(file => (file.stage || 'submitted') === stage)
}

export interface TransactionOutcome {
  action: 'approved' | 'rejected' | 'voided'
  by: { id: string; name: string }
  at: string
  notes?: string
}

export interface Transaction {
  id: string
  transaction_number: string
  type: 'income' | 'expense'
  status: string
  branch_id: string
  category_id?: string | null
  payment_method_id: string
  currency: 'USD' | 'KHR'
  amount: string
  amount_display: string
  transaction_date: string
  description: string
  counterparty?: string
  /** Up to MAX_ATTACHMENTS images/PDFs (each served by GET /transactions/:id/attachments/:attachmentId) */
  attachments?: TransactionAttachment[]
  /** List rows only: whether the signed-in user may approve/reject this pending transaction */
  can_approve?: boolean
  can_reject?: boolean
  created_by?: { id: string; name: string }
  /** List rows only: who approved, rejected or deleted it, when, and their note or reason (null while pending). */
  outcome?: TransactionOutcome | null
  created_at?: string
  branch?: { id: string; code: string; name: string }
  category?: { id: string; code: string; name: string; type: string }
  payment_method?: { id: string; code: string; name: string }
}

/** Income, expense and net in one currency. */
export interface CurrencyFlow {
  income: string
  expense: string
  net: string
}

export interface DashboardSummary {
  period: string
  date_from: string
  date_to: string
  income_usd: string
  expense_usd: string
  net_usd: string
  income_khr: string
  expense_khr: string
  net_khr: string
  income_usd_display: string
  expense_usd_display: string
  net_usd_display: string
  income_khr_display: string
  expense_khr_display: string
  net_khr_display: string
  transaction_count: number
  daily?: DashboardDailyRow[]
  recent_transactions?: Transaction[]
  income_by_category?: { category_name: string; currency: string; total: string }[]
  expense_by_category?: { category_name: string; currency: string; total: string }[]
  by_branch?: { branch_id: string; branch_code: string; branch_name: string; income_usd: string; expense_usd: string; net_usd: string; income_khr: string; expense_khr: string; net_khr: string; by_currency?: Record<string, CurrencyFlow> }[]
  /** Income/expense/net per currency (every currency, not only USD and KHR). */
  by_currency?: Record<string, CurrencyFlow>
}

export interface DashboardDailyRow {
  date: string
  income_usd: string
  expense_usd: string
  net_usd: string
  income_khr: string
  expense_khr: string
  net_khr: string
  count: number
  by_currency?: Record<string, CurrencyFlow>
}

export interface Category {
  id: string
  code: string
  name: string
  type: string
  description?: string
  is_active: boolean
  is_system?: boolean
  /** null = shared by every branch */
  branch_id?: string | null
  branch_name?: string
}

export interface PaymentMethod {
  id: string
  code: string
  name: string
  is_active: boolean
}

export interface Branch {
  id: string
  code: string
  name: string
  address?: string
  phone?: string
  email?: string
  manager_id?: string | null
  is_active: boolean
  /** From the branch list: the branch sends its transaction updates to Telegram. */
  telegram_enabled?: boolean
}

/** A branch's Telegram chat for transaction updates (the bot token is never sent back). */
export interface BranchTelegramSettings {
  enabled: boolean
  chat_id: string
  events: string[]
  /** A bot token is saved for this branch. */
  own_bot: boolean
  /** The default bot (Backups page or server environment) exists. */
  default_bot_ready: boolean
  ready: boolean
}

export interface Role {
  id: string
  code: string
  name: string
  description?: string
  is_system?: boolean
  permissions?: { id: string; code: string; name: string; module?: string }[]
  /** Always "assigned": users see only their own branches (Manage Branches sees every branch). */
  data_scope?: DataScope
  /** A role for one branch (shown and given only there); empty = shared by every branch. */
  branch_id?: string | null
  branch?: { id: string; code: string; name: string }
  /** From the role list: whether the signed-in user may edit (and delete) this role. */
  can_manage?: boolean
}

export interface Permission {
  id: string
  code: string
  name: string
  module?: string
  description?: string
}

export interface AdminUser {
  /** false when a branch-level manager may view but not manage this user */
  can_manage?: boolean
  /** People who approve or reject this user's pending transactions */
  managers?: { id: string; username: string; full_name: string }[]
  id: string
  username: string
  email: string
  full_name: string
  is_active: boolean
  is_super_admin: boolean
  default_branch_id?: string | null
  roles: { id: string; code: string; name: string; permission_codes?: string[]; data_scope?: DataScope; branch_id?: string | null }[]
  branches: { id: string; code: string; name: string; is_active?: boolean }[]
  /** Whose data this user sees, from their roles: every branch, or these branch IDs. */
  data_scope?: { all: boolean; branch_ids: string[] }
  permission_codes?: string[]
  last_login_at?: string | null
  created_at?: string
}

export interface AuditEntry {
  id: string
  created_at: string
  actor_id?: string
  actor_email?: string
  action: string
  entity_type?: string
  entity_id?: string
  branch_id?: string
  ip_address?: string
  outcome: string
  details?: string
  correlation_id?: string
}

// ===== Inventory (separate from the income/expense ledger) =====

/** A currency code from Manage → Currency (USD, KHR, THB…). */
export type Currency = string

export interface ProductCategory {
  id: string
  code: string
  name: string
  description?: string
  is_active: boolean
}

/** A product (shared by every branch) with its stock: one branch, or the user's branches added up. */
export interface Product {
  id: string
  /** Generated when the product is added (SKU-000001). */
  sku: string
  /** A picture on the web; an uploaded picture is has_image (GET /inventory/products/:id/image). */
  image_url?: string
  has_image?: boolean
  updated_at?: string
  name: string
  category_id?: string | null
  category_name?: string
  unit: string
  barcode?: string
  currency: Currency
  sale_price: string
  cost_price: string
  reorder_level: string
  description?: string
  is_active: boolean
  quantity: string
  avg_cost: string
}

/** One product's stock in one branch. */
export interface StockRow {
  branch_id: string
  branch_code: string
  branch_name: string
  product_id: string
  sku: string
  name: string
  unit: string
  currency: Currency
  category_name: string
  quantity: string
  avg_cost: string
  value: string
  sale_price: string
  reorder_level: string
  low: boolean
  updated_at: string
}

export type StockMovementType = 'sale' | 'sale_void' | 'purchase' | 'purchase_void' | 'transfer_out' | 'transfer_in' | 'transfer_void' | 'adjustment'

export interface StockMovement {
  id: string
  branch_id: string
  branch_code: string
  product_id: string
  sku: string
  name: string
  unit: string
  currency: Currency
  type: StockMovementType
  quantity: string
  unit_cost: string
  balance_after: string
  ref_type: string
  ref_number: string
  note: string
  created_by: string
  created_at: string
}

export interface DocLine {
  product_id: string
  sku: string
  name: string
  unit: string
  quantity: string
  unit_price: string
  unit_cost: string
  discount: string
  line_total: string
  /** Purchases: unit cost into stock, with the purchase's shipping and discount spread in. */
  landed_cost?: string
}

export interface Purchase {
  id: string
  number: string
  branch_id: string
  branch_code: string
  branch_name: string
  supplier: string
  purchase_date: string
  currency: Currency
  subtotal: string
  discount: string
  shipping: string
  total: string
  /** KHR per USD when saved, and the total in USD (stock cost is kept in USD). */
  exchange_rate: string
  total_base: string
  payment_method: string
  status: 'pending' | 'approved' | 'rejected' | 'voided'
  note: string
  created_by_id: string
  created_by: string
  decided_by: string
  decided_at?: string | null
  decision_note: string
  voided_by: string
  voided_at?: string | null
  void_reason: string
  lines: DocLine[]
  created_at: string
  can_approve: boolean
  can_void: boolean
}

export interface Sale {
  id: string
  number: string
  branch_id: string
  branch_code: string
  branch_name: string
  customer: string
  sale_date: string
  currency: Currency
  payment_method: string
  subtotal: string
  discount: string
  shipping: string
  total: string
  /** KHR per USD when saved, and the total in USD; cost and profit are in USD. */
  exchange_rate: string
  total_base: string
  total_cost: string
  profit: string
  status: 'completed' | 'voided'
  note: string
  created_by: string
  voided_by: string
  voided_at?: string | null
  void_reason: string
  lines: DocLine[]
  created_at: string
  can_void: boolean
}

export interface StockTransfer {
  id: string
  number: string
  from_branch_id: string
  to_branch_id: string
  from_code: string
  from_name: string
  to_code: string
  to_name: string
  transfer_date: string
  status: 'completed' | 'voided'
  note: string
  created_by: string
  voided_by: string
  voided_at?: string | null
  void_reason: string
  lines: DocLine[]
  created_at: string
  can_void: boolean
}

/** Inventory figures for a period, in USD (each document converted at its own rate). */
export interface InventorySummary {
  currency: 'USD'
  stock_value: string
  sales_total: string
  sales_cost: string
  profit: string
  sales_count: number
  purchases_total: string
  purchases_count: number
  pending_purchases: number
  expenses_total: string
  expenses_count: number
  /** Profit − inventory expenses. */
  net_profit: string
  products: number
  low_stock: number
}

/** One currency from Manage → Currency (rate: units for 1 USD; USD is the base at 1). */
export interface CurrencyInfo {
  code: string
  name: string
  symbol: string
  decimals: number
  rate: string
  active: boolean
  base: boolean
}

/** Manage → Currency: USD is the base; every other currency converts at its rate. */
export interface CurrencySettings {
  base: 'USD'
  currencies: CurrencyInfo[]
  default_currency: string
  updated_at?: string
  updated_by?: string
}

export interface Paged<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

/** A line being edited on a sale, purchase or transfer form (price: unit price or unit cost). */
export interface DocLineForm {
  key: number
  product_id: string
  quantity: string
  price: string
  /** Typed as an amount or a percent of the line; sent as an amount. */
  discount: string
  discount_mode: DiscountMode
}

export type DiscountMode = 'amount' | 'percent'

/** A cost of the inventory side (shipping, packaging…), in an inventory expense category. */
export interface InventoryExpense {
  id: string
  number: string
  branch_id: string
  branch_code: string
  branch_name: string
  category_id: string
  category_name: string
  expense_date: string
  currency: Currency
  amount: string
  exchange_rate: string
  amount_base: string
  payment_method: string
  payee: string
  note: string
  status: 'completed' | 'voided'
  created_by: string
  voided_by: string
  voided_at?: string | null
  void_reason: string
  created_at: string
  can_void: boolean
}
