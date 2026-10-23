<template>
  <Teleport to="body">
    <div v-if="modelValue && current" class="viewer-backdrop" @click.self="close">
      <section class="viewer" role="dialog" aria-modal="true" :aria-label="`Attachment: ${current.name}`">
        <header class="viewer-header">
          <span class="viewer-icon"><ImageIcon v-if="isImage" :size="18" /><FileText v-else :size="18" /></span>
          <div class="viewer-title">
            <strong>{{ current.name }}</strong>
            <small>{{ transaction?.transaction_number }} · {{ formatBytes(current.size) }}<template v-if="files.length > 1"> · file {{ index + 1 }} of {{ files.length }}</template></small>
          </div>
          <a v-if="objectUrl" class="viewer-button" :href="objectUrl" :download="current.name" title="Download" aria-label="Download"><Download :size="17" /></a>
          <button v-if="objectUrl && !isImage" type="button" class="viewer-button" title="Open in a new tab" aria-label="Open in a new tab" @click="openInTab"><ExternalLink :size="17" /></button>
          <button v-if="canDelete" type="button" class="viewer-button is-danger" title="Delete this file" aria-label="Delete this file" :disabled="deleting" @click="confirmingDelete = true"><Trash2 :size="17" /></button>
          <button type="button" class="viewer-button" aria-label="Close" @click="close"><X :size="18" /></button>
        </header>

        <!-- One tab per file when there is more than one (also ← / → keys). -->
        <nav v-if="files.length > 1" class="viewer-tabs" aria-label="Files">
          <button
            v-for="(file, i) in files"
            :key="file.id"
            type="button"
            class="viewer-tab"
            :class="{ 'is-active': i === index }"
            :aria-current="i === index ? 'true' : undefined"
            @click="show(i)"
          >
            <ImageIcon v-if="file.type.startsWith('image/')" :size="14" /><FileText v-else :size="14" />
            <span>{{ file.name }}</span>
          </button>
        </nav>

        <div v-if="confirmingDelete" class="viewer-confirm" role="alertdialog" aria-label="Delete this file?">
          <span><strong>Delete “{{ current.name }}”?</strong> It is removed from this transaction for everyone.</span>
          <div>
            <button type="button" class="viewer-confirm-cancel" :disabled="deleting" @click="confirmingDelete = false">Cancel</button>
            <button type="button" class="viewer-confirm-delete" :disabled="deleting" @click="removeCurrent">{{ deleting ? 'Deleting…' : 'Delete' }}</button>
          </div>
        </div>
        <p v-if="deleteError" class="viewer-delete-error" role="alert">{{ deleteError }}</p>

        <div class="viewer-stage">
          <button v-if="files.length > 1" type="button" class="viewer-step is-prev" aria-label="Previous file" @click="step(-1)"><ChevronLeft :size="22" /></button>
          <div class="viewer-body" :class="{ 'is-pdf': !isImage }">
            <div v-if="loading" class="viewer-state"><span class="viewer-spinner"></span> Loading file…</div>
            <div v-else-if="error" class="viewer-state is-error">
              <span>{{ error }}</span>
              <button type="button" @click="load">Try again</button>
            </div>
            <img
              v-else-if="isImage && objectUrl"
              :src="objectUrl"
              :alt="current.name"
              class="viewer-image"
              :class="{ 'is-zoomed': zoomed }"
              :title="zoomed ? 'Click to fit' : 'Click to zoom'"
              @click="zoomed = !zoomed"
            />
            <iframe v-else-if="objectUrl" :src="objectUrl" class="viewer-pdf" :title="current.name"></iframe>
          </div>
          <button v-if="files.length > 1" type="button" class="viewer-step is-next" aria-label="Next file" @click="step(1)"><ChevronRight :size="22" /></button>
        </div>

        <footer v-if="descriptionHtml" class="viewer-description">
          <span>Description</span>
          <!-- Sanitized with DOMPurify (allow-list of formatting tags only). -->
          <div class="rich-text-content" v-html="descriptionHtml"></div>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useEventListener } from '@vueuse/core'
import { ChevronLeft, ChevronRight, Download, ExternalLink, FileText, ImageIcon, Trash2, X } from 'lucide-vue-next'
import type { Transaction } from '~/types/api'
import { sanitizeRichText } from '~/utils/richText'

const props = withDefaults(defineProps<{
  modelValue: boolean
  transaction: Transaction | null
  /** Which file to show first. */
  startIndex?: number
}>(), { startIndex: 0 })
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  /** A file was deleted; the updated transaction. */
  changed: [transaction: Transaction]
}>()

const { api, apiBase } = useApi()
const { getToken, tryRefresh } = useAuth()
const { canEditIncome, canEditExpense, canCreateIncome, canCreateExpense } = usePermissions()
const index = ref(0)
const objectUrl = ref('')
const loading = ref(false)
const error = ref('')
const zoomed = ref(false)
const confirmingDelete = ref(false)
const deleting = ref(false)
const deleteError = ref('')
// Files already fetched in this session of the viewer, so switching back is instant.
const cache = new Map<string, string>()
let request = 0

const files = computed(() => props.transaction?.attachments || [])
const current = computed(() => files.value[index.value] || null)
const isImage = computed(() => !!current.value?.type.startsWith('image/'))
const descriptionHtml = computed(() => sanitizeRichText(props.transaction?.description))
// Same rule as the API: edit rights, or create rights while the transaction isn't approved yet.
const canDelete = computed(() => {
  const txn = props.transaction
  if (!txn || txn.status === 'voided' || txn.status === 'reversed') return false
  const edit = txn.type === 'income' ? canEditIncome.value : canEditExpense.value
  const create = txn.type === 'income' ? canCreateIncome.value : canCreateExpense.value
  return edit || (create && txn.status !== 'approved')
})

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function clearCache() {
  for (const url of cache.values()) URL.revokeObjectURL(url)
  cache.clear()
  objectUrl.value = ''
}

// The files are behind the API's auth, so fetch them with the access token and show them from blob URLs.
async function load() {
  const txn = props.transaction
  const file = current.value
  if (!txn || !file) return
  const ticket = ++request
  zoomed.value = false
  error.value = ''
  const cached = cache.get(file.id)
  if (cached) {
    objectUrl.value = cached
    loading.value = false
    return
  }
  objectUrl.value = ''
  loading.value = true
  const url = `${apiBase}/transactions/${txn.id}/attachments/${file.id}`
  const get = () => {
    const token = getToken()
    return fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {}, credentials: 'include' })
  }
  try {
    let res = await get()
    if (res.status === 401 && await tryRefresh()) res = await get()
    if (!res.ok) throw new Error(res.status === 404 ? 'This file is no longer available.' : 'Could not load the file.')
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    cache.set(file.id, blobUrl)
    if (ticket === request) objectUrl.value = blobUrl
  } catch (e: any) {
    if (ticket === request) error.value = e?.message || 'Could not load the file.'
  } finally {
    if (ticket === request) loading.value = false
  }
}

function show(i: number) {
  if (!files.value.length) return
  index.value = (i + files.value.length) % files.value.length
  confirmingDelete.value = false
  deleteError.value = ''
  void load()
}

function step(by: number) {
  show(index.value + by)
}

async function removeCurrent() {
  const txn = props.transaction
  const file = current.value
  if (!txn || !file) return
  deleting.value = true
  deleteError.value = ''
  try {
    const updated = await api<Transaction>(`/transactions/${txn.id}/attachments/${file.id}`, { method: 'DELETE' })
    const url = cache.get(file.id)
    if (url) URL.revokeObjectURL(url)
    cache.delete(file.id)
    confirmingDelete.value = false
    emit('changed', updated)
    if (!updated.attachments?.length) close()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this file.'
  } finally {
    deleting.value = false
  }
}

function openInTab() {
  if (objectUrl.value) window.open(objectUrl.value, '_blank', 'noopener')
}

function close() {
  emit('update:modelValue', false)
}

watch(() => [props.modelValue, props.transaction?.id], ([open]) => {
  clearCache()
  request++
  confirmingDelete.value = false
  deleteError.value = ''
  if (open) show(props.startIndex)
})
// After a delete the list is shorter: stay in range and show what is now in this slot.
watch(() => files.value.map(file => file.id).join(','), (ids, previous) => {
  if (!props.modelValue || ids === previous || !files.value.length) return
  show(Math.min(index.value, files.value.length - 1))
})
useEventListener(window, 'keydown', (event: KeyboardEvent) => {
  if (!props.modelValue) return
  if (event.key === 'Escape') close()
  else if (event.key === 'ArrowLeft' && files.value.length > 1) step(-1)
  else if (event.key === 'ArrowRight' && files.value.length > 1) step(1)
})
onBeforeUnmount(clearCache)
</script>

<style scoped>
.viewer-backdrop { position: fixed; inset: 0; z-index: 140; display: flex; align-items: center; justify-content: center; padding: 20px; background: rgb(6 16 34 / 78%); backdrop-filter: blur(6px); animation: viewer-fade .18s ease-out; }
.viewer { display: flex; width: min(920px, 100%); max-height: calc(100dvh - 40px); flex-direction: column; overflow: hidden; border-radius: 18px; background: #0f1b2f; box-shadow: 0 30px 80px rgb(0 0 0 / 45%); animation: viewer-pop .22s cubic-bezier(.2, .9, .25, 1); }
.viewer-header { display: flex; align-items: center; gap: 10px; padding: 12px 14px; color: #e8eef8; background: #14243d; }
.viewer-icon { display: grid; flex: 0 0 auto; width: 34px; height: 34px; place-items: center; border-radius: 10px; color: #9cc3ff; background: rgb(255 255 255 / 8%); }
.viewer-title { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 2px; }
.viewer-title strong { overflow: hidden; font-size: 14px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.viewer-title small { color: #93a6c4; font-size: 12px; }
.viewer-button { display: grid; flex: 0 0 auto; width: 36px; height: 36px; place-items: center; border: 0; border-radius: 10px; color: #dbe6f6; background: rgb(255 255 255 / 8%); cursor: pointer; text-decoration: none; }
.viewer-button:hover { background: rgb(255 255 255 / 16%); }
.viewer-button.is-danger:hover { color: #ffb3bd; background: rgb(255 90 110 / 18%); }
.viewer-button:disabled { opacity: .5; cursor: wait; }
.viewer-tabs { display: flex; gap: 6px; overflow-x: auto; padding: 0 14px 10px; background: #14243d; }
.viewer-tab { display: inline-flex; min-width: 0; max-width: 260px; flex: 0 1 auto; align-items: center; gap: 6px; border: 1px solid rgb(255 255 255 / 12%); border-radius: 999px; padding: 6px 12px; color: #b9c8de; background: transparent; font-size: 12px; font-weight: 650; cursor: pointer; }
.viewer-tab span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.viewer-tab:hover { color: #fff; background: rgb(255 255 255 / 8%); }
.viewer-tab.is-active { border-color: #5b9bff; color: #fff; background: rgb(91 155 255 / 22%); }
.viewer-confirm { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px; padding: 10px 14px; color: #ffe1e5; background: #3a1a26; font-size: 13px; }
.viewer-confirm strong { color: #fff; }
.viewer-confirm div { display: flex; gap: 8px; }
.viewer-confirm button { border-radius: 8px; padding: 7px 14px; font-size: 13px; font-weight: 700; cursor: pointer; }
.viewer-confirm-cancel { border: 1px solid rgb(255 255 255 / 25%); color: #fff; background: transparent; }
.viewer-confirm-delete { border: 0; color: #fff; background: #e2445c; }
.viewer-confirm button:disabled { opacity: .6; cursor: wait; }
.viewer-delete-error { margin: 0; padding: 8px 14px; color: #ffb3bd; background: #2a1520; font-size: 13px; }
/* Holds the scrolling body and the ← → buttons, which stay put while a zoomed image scrolls. */
.viewer-stage { position: relative; display: flex; min-height: 0; flex: 1; }
.viewer-step { position: absolute; top: 50%; z-index: 1; display: grid; width: 40px; height: 40px; place-items: center; border: 0; border-radius: 50%; color: #fff; background: rgb(6 16 34 / 62%); cursor: pointer; transform: translateY(-50%); }
.viewer-step:hover { background: rgb(6 16 34 / 85%); }
.viewer-step.is-prev { left: 10px; }
.viewer-step.is-next { right: 10px; }
.viewer-body { display: flex; min-height: 240px; flex: 1; align-items: center; justify-content: center; overflow: auto; background: #0b1424; }
.viewer-body.is-pdf { align-items: stretch; background: #525659; }
.viewer-image { display: block; max-width: 100%; max-height: calc(100dvh - 190px); margin: auto; object-fit: contain; cursor: zoom-in; }
.viewer-image.is-zoomed { max-width: none; max-height: none; cursor: zoom-out; }
.viewer-pdf { width: 100%; height: calc(100dvh - 180px); min-height: 360px; border: 0; background: #fff; }
.viewer-state { display: flex; align-items: center; gap: 10px; padding: 40px 20px; color: #c9d6ea; font-size: 14px; }
.viewer-state.is-error { flex-direction: column; color: #ffb3bd; }
.viewer-state button { border: 1px solid rgb(255 255 255 / 25%); border-radius: 9px; padding: 7px 14px; color: #fff; background: transparent; font-size: 13px; cursor: pointer; }
.viewer-spinner { width: 18px; height: 18px; border: 2px solid rgb(255 255 255 / 25%); border-top-color: #fff; border-radius: 50%; animation: viewer-spin .7s linear infinite; }
.viewer-description { max-height: 140px; overflow-y: auto; padding: 12px 16px 14px; color: #dbe6f6; background: #14243d; font-size: 13px; line-height: 1.5; }
.viewer-description > span { display: block; margin-bottom: 4px; color: #8ea3c3; font-size: 10px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; }
.viewer-description :deep(a) { color: #9cc3ff; }
@media (max-width: 640px) {
  .viewer-backdrop { padding: 0; }
  .viewer { width: 100%; height: 100dvh; max-height: none; border-radius: 0; }
  .viewer-header { padding: calc(10px + env(safe-area-inset-top)) 12px 10px; }
  .viewer-image { max-height: calc(100dvh - 160px); }
  .viewer-pdf { height: auto; flex: 1; }
  .viewer-description { padding-bottom: calc(14px + env(safe-area-inset-bottom)); }
}
@keyframes viewer-fade { from { opacity: 0; } }
@keyframes viewer-pop { from { opacity: 0; transform: scale(.96) translateY(8px); } }
@keyframes viewer-spin { to { transform: rotate(360deg); } }
</style>
