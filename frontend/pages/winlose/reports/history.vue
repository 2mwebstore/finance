<template>
  <!-- RP's History: Group summary images that were downloaded or shared to Telegram, newest first -->
  <section class="rp-compact space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP</p>
        <h1 class="text-xl font-bold text-navy">History</h1>
        <p class="text-xs text-muted">Images from Group summary that were downloaded or shared to Telegram.</p>
      </div>
      <NuxtLink to="/winlose/reports/branch-summary" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><ChartColumn :size="16" /> Group Summary</NuxtLink>
    </header>

    <div class="grid grid-cols-1 gap-2" :class="rp.show.value ? 'md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)]' : 'md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'">
      <ListSearch v-model="search" placeholder="Search by data entry or group…" />
      <!-- Only for people who work in more than one branch (chosen once for all the RP pages) -->
      <SearchSelect v-if="rp.show.value" v-model="rp.branchId.value" :options="rp.optionsFor(true)" :searchable="false" aria-label="Branch" />
      <!-- The days the images were saved (empty: any day) -->
      <WinloseDateFilter v-model="saved" allow-all label="Saved" />
    </div>

    <div v-if="loading && !images.length" class="text-sm text-muted">Loading history…</div>
    <AppAlert v-else-if="error && !images.length" title="Couldn’t load history" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <div v-else-if="!images.length" class="rounded-xl border border-dashed border-border bg-white px-4 py-10 text-center">
      <History :size="22" class="mx-auto text-muted" />
      <p class="mt-2 text-sm font-semibold text-navy">{{ search ? 'No images match this search.' : saved.from ? 'No images saved in these dates.' : 'No images yet' }}</p>
      <p v-if="!search" class="mt-1 text-xs text-muted">In Group summary, an entry’s Image that is downloaded or shared to Telegram shows up here.</p>
    </div>
    <template v-else>
      <ul class="grid grid-cols-1 gap-3 transition-opacity min-[460px]:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4" :class="loading ? 'opacity-60' : ''" :aria-busy="loading">
        <li v-for="img in images" :key="img.id">
          <article class="flex h-full flex-col overflow-hidden rounded-xl border border-border bg-white transition hover:border-primary/40 hover:shadow-sm">
            <!-- The picture's top, full width (its title and first rows), and its titles: open it -->
            <button type="button" class="block w-full text-left focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-primary" :aria-label="`View ${titleOf(img)}`" @click="view(img)">
              <span class="block h-40 w-full overflow-hidden border-b border-border bg-white">
                <WinloseHistoryImage :id="img.id" thumb :alt="titleOf(img)" img-class="block h-auto w-full" />
              </span>
              <span class="block space-y-1 px-3 pt-3">
                <!-- The data entry's title (and branch) on one line, then the group's -->
                <span class="flex items-center gap-1.5 text-sm text-navy"><FileText :size="13" class="shrink-0 text-muted" /><span class="min-w-0 truncate"><span class="font-semibold">{{ img.entry_title }}</span><span v-if="rp.show.value && img.branch_code" class="text-[11px] text-muted"> · {{ img.branch_name || img.branch_code }}</span></span></span>
                <span class="flex items-center gap-1.5 text-xs text-navy"><Network :size="12" class="shrink-0 text-muted" /><span class="truncate">{{ img.group_title || 'No group' }}</span></span>
              </span>
            </button>
            <div class="flex flex-1 flex-col gap-1.5 px-3 pb-3 pt-1.5">
              <!-- The group's ✅ / ❌ / 📤 in the data entry (as by its title in Group summary) -->
              <WinloseEntryReactions v-if="reactionsUrl(img)" :url="reactionsUrl(img)" :reactions="img.reactions || []" />
              <span class="mt-auto flex flex-wrap items-center gap-1">
                <span v-if="img.downloaded_at" class="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-[10px] font-semibold text-primary"><Download :size="11" /> Downloaded</span>
                <span v-if="img.shared_at" class="inline-flex items-center gap-1 rounded-full bg-sky-50 px-2 py-0.5 text-[10px] font-semibold text-sky-700"><Send :size="11" /> Shared</span>
                <span v-if="!img.entry_id" class="inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600">Entry deleted</span>
              </span>
              <span class="text-[10px] text-muted">{{ formatDocDateTime(img.created_at) }}<template v-if="img.created_by"> · <span class="font-medium text-navy">{{ img.created_by }}</span></template></span>
            </div>
          </article>
        </li>
      </ul>
      <InventoryPager v-model:page="page" v-model:page-size="pageSize" :total="total" :sizes="[12, 24, 48]" />
    </template>

    <!-- One image, full size, to download again or open its entry in Group summary -->
    <AppModal v-model="viewerOpen" :title="viewing ? titleOf(viewing) : 'Image'" :subtitle="viewing ? subtitleOf(viewing) : ''" width="720px">
      <div v-if="viewing" class="overflow-auto rounded-lg border border-border bg-canvas p-2">
        <WinloseHistoryImage :id="viewing.id" :alt="titleOf(viewing)" img-class="mx-auto block h-auto max-w-full" @loaded="fullUrl = $event" />
      </div>
      <template #footer>
        <button type="button" class="inline-flex items-center justify-center gap-1.5 rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-navy disabled:opacity-50" :disabled="!viewing?.entry_id" :title="viewing?.entry_id ? 'Open this data entry in Group summary' : 'The data entry was deleted'" @click="openInSummary"><ChartColumn :size="15" /> Group summary</button>
        <button type="button" class="inline-flex items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="!fullUrl" @click="download"><Download :size="15" /> Download</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { ChartColumn, Download, FileText, History, Network, Send } from 'lucide-vue-next'
import type { WinLoseEntryImage } from '~/types/api'
import { useRpBranch } from '~/composables/useRpBranch'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatDocDateTime } from '~/utils/inventory'

const { api } = useApi()
const router = useRouter()

// The branch the RP pages work in (only for people with several).
const rp = useRpBranch()
const search = ref('')
const saved = ref(currentMonthRange()) // this month; All dates in the filter

// One page at a time (Previous / Next, 12, 24 or 48 a page), newest first.
const images = ref<WinLoseEntryImage[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const loading = ref(false)
const error = ref('')
let request = 0 // answers to an earlier request are stale
async function load() {
  const mine = ++request
  loading.value = true
  error.value = ''
  try {
    const res = await api<{ items: WinLoseEntryImage[]; total: number }>('/winlose/images', {
      query: { search: search.value.trim() || undefined, created_from: saved.value.from || undefined, created_to: saved.value.to || undefined, branch_id: rp.branchId.value || undefined, page: page.value, page_size: pageSize.value },
    })
    if (mine !== request) return
    images.value = res.items || []
    total.value = res.total || 0
    // Past the last page (fewer images now): the last one.
    const last = Math.max(1, Math.ceil(total.value / pageSize.value))
    if (page.value > last) page.value = last
  } catch (e: any) {
    if (mine === request) error.value = e?.data?.error || 'Could not load history.'
  } finally {
    if (mine === request) loading.value = false
  }
}
// Another search, branch, dates or page size start from page 1 again.
function fromFirstPage() {
  if (page.value === 1) load()
  else page.value = 1
}
watch(search, fromFirstPage)
watch(saved, fromFirstPage, { deep: true })
watch(() => rp.branchId.value, fromFirstPage)
watch(pageSize, fromFirstPage)
watch(page, () => {
  load()
  if (import.meta.client) window.scrollTo({ top: 0, behavior: 'smooth' })
})
onMounted(load)

// A picture's ✅ / ❌ / 📤: its group's in the data entry, or the entry's when it isn't of one group
// (none once the entry is deleted).
function reactionsUrl(img: WinLoseEntryImage) {
  if (!img.entry_id) return ''
  return img.group_id ? `/winlose/entries/${img.entry_id}/groups/${img.group_id}/reactions` : `/winlose/entries/${img.entry_id}/reactions`
}

// The data entry's title first, then the group's.
const titleOf = (img: WinLoseEntryImage) => img.group_title ? `${img.entry_title} · ${img.group_title}` : img.entry_title
const subtitleOf = (img: WinLoseEntryImage) => {
  const done = [img.downloaded_at && 'downloaded', img.shared_at && 'shared'].filter(Boolean).join(' and ')
  return `${done.charAt(0).toUpperCase()}${done.slice(1)} ${formatDocDateTime(img.created_at)}${img.created_by ? ` by ${img.created_by}` : ''}`
}

// --- one image ---
const viewing = ref<WinLoseEntryImage | null>(null)
const viewerOpen = ref(false)
const fullUrl = ref('')
function view(img: WinLoseEntryImage) {
  fullUrl.value = ''
  viewing.value = img
  viewerOpen.value = true
}
function download() {
  const img = viewing.value
  if (!img || !fullUrl.value) return
  const link = document.createElement('a')
  link.href = fullUrl.value
  link.download = `${titleOf(img).replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '') || 'entry'}.png`
  link.click()
}
// Group summary on the image's data entry and group.
function openInSummary() {
  const img = viewing.value
  if (!img?.entry_id) return
  viewerOpen.value = false
  router.push({ path: '/winlose/reports/branch-summary', query: { entry: img.entry_id, week: img.week_start, ...(img.group_id ? { group: img.group_id } : {}) } })
}
</script>
