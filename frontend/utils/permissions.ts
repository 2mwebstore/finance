/** Permission codes — must match backend internal/permissions */
export const P = {
  DashboardView: 'dashboard.view',

  IncomeView: 'income.view',
  IncomeCreate: 'income.create',
  IncomeEdit: 'income.edit',
  IncomeVoid: 'income.void',

  ExpenseView: 'expense.view',
  ExpenseCreate: 'expense.create',
  ExpenseEdit: 'expense.edit',
  ExpenseVoid: 'expense.void',

  TransactionsView: 'transactions.view',
  TransactionsApprove: 'transactions.approve',
  TransactionsReject: 'transactions.reject',
  TransactionsReverse: 'transactions.reverse',

  ReportsView: 'reports.view',
  ReportsExport: 'reports.export',

  BranchesView: 'branches.view',
  BranchesCreate: 'branches.create',
  BranchesEdit: 'branches.edit',
  BranchesManage: 'branches.manage',

  UsersView: 'users.view',
  UsersCreate: 'users.create',
  UsersEdit: 'users.edit',
  UsersManage: 'users.manage',

  RolesView: 'roles.view',
  RolesManage: 'roles.manage',
  PermissionsManage: 'permissions.manage',
  PermissionsView: 'permissions.view',

  CategoriesManage: 'categories.manage',
  CurrencySettingsManage: 'currency-settings.manage',
  AuditLogsView: 'audit-logs.view',
  SettingsManage: 'settings.manage',
  BackupsManage: 'backups.manage',

  InventoryView: 'inventory.view',
  InventoryManage: 'inventory.manage',
  InventoryAdjust: 'inventory.adjust',
  InventoryTransfer: 'inventory.transfer',
  SalesView: 'sales.view',
  SalesCreate: 'sales.create',
  SalesVoid: 'sales.void',
  PurchasesView: 'purchases.view',
  PurchasesCreate: 'purchases.create',
  PurchasesApprove: 'purchases.approve',
  PurchasesVoid: 'purchases.void',

  InventoryExpensesView: 'inventory-expenses.view',
  InventoryExpensesCreate: 'inventory-expenses.create',
  InventoryExpensesVoid: 'inventory-expenses.void',
} as const

export type PermissionCode = (typeof P)[keyof typeof P]

/** Full catalog for docs / admin UI */
export const PERMISSION_CATALOG: {
  code: PermissionCode
  name: string
  module: string
  description: string
}[] = [
  { code: P.DashboardView, name: 'View Dashboard', module: 'Dashboard', description: 'Access the main dashboard' },

  { code: P.IncomeView, name: 'View Income', module: 'Income', description: 'View income transactions' },
  { code: P.IncomeCreate, name: 'Create Income', module: 'Income', description: 'Create income transactions' },
  { code: P.IncomeEdit, name: 'Edit Income', module: 'Income', description: 'Edit income transactions' },
  { code: P.IncomeVoid, name: 'Void Income', module: 'Income', description: 'Void income transactions' },

  { code: P.ExpenseView, name: 'View Expenses', module: 'Expenses', description: 'View expense transactions' },
  { code: P.ExpenseCreate, name: 'Create Expenses', module: 'Expenses', description: 'Create expense transactions' },
  { code: P.ExpenseEdit, name: 'Edit Expenses', module: 'Expenses', description: 'Edit expense transactions' },
  { code: P.ExpenseVoid, name: 'Void Expenses', module: 'Expenses', description: 'Void expense transactions' },

  { code: P.TransactionsView, name: 'View Transactions', module: 'Transactions', description: 'View all transactions' },
  { code: P.TransactionsApprove, name: 'Approve Transactions', module: 'Transactions', description: 'Approve pending transactions' },
  { code: P.TransactionsReject, name: 'Reject Transactions', module: 'Transactions', description: 'Reject pending transactions' },
  { code: P.TransactionsReverse, name: 'Reverse Transactions', module: 'Transactions', description: 'Reverse finalized transactions' },

  { code: P.ReportsView, name: 'View Reports', module: 'Reports', description: 'View financial reports' },
  { code: P.ReportsExport, name: 'Export Reports', module: 'Reports', description: 'Export reports to CSV/Excel/PDF' },

  { code: P.BranchesView, name: 'View Branches', module: 'Branches', description: 'View branch list and details' },
  { code: P.BranchesCreate, name: 'Create Branches', module: 'Branches', description: 'Create new branches' },
  { code: P.BranchesEdit, name: 'Edit Branches', module: 'Branches', description: 'Edit branch details' },
  { code: P.BranchesManage, name: 'Manage Branches', module: 'Branches', description: 'Full branch management including assignments' },

  { code: P.UsersView, name: 'View Users', module: 'Users', description: 'View user list and profiles' },
  { code: P.UsersCreate, name: 'Create Users', module: 'Users', description: 'Create new users' },
  { code: P.UsersEdit, name: 'Edit Users', module: 'Users', description: 'Edit user profiles' },
  { code: P.UsersManage, name: 'Manage Users', module: 'Users', description: 'Manage user roles, passwords, and account activation' },

  { code: P.RolesView, name: 'View Roles', module: 'Roles', description: 'View roles and their permissions' },
  { code: P.RolesManage, name: 'Manage Roles', module: 'Roles', description: 'Create, edit, and delete non-system roles' },
  { code: P.PermissionsManage, name: 'Manage Permissions', module: 'Roles', description: 'Assign permissions to roles' },
  { code: P.PermissionsView, name: 'View Permission Codes', module: 'Roles', description: 'See the Permission Codes page with every permission in the system' },

  { code: P.CategoriesManage, name: 'Manage Categories', module: 'Settings', description: 'Create, edit, and deactivate income/expense categories' },
  { code: P.CurrencySettingsManage, name: 'Manage Currency Settings', module: 'Settings', description: 'Configure currency options' },
  { code: P.AuditLogsView, name: 'View Audit Logs', module: 'Audit', description: 'View system audit logs' },
  { code: P.SettingsManage, name: 'Manage Settings', module: 'Settings', description: 'Change application settings' },
  { code: P.BackupsManage, name: 'Manage Backups', module: 'Backups', description: 'Back up the database, send backups to Telegram, and restore from a backup' },

  { code: P.InventoryView, name: 'View Inventory', module: 'Inventory', description: 'See products, stock levels and stock history of your branches' },
  { code: P.InventoryManage, name: 'Manage Products', module: 'Inventory', description: 'Create and edit products and product categories (shared by every branch)' },
  { code: P.InventoryAdjust, name: 'Adjust Stock', module: 'Inventory', description: 'Add, remove or count stock in your branches, with a reason' },
  { code: P.InventoryTransfer, name: 'Transfer Stock', module: 'Inventory', description: 'Move stock between branches' },
  { code: P.SalesView, name: 'View Sales', module: 'Sales', description: 'See sales of your branches' },
  { code: P.SalesCreate, name: 'Create Sales', module: 'Sales', description: 'Record sales; their stock goes out right away' },
  { code: P.SalesVoid, name: 'Void Sales', module: 'Sales', description: 'Void a sale and put its stock back' },
  { code: P.PurchasesView, name: 'View Purchases', module: 'Purchases', description: 'See purchases of your branches' },
  { code: P.PurchasesCreate, name: 'Create Purchases', module: 'Purchases', description: 'Record purchases from suppliers; stock goes in once approved' },
  { code: P.PurchasesApprove, name: 'Approve Purchases', module: 'Purchases', description: 'Approve or reject purchases; approved purchases add stock' },
  { code: P.PurchasesVoid, name: 'Void Purchases', module: 'Purchases', description: 'Void a purchase and take its stock back out' },

  { code: P.InventoryExpensesView, name: 'View Inventory Expenses', module: 'Inventory Expenses', description: 'See inventory expenses of your branches' },
  { code: P.InventoryExpensesCreate, name: 'Create Inventory Expenses', module: 'Inventory Expenses', description: 'Record inventory expenses (shipping, packaging, storage…)' },
  { code: P.InventoryExpensesVoid, name: 'Void Inventory Expenses', module: 'Inventory Expenses', description: 'Void an inventory expense' },
]

/** Permission modules in the same order as the sidebar (Settings holds Categories). */
export const PERMISSION_MODULE_ORDER = ['Dashboard', 'Transactions', 'Income', 'Expenses', 'Reports', 'Inventory', 'Sales', 'Purchases', 'Inventory Expenses', 'Branches', 'Roles', 'Users', 'Settings', 'Audit', 'Backups']

/** Sort key for a module: sidebar order, unknown modules last. */
export function permissionModuleRank(module: string) {
  const index = PERMISSION_MODULE_ORDER.indexOf(module)
  return index === -1 ? PERMISSION_MODULE_ORDER.length : index
}

/** Route → any-of permissions required to enter */
export const ROUTE_PERMISSIONS: Record<string, PermissionCode[]> = {
  '/dashboard': [P.DashboardView],
  '/transactions': [P.TransactionsView, P.IncomeView, P.ExpenseView],
  '/transactions/new': [P.IncomeCreate, P.ExpenseCreate],
  '/reports': [P.ReportsView, P.DashboardView],
  '/admin/branches': [P.BranchesView, P.BranchesManage],
  '/admin/users': [P.UsersView, P.UsersManage],
  '/admin/roles': [P.RolesView, P.RolesManage],
  '/admin/audit': [P.AuditLogsView],
  '/admin/backups': [P.BackupsManage],
  '/admin/permissions': [P.PermissionsView],
  '/admin/categories': [P.CategoriesManage],
  '/inventory/products': [P.InventoryView],
  '/inventory/stock': [P.InventoryView],
  '/inventory/sales': [P.SalesView],
  '/inventory/sales/new': [P.SalesCreate],
  '/inventory/purchases': [P.PurchasesView],
  '/inventory/purchases/new': [P.PurchasesCreate],
  '/inventory/transfers': [P.InventoryView],
  '/inventory/expenses': [P.InventoryExpensesView],
  '/inventory/expense-categories': [P.InventoryManage],
  '/inventory/payment-methods': [P.InventoryManage, P.CategoriesManage],
  '/admin/currency': [P.CurrencySettingsManage],
  // The API also needs inventory.view; the menu shows it only with both.
  '/inventory/reports': [P.ReportsView],
}
