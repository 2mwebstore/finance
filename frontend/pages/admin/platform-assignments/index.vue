<template>
  <section class="rp-compact space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP</p>
        <h1 class="text-xl font-bold text-navy">Group agent assignments</h1>
      </div>
      <NuxtLink v-if="canCreate" to="/admin/platform-assignments/new" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover"><Plus :size="16" /> Assign agents</NuxtLink>
    </header>

    <div class="grid grid-cols-1 gap-2" :class="rp.show.value ? 'sm:grid-cols-4' : 'sm:grid-cols-3'">
      <ListSearch v-model="search" placeholder="Search by title…" />
      <!-- Only for people who work in more than one branch (chosen once for all the RP pages) -->
      <SearchSelect v-if="rp.show.value" v-model="rp.branchId.value" :options="rp.optionsFor(true)" :searchable="false" aria-label="Branch" />
      <SearchSelect v-model="platformFilter" :options="[{ value: '', label: 'All agents' }, ...platformOptions]" aria-label="Agent" />
      <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
    </div>

    <div v-if="loading && !groups.length" class="text-sm text-muted">Loading groups…</div>
    <AppAlert v-else-if="error && !groups.length" title="Couldn’t load groups" :message="error"><template #action><button type="button" @click="reset">Try again</button></template></AppAlert>
    <template v-else>
      <article v-for="g in groups" :key="g.id" class="rounded-xl border border-border bg-white p-4" :class="g.is_active ? '' : 'bg-canvas/60'">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div class="min-w-0 flex-1 space-y-2">
            <div>
              <h2 class="flex flex-wrap items-center gap-2 text-sm font-bold" :class="g.is_active ? 'text-navy' : 'text-slate-400'">
                <Network :size="15" class="shrink-0 text-muted" />{{ g.title }}
                <span v-if="!g.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
              </h2>
              <!-- Its agents are in View -->
              <p class="mt-0.5 text-[11px] text-muted"><template v-if="rp.show.value && g.branch_code">{{ g.branch_code }} · </template>From {{ formatDocDate(g.starts_on) }} · {{ g.agents.length }} {{ g.agents.length === 1 ? 'agent' : 'agents' }} · {{ g.weeks }} {{ g.weeks === 1 ? 'week' : 'weeks' }} entered<template v-if="g.created_by"> · by {{ g.created_by }}</template></p>
            </div>
            <!-- Partners a group still has from before (their % decide the Total until it is updated) -->
            <ul v-if="g.partners.length" class="flex flex-wrap gap-1">
              <li v-for="p in g.partners" :key="p.partner_id" class="inline-flex items-center gap-1 rounded-md bg-blue-50 px-1.5 py-0.5 text-[11px] text-navy" :class="p.partner_active ? '' : 'opacity-60'"><Handshake :size="11" />{{ p.partner_name }}<template v-if="!p.partner_active"> (off)</template> · {{ formatRate(p.commission_rate) }}</li>
            </ul>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2">
            <label v-if="canEdit" class="flex items-center gap-2 text-xs font-semibold" :class="g.is_active ? 'text-navy' : 'text-muted'">
              Active
              <input :checked="g.is_active" type="checkbox" role="switch" class="switch" :disabled="busyId === g.id" :aria-label="`${g.title} active`" @change="toggle(g)" />
            </label>
            <div class="flex items-center gap-1.5">
              <NuxtLink :to="`/admin/platform-assignments/view/${g.id}`" class="action-chip"><Eye :size="13" /> View</NuxtLink>
              <NuxtLink v-if="canEdit" :to="`/admin/platform-assignments/${g.id}`" class="action-chip action-edit"><Pencil :size="13" /> Update</NuxtLink>
              <button v-if="canDelete" type="button" class="action-chip action-danger" @click="openDelete(g)"><Trash2 :size="13" /> Delete</button>
            </div>
          </div>
        </div>
      </article>
      <InfiniteLoader :has-more="hasMore" :loading="loadingMore" :count="groups.length" @more="loadMore" />
      <div v-if="!groups.length" class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">{{ search || platformFilter || statusFilter !== 'all' ? 'No groups match these filters.' : 'No groups yet. Use Assign agents to make one.' }}</div>
    </template>

    <!-- Delete: only while no week has been entered -->
    <AppModal :model-value="!!deleting" form :title="deleting ? `Delete “${deleting.title}”?` : ''" eyebrow="DELETE GROUP" :subtitle="deleting && deleting.weeks ? 'Weeks are already entered for this group, so it can’t be deleted. Turn it off instead: it and its weeks are hidden from Data entry and the reports, and nothing is deleted.' : 'The group and its agents’ assignments are removed.'" :busy="saving" width="460px" @update:model-value="(open) => { if (!open && !saving) deleting = null }" @submit="saveDelete">
      <AppAlert v-if="formError" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="deleting = null">Cancel</button>
        <button v-if="deleting && !deleting.weeks" type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Deleting…' : 'Delete group' }}</button>
        <button v-else-if="deleting && deleting.is_active && canEdit" type="button" class="bg-primary text-white disabled:opacity-60" :disabled="saving" @click="turnOffDeleting">Turn it off</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Eye, Handshake, Network, Pencil, Plus, Trash2 } from 'lucide-vue-next'
import type { AssignmentGroup, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useInfiniteList } from '~/composables/useInfiniteList'
import { useRpBranch } from '~/composables/useRpBranch'
import { formatDocDate } from '~/utils/inventory'
import { formatRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
// Each action has its own permission (Add, Update, Delete Group Agent Assignments).
const canCreate = computed(() => can('platform-assignments.create'))
const canEdit = computed(() => can('platform-assignments.edit'))
const canDelete = computed(() => can('platform-assignments.delete'))

const platforms = ref<Platform[]>([])
const platformOptions = computed(() => platforms.value.map(p => ({ value: p.id, label: p.is_active ? p.code : `${p.code} (off)` })))
const statusOptions = [{ value: 'all', label: 'All groups' }, { value: 'active', label: 'Active' }, { value: 'off', label: 'Off' }]

// The person's own groups (the API works the branch out): switched-on first, by title.
// The branch the RP pages work in (only for people with several).
const rp = useRpBranch()
const search = ref('')
const platformFilter = ref('')
const statusFilter = ref('all')
// Groups load a page at a time as the list is scrolled; a save or delete fetches the pages already
// shown again.
const { items: groups, loading, loadingMore, error, hasMore, reset, loadMore, refresh: load } = useInfiniteList<AssignmentGroup>(
  async (page, size) => await api<{ items: AssignmentGroup[]; total: number }>('/platform-groups', {
    query: { search: search.value.trim() || undefined, platform_id: platformFilter.value || undefined, status: statusFilter.value, branch_id: rp.show.value ? rp.branchId.value || 'all' : undefined, page, page_size: size },
  }),
  group => group.id, 30, 'Could not load groups.',
)
// Another search or filter starts from the top again.
watch([platformFilter, statusFilter, () => rp.branchId.value], reset)
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(reset, 300)
})
onMounted(async () => {
  reset()
  try { platforms.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true' } })).items || [] } catch { platforms.value = [] }
})

const saving = ref(false)
const formError = ref('')
const busyId = ref('')


// --- active on/off ---
async function toggle(g: AssignmentGroup) {
  busyId.value = g.id
  try {
    const updated = await api<AssignmentGroup>(`/platform-groups/${g.id}/active`, { method: 'POST', body: { is_active: !g.is_active } })
    toast.success(updated.is_active ? 'It and its weeks show in Data entry and the reports again.' : 'It and its weeks are hidden from Data entry and the reports; nothing is deleted.', { title: `“${g.title}” turned ${updated.is_active ? 'on' : 'off'}` })
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Could not change this group.')
    await load()
  } finally {
    busyId.value = ''
  }
}

// --- delete ---
const deleting = ref<AssignmentGroup | null>(null)
function openDelete(g: AssignmentGroup) {
  formError.value = ''
  deleting.value = g
}
async function saveDelete() {
  const g = deleting.value
  if (!g) return
  saving.value = true
  formError.value = ''
  try {
    await api(`/platform-groups/${g.id}`, { method: 'DELETE' })
    deleting.value = null
    toast.success(`“${g.title}” is removed.`, { title: 'Group deleted' })
    await load()
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not delete the group.'
  } finally {
    saving.value = false
  }
}
async function turnOffDeleting() {
  const g = deleting.value
  if (!g) return
  deleting.value = null
  await toggle(g)
}
</script>

<style scoped>
.switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.switch:checked { background: #1769e8; }
.switch:checked::before { transform: translateX(18px); }
.switch:disabled { cursor: wait; opacity: .6; }
</style>
