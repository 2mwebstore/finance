import { P, ROUTE_PERMISSIONS, type PermissionCode } from '~/utils/permissions'

/**
 * Role-based UI permissions backed by the Pinia permissions store.
 * User grants come from /auth/me; catalog auto-syncs from GET /permissions.
 * Server remains the source of truth for authorization.
 */
export function usePermissions() {
  const store = usePermissionsStore()
  const { user, hasPermission } = useAuth()

  const isSuperAdmin = computed(
    () => store.isSuperAdmin || !!user.value?.is_super_admin,
  )

  const permissions = computed(() => {
    if (store.userCodes.length) return store.userCodes
    return user.value?.permissions || []
  })

  // Owner/admin/Super Admin run the whole system (branches.manage); everyone else is branch-level.
  const hasFullAccess = computed(() => isSuperAdmin.value || can(P.BranchesManage))

  const catalog = computed(() => store.catalog)
  const catalogSynced = computed(() => store.catalogSynced)
  const catalogSyncError = computed(() => store.catalogSyncError)

  function can(...codes: PermissionCode[] | string[]) {
    if (!codes.length) return true
    if (isSuperAdmin.value) return true
    // Prefer store; fall back to auth composable
    if (store.userCodes.length) {
      return codes.some((c) => store.userCodes.includes(c))
    }
    return hasPermission(...codes)
  }

  function canAll(...codes: PermissionCode[] | string[]) {
    if (isSuperAdmin.value) return true
    if (store.userCodes.length) {
      return codes.every((c) => store.userCodes.includes(c))
    }
    if (!user.value) return false
    const set = new Set(user.value.permissions || [])
    return codes.every((c) => set.has(c))
  }

  function canAccessRoute(path: string) {
    const entry = Object.entries(ROUTE_PERMISSIONS).find(([route]) => {
      if (path === route) return true
      if (route !== '/' && path.startsWith(route + '/')) return true
      return false
    })
    if (!entry) return true
    return can(...entry[1])
  }

  async function refreshCatalog() {
    const { api } = useApi()
    await store.syncCatalogFromApi(api)
  }

  const canViewDashboard = computed(() => can(P.DashboardView))
  const canViewTransactions = computed(() => can(P.TransactionsView, P.IncomeView, P.ExpenseView))
  const canCreateIncome = computed(() => can(P.IncomeCreate))
  const canCreateExpense = computed(() => can(P.ExpenseCreate))
  const canCreateTransaction = computed(() => can(P.IncomeCreate, P.ExpenseCreate))
  const canEditIncome = computed(() => can(P.IncomeEdit))
  const canEditExpense = computed(() => can(P.ExpenseEdit))
  const canVoidIncome = computed(() => can(P.IncomeVoid))
  const canVoidExpense = computed(() => can(P.ExpenseVoid))
  const canApprove = computed(() => can(P.TransactionsApprove))
  const canReject = computed(() => can(P.TransactionsReject))
  const canReverse = computed(() => can(P.TransactionsReverse))
  const canViewReports = computed(() => can(P.ReportsView, P.DashboardView))
  const canExportReports = computed(() => can(P.ReportsExport))
  const canManageBranches = computed(() => can(P.BranchesCreate, P.BranchesEdit, P.BranchesManage))
  const canCreateBranch = computed(() => can(P.BranchesCreate, P.BranchesManage))
  const canViewBranches = computed(() => can(P.BranchesView, P.BranchesManage))
  const canViewUsers = computed(() => can(P.UsersView, P.UsersManage))
  const canCreateUser = computed(() => can(P.UsersCreate, P.UsersManage))
  const canViewRoles = computed(() => can(P.RolesView, P.RolesManage))
  // The Permission Codes page has its own permission, so it can be shown or hidden apart from roles.
  const canViewPermissionCodes = computed(() => can(P.PermissionsView))
  const canViewAudit = computed(() => can(P.AuditLogsView))
  const canManageSettings = computed(() => can(P.SettingsManage))
  // The API also requires full access (owner / Super Admin) for backups.
  const canManageBackups = computed(() => hasFullAccess.value && can(P.BackupsManage))

  return {
    P,
    catalog,
    catalogSynced,
    catalogSyncError,
    refreshCatalog,
    isSuperAdmin,
    hasFullAccess,
    permissions,
    can,
    canAll,
    canAccessRoute,
    canViewDashboard,
    canViewTransactions,
    canCreateIncome,
    canCreateExpense,
    canCreateTransaction,
    canEditIncome,
    canEditExpense,
    canVoidIncome,
    canVoidExpense,
    canApprove,
    canReject,
    canReverse,
    canViewReports,
    canExportReports,
    canManageBranches,
    canCreateBranch,
    canViewBranches,
    canViewUsers,
    canCreateUser,
    canViewRoles,
    canViewPermissionCodes,
    canViewAudit,
    canManageSettings,
    canManageBackups,
  }
}
