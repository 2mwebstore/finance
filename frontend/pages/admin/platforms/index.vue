<template>
  <section class="rp-compact space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP</p>
        <h1 class="text-xl font-bold text-navy">Agents</h1>
      </div>
      <div class="flex gap-2">
        <NuxtLink v-if="can('platform-assignments.view', 'platform-assignments.create', 'platform-assignments.edit', 'platform-assignments.delete')" to="/admin/platform-assignments" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><Network :size="16" /> Group assignments</NuxtLink>
        <!-- Add and Update open the agent's own page (not a pop-up) -->
        <NuxtLink v-if="canCreate" to="/admin/platforms/new" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover"><Plus :size="16" /> Add agent</NuxtLink>
      </div>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by user or name…" /></div>
      <!-- Only for people who work in more than one branch -->
      <div v-if="showBranch" class="sm:w-56"><SearchSelect v-model="branchFilter" :options="branchOptions" aria-label="Branch" /></div>
      <label class="flex items-center gap-2 text-xs font-medium text-navy"><input v-model="showInactive" type="checkbox" class="accent-primary" /> Show turned-off agents</label>
    </div>

    <div v-if="loading && !items.length" class="text-sm text-muted">Loading agents…</div>
    <AppAlert v-else-if="error && !items.length" title="Couldn’t load agents" :message="error"><template #action><button type="button" @click="reset">Try again</button></template></AppAlert>
    <!-- Like Data entry: on phones the buttons are icons only -->
    <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-2 py-2 font-bold sm:px-3">User</th>
            <!-- On phones the currency goes under the user -->
            <th class="hidden px-2 py-2 font-bold sm:table-cell sm:px-3">Currency</th>
            <th v-if="showBranch" class="hidden px-2 py-2 font-bold sm:table-cell sm:px-3">Branch</th>
            <th class="px-2 py-2 text-right font-bold sm:px-3">Groups</th>
            <th v-if="canEdit || canDelete" class="px-2 py-2 sm:px-3"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="platform in shown" :key="platform.id" :class="platform.is_active ? 'text-navy' : 'bg-canvas/60 text-slate-400'">
            <td class="px-2 py-2.5 sm:px-3">
              <span class="whitespace-nowrap font-mono font-semibold">{{ platform.code }}</span>
              <span v-if="!platform.is_active" class="ml-1.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
              <span class="block text-[11px] text-muted"><span class="sm:hidden">{{ platform.default_currency }}</span><template v-if="platform.name && platform.name !== platform.code"><span class="sm:hidden"> · </span>{{ platform.name }}</template></span>
            </td>
            <td class="hidden px-2 py-2.5 sm:table-cell sm:px-3">{{ platform.default_currency }}</td>
            <td v-if="showBranch" class="hidden px-2 py-2.5 sm:table-cell sm:px-3"><span v-if="platform.branch_code">{{ platform.branch_code }}</span><span v-else class="text-muted" title="Shared: every branch sees it">Shared</span></td>
            <td class="px-2 py-2.5 text-right tabular-nums sm:px-3">{{ platform.active_hqs }}</td>
            <td v-if="canEdit || canDelete" class="px-2 py-2 sm:px-3">
              <div class="flex justify-end gap-1 sm:gap-1.5">
                <NuxtLink v-if="canEdit" :to="`/admin/platforms/${platform.id}`" class="action-chip action-edit" title="Update"><Pencil :size="13" /> <span class="max-sm:sr-only">Update</span></NuxtLink>
                <button v-if="canEdit" type="button" class="action-chip" :class="platform.is_active ? 'action-danger' : 'action-success'" :title="platform.is_active ? 'Turn off' : 'Turn on'" :disabled="busyId === platform.id" @click="platform.is_active ? (turningOff = platform) : setActive(platform, true)"><Power :size="13" /> <span class="max-sm:sr-only">{{ platform.is_active ? 'Turn off' : 'Turn on' }}</span></button>
                <button v-if="canDelete" type="button" class="action-chip action-danger" title="Delete" @click="openDelete(platform)"><Trash2 :size="13" /> <span class="max-sm:sr-only">Delete</span></button>
              </div>
            </td>
          </tr>
          <tr v-if="!shown.length"><td :colspan="(canEdit || canDelete ? 4 : 3) + (showBranch ? 1 : 0)" class="px-3 py-8 text-center text-sm text-muted">{{ search ? `No agent matches “${search}”.` : 'No agents yet.' }}</td></tr>
        </tbody>
      </table>
    </div>
    <InfiniteLoader :has-more="hasMore" :loading="loadingMore" :count="items.length" @more="loadMore" />

    <!-- Delete: only an agent never in a group or entered; a used one can only be turned off -->
    <AppModal :model-value="!!deleting" form :title="deleting ? `Delete ${deleting.code}?` : ''" eyebrow="DELETE AGENT" subtitle="Only an agent that was never in a group or entered can be deleted; a used one can only be turned off." :busy="deleteBusy" width="460px" @update:model-value="(open) => { if (!open && !deleteBusy) deleting = null }" @submit="deleteAgent">
      <AppAlert v-if="deleteError" compact :message="deleteError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="deleteBusy" @click="deleting = null">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="deleteBusy">{{ deleteBusy ? 'Deleting…' : 'Delete agent' }}</button>
      </template>
    </AppModal>

    <!-- Turning an agent off needs a reason (the audit log keeps it); turning on doesn't -->
    <InventoryReasonDialog :model-value="!!turningOff" :title="turningOff ? `Turn off ${turningOff.code}?` : ''" eyebrow="TURN OFF AGENT" subtitle="It and its weeks are hidden from Data entry and the reports until it is turned on again; nothing is deleted. The audit log keeps your reason." label="Reason for turning off" confirm-label="Turn off" danger :busy="!!turningOff && busyId === turningOff.id" :server-error="turnOffError" @update:model-value="(open) => { if (!open) { turningOff = null; turnOffError = '' } }" @confirm="(reason) => turningOff && setActive(turningOff, false, reason)" />
  </section>
</template>

<script setup lang="ts">
import { Network, Pencil, Plus, Power, Trash2 } from 'lucide-vue-next'
import type { Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useInfiniteList } from '~/composables/useInfiniteList'
import { useBranchOptions } from '~/composables/useBranchOptions'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
// Each action has its own permission (Add, Update, Delete Agents); turning off or on is an update.
const canCreate = computed(() => can('platforms.create'))
const canEdit = computed(() => can('platforms.edit'))
const canDelete = computed(() => can('platforms.delete'))

const search = ref('')
const showInactive = ref(false)
// Agents belong to a branch. People who work in one branch just get theirs (nothing to choose);
// with several, a Branch filter (and a Branch choice on the agent's page).
const { branches, options: branchOptions } = useBranchOptions('All branches')
const showBranch = computed(() => branches.value.length > 1)
const branchFilter = ref('')
// Agents load a page at a time as the list is scrolled (search and turned-off agents are worked out
// by the API); turning one off or on, or deleting it, fetches the pages already shown again (Add and
// Update are on the agent's own page, which comes back here).
const { items, loading, loadingMore, error, hasMore, reset, loadMore, refresh: load } = useInfiniteList<Platform>(
  async (page, size) => await api<{ items: Platform[]; total: number }>('/platforms', {
    query: { include_inactive: showInactive.value ? 'true' : undefined, search: search.value.trim() || undefined, branch_id: branchFilter.value || undefined, page, page_size: size },
  }),
  platform => platform.id, 30, 'Could not load agents.',
)
const shown = computed(() => items.value)
// Another search or filter starts from the top again.
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(reset, 300)
})
watch([showInactive, branchFilter], reset)
onMounted(reset)

const busyId = ref('')

const deleting = ref<Platform | null>(null)
const deleteBusy = ref(false)
const deleteError = ref('')
function openDelete(platform: Platform) {
  deleteError.value = ''
  deleting.value = platform
}
async function deleteAgent() {
  const p = deleting.value
  if (!p) return
  deleteBusy.value = true
  deleteError.value = ''
  try {
    await api(`/platforms/${p.id}`, { method: 'DELETE' })
    deleting.value = null
    toast.success(`${p.code} is removed.`, { title: 'Agent deleted' })
    await load()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this agent.'
  } finally {
    deleteBusy.value = false
  }
}

const turningOff = ref<Platform | null>(null)
const turnOffError = ref('')
async function setActive(platform: Platform, active: boolean, reason = '') {
  busyId.value = platform.id
  turnOffError.value = ''
  try {
    await api(`/platforms/${platform.id}`, {
      method: 'PATCH',
      body: { code: platform.code, name: platform.name, default_currency: platform.default_currency, is_active: active, reason: reason || undefined },
    })
    turningOff.value = null
    toast.success(`${platform.code} is turned ${active ? 'on' : 'off'}.`)
    await load()
  } catch (e: any) {
    // Turning off: the error stays in the window (e.g. groups still have the agent)
    if (active) toast.error(e?.data?.error || 'Could not change this agent.')
    else turnOffError.value = e?.data?.error || 'Could not turn this agent off.'
  } finally {
    busyId.value = ''
  }
}
</script>
