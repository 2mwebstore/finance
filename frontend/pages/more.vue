<template>
  <section class="more-page">
    <header class="page-header">
      <p class="eyebrow">ACCOUNT &amp; ACCESS</p>
      <h1>More</h1>
      <p>Quick access to your workspace and account settings.</p>
    </header>

    <article class="account-card">
      <div class="account-avatar">{{ initials }}</div>
      <div class="account-copy"><strong>{{ user?.full_name || 'Account' }}</strong><span>{{ user?.email }}</span><small>{{ (user?.roles || []).join(', ') || 'No role assigned' }}</small></div>
      <span v-if="isSuperAdmin" class="admin-badge">Super Admin</span>
    </article>

    <section class="link-card">
      <div class="section-label">FINANCIAL ACTIVITY</div>
      <Can perm="dashboard.view"><NuxtLink to="/dashboard"><span>Dashboard</span><small>Business overview</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['transactions.view','income.view','expense.view']"><NuxtLink to="/transactions"><span>Transactions</span><small>Review financial activity</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['income.create','expense.create']"><NuxtLink to="/transactions/new"><span>Add transaction</span><small>Record income or expense</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['reports.view','dashboard.view']"><NuxtLink to="/reports"><span>Reports</span><small>Explore financial insights</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="categories.manage"><NuxtLink to="/admin/categories"><span>Categories</span><small>Categories for expense transactions</small><ChevronRight :size="17" /></NuxtLink></Can>
    </section>

    <section v-if="canViewInventory || canViewSales || canViewPurchases || canViewTransfers || canViewInventoryExpenses" class="link-card">
      <div class="section-label">INVENTORY</div>
      <Can perm="inventory.view"><NuxtLink to="/inventory/products"><span>Products</span><small>The products every branch sells</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="inventory.view"><NuxtLink to="/inventory/stock"><span>Stock</span><small>What each branch has now</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="sales.view"><NuxtLink to="/inventory/sales"><span>Sales</span><small>Goods sold</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="purchases.view"><NuxtLink to="/inventory/purchases"><span>Purchases</span><small>Goods bought from suppliers</small><ChevronRight :size="17" /></NuxtLink></Can>
      <NuxtLink v-if="canViewInventoryExpenses" to="/inventory/expenses"><span>Inventory expenses</span><small>Shipping, packaging, storage…</small><ChevronRight :size="17" /></NuxtLink>
      <Can perm="inventory.manage"><NuxtLink to="/inventory/expense-categories"><span>Expense categories</span><small>Categories for inventory expenses</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['inventory.manage', 'categories.manage']"><NuxtLink to="/inventory/payment-methods"><span>Payment methods</span><small>Cash, bank, KHQR…</small><ChevronRight :size="17" /></NuxtLink></Can>
      <NuxtLink v-if="canViewTransfers" to="/inventory/transfers"><span>Transfers</span><small>Stock moved between branches</small><ChevronRight :size="17" /></NuxtLink>
      <NuxtLink v-if="canViewInventoryReports" to="/inventory/reports"><span>Inventory reports</span><small>Stock value, sales and profit</small><ChevronRight :size="17" /></NuxtLink>
    </section>

    <section v-if="can('winlose.view', 'winlose.create', 'winlose.edit', 'winlose-reports.view', 'platforms.view', 'platforms.manage', 'platform-assignments.manage')" class="link-card">
      <div class="section-label">PLATFORMS</div>
      <Can :any="['winlose.view', 'winlose.create', 'winlose.edit']"><NuxtLink to="/winlose/entry"><span>Win/lose entry</span><small>Each platform’s week for an HQ</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="winlose-reports.view"><NuxtLink to="/winlose/reports/weekly"><span>Weekly report</span><small>Weeks by HQ and platform, and settlements</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="winlose-reports.view"><NuxtLink to="/winlose/reports/hq-summary"><span>HQ summary</span><small>Totals per HQ, USD and KHR apart</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['platforms.view', 'platforms.manage']"><NuxtLink to="/admin/platforms"><span>Platforms</span><small>Platforms, default rates and currencies</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can :any="['platforms.view', 'platform-assignments.manage']"><NuxtLink to="/admin/platform-assignments"><span>HQ platform assignments</span><small>Which platforms each HQ reports</small><ChevronRight :size="17" /></NuxtLink></Can>
    </section>

    <section v-if="showAdmin" class="link-card">
      <div class="section-label">ADMINISTRATION</div>
      <Can perm="branches.view"><NuxtLink to="/admin/branches"><span>Branches</span><small>Manage business locations</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="roles.view"><NuxtLink to="/admin/roles"><span>Roles &amp; permissions</span><small>Control team access</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="permissions.view"><NuxtLink to="/admin/permissions"><span>Permission codes</span><small>Review available permissions</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="users.view"><NuxtLink to="/admin/users"><span>Users</span><small>Manage team accounts</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="currency-settings.manage"><NuxtLink to="/admin/currency"><span>Currency</span><small>Exchange rate and default currency</small><ChevronRight :size="17" /></NuxtLink></Can>
      <Can perm="audit-logs.view"><NuxtLink to="/admin/audit"><span>Audit log</span><small>Review account activity</small><ChevronRight :size="17" /></NuxtLink></Can>
      <NuxtLink v-if="canManageBackups" to="/admin/backups"><span>Backups</span><small>Database backups and restore</small><ChevronRight :size="17" /></NuxtLink>
    </section>

    <section class="permissions-card">
      <div class="section-label">YOUR PERMISSIONS</div>
      <div class="permission-list"><span v-for="code in permissions" :key="code">{{ code }}</span><em v-if="isSuperAdmin && !permissions.length">All permissions</em><em v-if="!isSuperAdmin && !permissions.length">None loaded</em></div>
    </section>
    <button class="sign-out" @click="onLogout"><LogOut :size="17" /> Sign out</button>
  </section>
</template>

<script setup lang="ts">
import { ChevronRight, LogOut } from 'lucide-vue-next'
const { user, logout } = useAuth()
const { can, isSuperAdmin, permissions, canViewBranches, canViewUsers, canViewRoles, canViewPermissionCodes, canViewAudit, canManageBackups, canViewInventory, canViewSales, canViewPurchases, canViewTransfers, canViewInventoryReports, canViewInventoryExpenses } = usePermissions()
const showAdmin = computed(() => canViewBranches.value || canViewUsers.value || canViewRoles.value || canViewPermissionCodes.value || canViewAudit.value || canManageBackups.value || can('currency-settings.manage'))
const initials = computed(() => (user.value?.full_name || user.value?.username || 'A').split(/\s+/).filter(Boolean).slice(0, 2).map(part => part[0]?.toUpperCase()).join(''))
async function onLogout() { await logout(); navigateTo('/login') }
</script>

<style scoped>
.more-page { max-width: 760px; margin: 0 auto; color: #183054; }
.page-header { margin-bottom: 20px; }
.eyebrow, .section-label { color: #6180ae; font-size: 10px; font-weight: 800; letter-spacing: .14em; }
.page-header h1 { margin: 5px 0 0; color: #102348; font-size: 27px; letter-spacing: -.04em; }
.page-header p:last-child { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.account-card, .link-card, .permissions-card { margin-bottom: 14px; border: 1px solid #e2eaf5; border-radius: 14px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.account-card { display: flex; align-items: center; gap: 12px; padding: 17px; }
.account-avatar { display: grid; flex: 0 0 auto; width: 46px; height: 46px; place-items: center; border-radius: 14px; color: #fff; background: linear-gradient(140deg, #3388ff, #1461e7); font-weight: 800; }
.account-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 4px; }
.account-copy strong { color: #19345d; font-size: 14px; }
.account-copy span, .account-copy small { overflow: hidden; color: #7b8eaa; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.admin-badge { align-self: flex-start; border-radius: 999px; padding: 5px 8px; color: #1769e8; background: #eef5ff; font-size: 10px; font-weight: 700; white-space: nowrap; }
.link-card { overflow: hidden; padding: 15px 17px 5px; }
.section-label { margin-bottom: 8px; }
.link-card a { display: flex; align-items: center; gap: 12px; min-height: 57px; border-top: 1px solid #edf1f7; color: #203a61; text-decoration: none; }
.link-card a span { flex: 1; font-size: 13px; font-weight: 650; }
.link-card a small { margin-left: auto; color: #8a9ab3; font-size: 11px; }
.link-card a svg { color: #9aabc0; }
.link-card a:hover span { color: #1769e8; }
.permissions-card { padding: 16px 17px; }
.permission-list { display: flex; flex-wrap: wrap; gap: 6px; }
.permission-list span, .permission-list em { border: 1px solid #e1e9f4; border-radius: 7px; padding: 5px 7px; color: #456181; background: #f7faff; font: 10px ui-monospace, SFMono-Regular, Menlo, monospace; font-style: normal; }
.permission-list em { color: #7184a2; font-family: inherit; }
.sign-out { display: inline-flex; align-items: center; gap: 8px; border: 1px solid #f0d9dd; border-radius: 10px; padding: 11px 15px; color: #d14356; background: #fff; font-size: 12px; font-weight: 700; cursor: pointer; }
.sign-out:hover { background: #fff5f6; }
@media (max-width: 560px) { .account-card { align-items: flex-start; } .admin-badge { margin-left: auto; } .link-card a { min-height: 62px; } .link-card a small { display: none; } }
</style>