<template>
  <section class="manage-page">
    <header class="page-header">
      <p class="eyebrow">ADMINISTRATION</p>
      <h1>Manage</h1>
      <p>Choose an area to manage your business.</p>
    </header>

    <ClientOnly>
      <div v-if="manageItems.length" class="manage-grid">
        <NuxtLink v-for="item in manageItems" :key="item.label" :to="item.to" class="manage-card">
          <span class="manage-icon"><component :is="item.icon" :size="21" :stroke-width="1.9" /></span>
          <h2>{{ item.label }}</h2>
          <p>{{ item.description }}</p>
          <ChevronRight class="manage-arrow" :size="17" />
        </NuxtLink>
      </div>

      <div v-else class="manage-empty">No management sections are available for your account.</div>

      <template #fallback>
        <div class="manage-empty">Loading management sections…</div>
      </template>
    </ClientOnly>
  </section>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { Building2, ChevronRight, KeyRound, ReceiptText, ShieldCheck, Tags, UsersRound } from 'lucide-vue-next'

type ManageItem = { label: string; description: string; to: string; icon: Component }

const { can, canViewBranches, canViewUsers, canViewRoles, canViewAudit } = usePermissions()

const manageItems = computed<ManageItem[]>(() => {
  const items: ManageItem[] = []
  if (canViewBranches.value) items.push({ label: 'Branches', description: 'Create and update business locations.', to: '/admin/branches', icon: Building2 })
  if (canViewRoles.value) items.push({ label: 'Roles & Permissions', description: 'Manage team roles and access.', to: '/admin/roles', icon: ShieldCheck })
  if (canViewRoles.value) items.push({ label: 'Permission Codes', description: 'Review the permissions available in the system.', to: '/admin/permissions', icon: KeyRound })
  if (canViewUsers.value) items.push({ label: 'Users', description: 'Manage user accounts, roles, and status.', to: '/admin/users', icon: UsersRound })
  if (can('categories.manage')) items.push({ label: 'Categories', description: 'Manage expense categories.', to: '/admin/categories', icon: Tags })
  if (canViewAudit.value) items.push({ label: 'Audit Log', description: 'Review account activity and filter by date.', to: '/admin/audit', icon: ReceiptText })
  return items
})
</script>

<style scoped>
.manage-page { color: #183054; }
.page-header { margin-bottom: 20px; }
.eyebrow { margin: 0; color: #6180ae; font-size: 10px; font-weight: 800; letter-spacing: .14em; }
.page-header h1 { margin: 5px 0 0; color: #102348; font-size: 27px; letter-spacing: -.04em; }
.page-header p:last-child { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.manage-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 13px; }
.manage-card { position: relative; display: block; min-width: 0; min-height: 150px; border: 1px solid #e2eaf5; border-radius: 14px; padding: 16px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); text-decoration: none; transition: transform .16s ease, box-shadow .16s ease, border-color .16s ease; }
.manage-card:hover { transform: translateY(-2px); border-color: #c6dcfb; box-shadow: 0 10px 24px rgb(24 60 110 / 10%); }
.manage-icon { display: grid; width: 44px; height: 44px; place-items: center; border-radius: 13px; color: #1769e8; background: #eef5ff; }
.manage-card h2 { margin: 14px 0 0; color: #1a3155; font-size: 14px; font-weight: 720; }
.manage-card p { margin: 6px 0 0; color: #7e90ac; font-size: 12px; line-height: 1.5; }
.manage-arrow { position: absolute; top: 18px; right: 16px; color: #b3c1d3; }
.manage-card:hover .manage-arrow { color: #1769e8; }
.manage-empty { border: 1px solid #e2eaf5; border-radius: 14px; padding: 24px; color: #7083a2; background: #fff; font-size: 13px; text-align: center; }
@media (min-width: 1024px) { .manage-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 420px) { .manage-grid { grid-template-columns: 1fr; } .manage-card { min-height: 0; } }
</style>