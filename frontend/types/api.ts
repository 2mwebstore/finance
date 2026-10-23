export interface User {
  id: string
  username: string
  email: string
  full_name: string
  is_active: boolean
  is_super_admin: boolean
  roles: string[]
  permissions: string[]
  branches: { id: string; code: string; name: string }[]
  last_login_at?: string
}

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
  by_branch?: { branch_id: string; branch_code: string; branch_name: string; income_usd: string; expense_usd: string; net_usd: string; income_khr: string; expense_khr: string; net_khr: string }[]
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
}

export interface Role {
  id: string
  code: string
  name: string
  description?: string
  is_system?: boolean
  permissions?: { id: string; code: string; name: string; module?: string }[]
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
  roles: { id: string; code: string; name: string; permission_codes?: string[] }[]
  branches: { id: string; code: string; name: string; is_active?: boolean }[]
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
