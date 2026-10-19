<template>
  <div class="app-shell">
    <aside class="sidebar">
      <NuxtLink to="/dashboard" class="brand-lockup">
        <span class="brand-mark"><ChartNoAxesCombined :size="23" :stroke-width="2.4" /></span>
        <span class="brand-copy"><strong>Income &amp; Expense</strong><small>Manage your business</small></span>
      </NuxtLink>

      <nav class="sidebar-nav" aria-label="Main navigation">
        <div class="nav-section">
          <NuxtLink v-for="item in mainNav" :key="item.label" :to="item.to" class="sidebar-link" :class="isActive(item) ? 'is-active' : ''">
            <component :is="item.icon" :size="18" :stroke-width="1.9" />
            <span>{{ item.label }}</span>
            <ChevronRight v-if="item.label === 'Branches' || item.label === 'Roles & Permissions'" class="nav-chevron" :size="15" />
          </NuxtLink>
        </div>

        <div v-if="adminNav.length" class="nav-divider"></div>
        <div v-if="adminNav.length" class="nav-section">
          <p class="nav-caption">MANAGE</p>
          <NuxtLink v-for="item in adminNav" :key="item.label" :to="item.to" class="sidebar-link" :class="isActive(item) ? 'is-active' : ''">
            <component :is="item.icon" :size="18" :stroke-width="1.9" />
            <span>{{ item.label }}</span>
            <ChevronRight v-if="item.label === 'Branches' || item.label === 'Roles & Permissions'" class="nav-chevron" :size="15" />
          </NuxtLink>
        </div>
      </nav>

      <div class="sidebar-profile">
        <div class="profile-avatar">{{ initials }}</div>
        <div class="sidebar-profile-copy"><strong>{{ user?.full_name || 'Account' }}</strong><small>{{ profileRole }}</small></div>
        <button class="logout-button" title="Sign out" aria-label="Sign out" @click="onLogout"><LogOut :size="16" /></button>
      </div>
    </aside>

    <div class="app-content">
      <header class="topbar">
        <div class="mobile-brand">
          <span class="brand-mark small-brand-mark"><ChartNoAxesCombined :size="19" /></span>
          <strong>Income &amp; Expense</strong>
        </div>
        <form v-if="canViewTransactions" class="global-search" role="search" @submit.prevent="submitSearch">
          <Search :size="17" />
          <input v-model="searchTerm" type="search" placeholder="Search transactions…" aria-label="Search transactions" />
          <kbd>↵</kbd>
        </form>
        <div class="topbar-spacer"></div>
        <NuxtLink to="/profile" class="topbar-user" aria-label="Open your profile">
          <div class="profile-avatar topbar-avatar">{{ initials }}</div>
          <div class="topbar-user-copy"><strong>{{ user?.full_name || 'Account' }}</strong><small>{{ profileRole }}</small></div>
          <ChevronDown :size="15" class="topbar-chevron" />
        </NuxtLink>
        <button class="mobile-logout" aria-label="Sign out" @click="onLogout"><LogOut :size="17" /></button>
      </header>

      <main class="page-content">
        <slot />
      </main>
    </div>

    <div v-if="mobileMenuOpen" class="mobile-menu-backdrop" @click="mobileMenuOpen = false"></div>
    <aside class="mobile-sheet" :class="{ 'is-open': mobileMenuOpen }" role="dialog" aria-modal="true" aria-label="Menu" :inert="!mobileMenuOpen">
      <div class="mobile-sheet-header">
        <span class="sheet-handle"></span>
        <strong>Menu</strong>
        <button type="button" class="sheet-close" aria-label="Close menu" @click="mobileMenuOpen = false"><X :size="18" /></button>
      </div>
      <div class="mobile-sheet-body">
        <template v-for="section in menuSections" :key="section.caption">
          <p class="sheet-caption">{{ section.caption }}</p>
          <div class="sheet-grid">
            <NuxtLink v-for="item in section.items" :key="`sheet-${item.label}`" :to="item.to" class="sheet-tile" :class="isActive(item) ? 'is-active' : ''" @click="mobileMenuOpen = false">
              <span class="sheet-tile-icon"><component :is="item.icon" :size="20" :stroke-width="1.9" /></span>
              <span>{{ item.label }}</span>
            </NuxtLink>
          </div>
        </template>
      </div>
      <NuxtLink to="/profile" class="drawer-profile" @click="mobileMenuOpen = false"><div class="profile-avatar">{{ initials }}</div><span><strong>{{ user?.full_name || 'Account' }}</strong><small>{{ profileRole }}</small></span><ChevronRight :size="16" /></NuxtLink>
    </aside>

    <nav class="mobile-nav" aria-label="Mobile navigation">
      <NuxtLink v-for="item in mobileNav" :key="item.label" :to="item.to" :class="[isActive(item) ? 'is-active' : '', item.center ? 'mobile-nav-center' : '']">
        <span v-if="item.center" class="mobile-nav-fab"><component :is="item.icon" :size="22" /></span>
        <component :is="item.icon" v-else :size="19" />
        <span>{{ item.label }}</span>
      </NuxtLink>
      <button type="button" class="mobile-nav-menu" :class="mobileMenuOpen || menuRouteActive ? 'is-active' : ''" :aria-expanded="mobileMenuOpen" @click="mobileMenuOpen = true"><LayoutGrid :size="19" /><span>Menu</span></button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import {
  Building2,
  ChartNoAxesCombined,
  ChevronDown,
  ChevronRight,
  FileChartColumnIncreasing,
  House,
  KeyRound,
  LayoutDashboard,
  LayoutGrid,
  LogOut,
  Menu,
  ReceiptText,
  Search,
  ShieldCheck,
  Tags,
  TrendingDown,
  TrendingUp,
  UsersRound,
  X,
} from 'lucide-vue-next'

type NavItem = { label: string; to: string; icon: Component; queryType?: string; center?: boolean }

const route = useRoute()
const { user, logout } = useAuth()
const {
  can,
  canViewDashboard,
  canViewTransactions,
  canViewReports,
  canViewBranches,
  canViewUsers,
  canViewRoles,
  canViewAudit,
  isSuperAdmin,
} = usePermissions()
const searchTerm = ref('')
const mobileMenuOpen = ref(false)

const mainNav = computed<NavItem[]>(() => {
  const items: NavItem[] = []
  if (canViewDashboard.value) items.push({ label: 'Dashboard', to: '/dashboard', icon: LayoutDashboard })
  if (canViewTransactions.value) items.push({ label: 'Transactions', to: '/transactions', icon: ReceiptText })
  if (can('income.view')) items.push({ label: 'Income', to: '/transactions?type=income', icon: TrendingUp, queryType: 'income' })
  if (can('expense.view')) items.push({ label: 'Expense', to: '/transactions?type=expense', icon: TrendingDown, queryType: 'expense' })
  if (canViewReports.value) items.push({ label: 'Reports', to: '/reports', icon: FileChartColumnIncreasing })
  return items
})

const adminNav = computed<NavItem[]>(() => {
  const items: NavItem[] = []
  if (canViewBranches.value) items.push({ label: 'Branches', to: '/admin/branches', icon: Building2 })
  if (canViewRoles.value) items.push({ label: 'Roles & Permissions', to: '/admin/roles', icon: ShieldCheck })
  if (canViewRoles.value) items.push({ label: 'Permission Codes', to: '/admin/permissions', icon: KeyRound })
  if (canViewUsers.value) items.push({ label: 'Users', to: '/admin/users', icon: UsersRound })
  if (canViewAudit.value) items.push({ label: 'Audit Log', to: '/admin/audit', icon: ReceiptText })
  if (can('categories.manage')) items.push({ label: 'Categories', to: '/admin/categories', icon: Tags })
  return items
})

// Bottom bar: Home, Income, Activity (raised centre button), Expense, then Menu.
const mobileNav = computed<NavItem[]>(() => {
  const items: NavItem[] = []
  if (canViewDashboard.value) items.push({ label: 'Home', to: '/dashboard', icon: House })
  if (can('income.view')) items.push({ label: 'Income', to: '/transactions?type=income', icon: TrendingUp, queryType: 'income' })
  if (canViewTransactions.value) items.push({ label: 'Activity', to: '/transactions', icon: ReceiptText, center: true })
  if (can('expense.view')) items.push({ label: 'Expense', to: '/transactions?type=expense', icon: TrendingDown, queryType: 'expense' })
  return items
})

// Everything the bottom bar does not cover lives in the Menu sheet's grid.
const menuSections = computed(() => [
  { caption: 'MANAGE', items: adminNav.value },
  { caption: 'MORE', items: canViewReports.value ? [{ label: 'Reports', to: '/reports', icon: FileChartColumnIncreasing }] : [] },
].filter((section) => section.items.length))

const menuRouteActive = computed(() => !mobileNav.value.some(isActive))

watch(() => route.fullPath, () => { mobileMenuOpen.value = false })

const initials = computed(() => (user.value?.full_name || user.value?.username || 'A')
  .split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join(''))
const profileRole = computed(() => isSuperAdmin.value ? 'Super Administrator' : user.value?.roles?.[0] || 'Administrator')

function isActive(item: NavItem) {
  const path = item.to.split('?')[0]
  if (item.queryType) return route.path === path && route.query.type === item.queryType
  if (path === '/dashboard') return route.path === '/dashboard' || route.path === '/'
  if (path === '/transactions') return route.path.startsWith('/transactions') && route.path !== '/transactions/new' && route.query.type !== 'income' && route.query.type !== 'expense'
  if (path === '/transactions/new') return route.path === path
  return route.path === path || route.path.startsWith(path + '/')
}

function submitSearch() {
  const search = searchTerm.value.trim()
  if (!search) return
  const type = route.path === '/transactions' && (route.query.type === 'income' || route.query.type === 'expense')
    ? route.query.type
    : undefined
  navigateTo({ path: '/transactions', query: { search, ...(type ? { type } : {}) } })
}

async function onLogout() {
  await logout()
  navigateTo('/login')
}
</script>

<style scoped>
.app-shell { display: flex; min-height: 100vh; background: #f3f7fd; }
.sidebar { position: sticky; top: 0; display: flex; flex: 0 0 244px; flex-direction: column; width: 244px; height: 100vh; color: #fff; background: linear-gradient(180deg, #102f5d 0%, #0d2852 53%, #0a234b 100%); box-shadow: 5px 0 20px rgb(18 46 88 / 9%); z-index: 30; }
.brand-lockup { display: flex; align-items: center; gap: 11px; min-height: 72px; padding: 13px 18px; border-bottom: 1px solid rgb(255 255 255 / 9%); color: #fff; text-decoration: none; }
.brand-mark { display: flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 38px; height: 38px; border-radius: 11px; color: #fff; background: linear-gradient(140deg, #2785ff, #1161d8); box-shadow: 0 6px 13px rgb(0 0 0 / 16%); }
.brand-copy { display: flex; min-width: 0; flex-direction: column; gap: 4px; }
.brand-copy strong { overflow: hidden; font-size: 14px; line-height: 1.1; font-weight: 720; text-overflow: ellipsis; white-space: nowrap; }
.brand-copy small { color: #b4c4dc; font-size: 11px; }
.sidebar-nav { flex: 1; overflow-y: auto; padding: 14px 10px; }
.nav-section { display: flex; flex-direction: column; gap: 3px; }
.sidebar-link { display: flex; align-items: center; gap: 11px; min-height: 42px; padding: 0 12px; border-radius: 9px; color: #d3dff1; font-size: 13px; font-weight: 550; text-decoration: none; transition: color .15s ease, background .15s ease; }
.sidebar-link svg { flex: 0 0 auto; color: #d7e4f5; }
.sidebar-link:hover { color: #fff; background: rgb(255 255 255 / 9%); }
.sidebar-link.is-active { color: #fff; background: linear-gradient(100deg, #287ef2, #1263df); box-shadow: 0 5px 12px rgb(0 0 0 / 14%); }
.sidebar-link.is-active svg { color: #fff; }
.nav-chevron { margin-left: auto; opacity: .8; }
.nav-divider { height: 1px; margin: 14px 10px 12px; background: rgb(255 255 255 / 11%); }
.nav-caption { margin: 0 12px 7px; color: #8098ba; font-size: 10px; font-weight: 750; letter-spacing: .13em; }
.sidebar-profile { display: flex; align-items: center; gap: 9px; padding: 13px 13px; border-top: 1px solid rgb(255 255 255 / 10%); }
.profile-avatar { display: flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 36px; height: 36px; border: 2px solid rgb(255 255 255 / 75%); border-radius: 50%; color: #31517f; background: linear-gradient(135deg, #e5f0ff, #afc8ed); font-size: 11px; font-weight: 800; }
.sidebar-profile-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 3px; }
.sidebar-profile-copy strong { overflow: hidden; color: #fff; font-size: 12px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.sidebar-profile-copy small { overflow: hidden; color: #9eb3d1; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.logout-button { display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; border: 0; border-radius: 8px; color: #b8c8df; background: transparent; cursor: pointer; }
.logout-button:hover { color: #fff; background: rgb(255 255 255 / 10%); }
.app-content { display: flex; flex: 1; min-width: 0; flex-direction: column; }
.topbar { position: sticky; top: 0; z-index: 20; display: flex; align-items: center; gap: 18px; min-height: 64px; padding: 0 23px; border-bottom: 1px solid #e7edf5; background: rgb(255 255 255 / 96%); backdrop-filter: blur(12px); }
.global-search { display: flex; align-items: center; gap: 9px; width: min(490px, 54vw); height: 38px; padding: 0 11px; border: 1px solid #dfe7f2; border-radius: 9px; color: #7d91b0; background: #fff; box-shadow: inset 0 1px 2px rgb(20 50 90 / 2%); }
.global-search:focus-within { border-color: #8ab6fa; box-shadow: 0 0 0 3px rgb(37 99 235 / 9%); }
.global-search input { width: 100%; border: 0; outline: 0; color: #223d66; background: transparent; font: inherit; font-size: 12px; }
.global-search input::placeholder { color: #899bb5; }
.global-search kbd { padding: 2px 5px; border: 1px solid #e4eaf3; border-radius: 4px; color: #91a1b7; font-size: 10px; }
.topbar-spacer { flex: 1; }
.topbar-user { display: flex; align-items: center; gap: 9px; min-width: 0; padding: 4px 8px 4px 4px; border-radius: 10px; color: inherit; text-decoration: none; transition: background .15s ease; }
.topbar-user:hover { background: #f5f8fd; }
.topbar-user:hover .topbar-avatar { box-shadow: 0 0 0 3px #e6effe; }
.topbar-avatar { width: 33px; height: 33px; }
.topbar-user-copy { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.topbar-user-copy strong { overflow: hidden; max-width: 150px; color: #1b3156; font-size: 12px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.topbar-user-copy small { color: #8293ad; font-size: 10px; }
.topbar-chevron { color: #8293ad; }
.mobile-brand, .mobile-logout{ display: none; }
.mobile-menu-backdrop { display: none; }
.mobile-sheet { display: none; }
.page-content { width: 100%; max-width: 1700px; margin: 0 auto; padding: 26px 28px 36px; }
.mobile-nav { display: none; }
@media (max-width: 900px) {
  .sidebar { display: none; }
  .topbar { min-height: 62px; padding: 0 20px; }
  .page-content { padding: 22px 20px 32px; }
}
@media (max-width: 760px) {
  .topbar { min-height: 60px; justify-content: space-between; padding: 0 14px; gap: 10px; }
  .mobile-brand { display: flex; align-items: center; gap: 8px; min-width: 0; color: #18355e; font-size: 13px; }
  .mobile-brand strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .small-brand-mark { width: 31px; height: 31px; border-radius: 9px; }
  .global-search { display: none; }
  .topbar-spacer, .topbar-user-copy, .topbar-chevron { display: none; }
  .topbar-user { margin-left: auto; border: 0; }
  .topbar-avatar { width: 33px; height: 33px; }
  .mobile-logout { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; border: 1px solid #e1e9f4; border-radius: 10px; color: #607898; background: #fff; cursor: pointer; }
  .page-content { padding: 18px 14px calc(92px + env(safe-area-inset-bottom)); }
  .mobile-nav { position: fixed; right: 10px; bottom: calc(8px + env(safe-area-inset-bottom)); left: 10px; z-index: 40; display: flex; justify-content: space-around; gap: 3px; padding: 7px; border: 1px solid #dfe8f4; border-radius: 17px; background: rgb(255 255 255 / 97%); box-shadow: 0 9px 28px rgb(26 58 100 / 15%); backdrop-filter: blur(12px); }
  .mobile-nav a, .mobile-nav button { display: flex; flex: 1 1 0; min-width: 0; flex-direction: column; align-items: center; justify-content: center; gap: 3px; border: 0; border-radius: 12px; padding: 7px 3px; color: #8998ad; background: transparent; font: inherit; font-size: 10px; font-weight: 650; text-decoration: none; white-space: nowrap; cursor: pointer; }
  .mobile-nav a.is-active, .mobile-nav button.is-active { color: #1769e8; background: #f0f6ff; }
  .mobile-nav .mobile-nav-menu { color: #5c759a; }
  .mobile-nav .mobile-nav-menu.is-active { color: #1769e8; }
  /* Activity: a raised round button that sits on top of the bar. */
  .mobile-nav a.mobile-nav-center, .mobile-nav a.mobile-nav-center.is-active { justify-content: flex-end; padding-top: 0; color: #1b3156; background: transparent; }
  .mobile-nav a.mobile-nav-center.is-active { color: #1769e8; }
  .mobile-nav-fab { display: grid; width: 52px; height: 52px; margin-top: -30px; place-items: center; border: 4px solid #fff; border-radius: 50%; color: #fff; background: linear-gradient(140deg, #2f86ff, #1263df); box-shadow: 0 8px 18px rgb(23 105 232 / 32%); transition: transform .15s ease, box-shadow .15s ease; }
  .mobile-nav a.mobile-nav-center:active .mobile-nav-fab { transform: scale(.94); }
  .mobile-nav a.mobile-nav-center.is-active .mobile-nav-fab { box-shadow: 0 0 0 3px #cfe1ff, 0 8px 18px rgb(23 105 232 / 32%); }
  .mobile-menu-backdrop { position: fixed; inset: 0; z-index: 50; display: block; background: rgb(10 35 75 / 42%); backdrop-filter: blur(2px); }
  /* Menu: a bottom sheet with a grid of everything not in the bar. */
  .mobile-sheet { position: fixed; right: 0; bottom: 0; left: 0; z-index: 60; display: flex; max-height: calc(100dvh - 48px); flex-direction: column; border-radius: 22px 22px 0 0; color: #162a4d; background: #fff; box-shadow: 0 -12px 32px rgb(20 48 86 / 18%); transform: translateY(105%); visibility: hidden; transition: transform .24s ease, visibility 0s linear .24s; }
  .mobile-sheet.is-open { transform: translateY(0); visibility: visible; transition: transform .24s ease; }
  .mobile-sheet-header { position: relative; display: flex; flex-direction: column; align-items: center; padding: 8px 16px 10px; }
  .sheet-handle { width: 38px; height: 4px; margin-bottom: 10px; border-radius: 999px; background: #d6dfeb; }
  .mobile-sheet-header strong { font-size: 16px; font-weight: 750; }
  .sheet-close { position: absolute; top: 12px; right: 14px; display: grid; width: 34px; height: 34px; place-items: center; border: 0; border-radius: 10px; color: #5c759a; background: #f2f6fc; cursor: pointer; }
  .mobile-sheet-body { overflow-y: auto; padding: 4px 16px 12px; }
  .sheet-caption { margin: 10px 2px 10px; color: #6180ae; font-size: 11px; font-weight: 800; letter-spacing: .13em; }
  .sheet-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
  .sheet-tile { display: flex; min-height: 96px; flex-direction: column; align-items: center; justify-content: center; gap: 8px; padding: 12px 6px; border: 1px solid #e4ebf5; border-radius: 14px; color: #1b3156; background: #fff; font-size: 12px; font-weight: 650; line-height: 1.25; text-align: center; text-decoration: none; transition: background .15s ease, border-color .15s ease; }
  .sheet-tile:active { background: #f5f8fd; }
  .sheet-tile-icon { display: grid; width: 42px; height: 42px; place-items: center; border-radius: 12px; color: #1769e8; background: #eef5ff; }
  .sheet-tile.is-active { border-color: #8ab6fa; color: #1769e8; background: #f0f6ff; }
  .sheet-tile.is-active .sheet-tile-icon { color: #fff; background: linear-gradient(140deg, #2f86ff, #1263df); }
  .drawer-profile { display: flex; align-items: center; gap: 10px; padding: 14px 16px calc(14px + env(safe-area-inset-bottom)); border-top: 1px solid #e7edf5; color: #1b3156; text-decoration: none; }
  .drawer-profile .profile-avatar { border-color: #dbe7f8; }
  .drawer-profile > span { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 3px; }
  .drawer-profile strong { overflow: hidden; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
  .drawer-profile small { color: #7b8eaa; font-size: 11px; }
  .drawer-profile > svg { color: #8293ad; }
}
</style>
