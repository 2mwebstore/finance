<template>
  <Teleport to="body">
    <div v-if="modelValue && transaction?.attachment" class="viewer-backdrop" @click.self="close">
      <section class="viewer" role="dialog" aria-modal="true" :aria-label="`Attachment: ${transaction.attachment.name}`">
        <header class="viewer-header">
          <span class="viewer-icon"><ImageIcon v-if="isImage" :size="18" /><FileText v-else :size="18" /></span>
          <div class="viewer-title">
            <strong>{{ transaction.attachment.name }}</strong>
            <small>{{ transaction.transaction_number }} · {{ formatBytes(transaction.attachment.size) }}</small>
          </div>
          <a v-if="objectUrl" class="viewer-button" :href="objectUrl" :download="transaction.attachment.name" title="Download" aria-label="Download"><Download :size="17" /></a>
          <button v-if="objectUrl && !isImage" type="button" class="viewer-button" title="Open in a new tab" aria-label="Open in a new tab" @click="openInTab"><ExternalLink :size="17" /></button>
          <button type="button" class="viewer-button" aria-label="Close" @click="close"><X :size="18" /></button>
        </header>

        <div class="viewer-body" :class="{ 'is-pdf': !isImage }">
          <div v-if="loading" class="viewer-state"><span class="viewer-spinner"></span> Loading file…</div>
          <div v-else-if="error" class="viewer-state is-error">
            <span>{{ error }}</span>
            <button type="button" @click="load">Try again</button>
          </div>
          <img
            v-else-if="isImage && objectUrl"
            :src="objectUrl"
            :alt="transaction.attachment.name"
            class="viewer-image"
            :class="{ 'is-zoomed': zoomed }"
            :title="zoomed ? 'Click to fit' : 'Click to zoom'"
            @click="zoomed = !zoomed"
          />
          <iframe v-else-if="objectUrl" :src="objectUrl" class="viewer-pdf" :title="transaction.attachment.name"></iframe>
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
import { Download, ExternalLink, FileText, ImageIcon, X } from 'lucide-vue-next'
import type { Transaction } from '~/types/api'
import { sanitizeRichText } from '~/utils/richText'

const props = defineProps<{ modelValue: boolean; transaction: Transaction | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { apiBase } = useApi()
const { getToken, tryRefresh } = useAuth()
const objectUrl = ref('')
const loading = ref(false)
const error = ref('')
const zoomed = ref(false)
let request = 0

const isImage = computed(() => !!props.transaction?.attachment?.type.startsWith('image/'))
const descriptionHtml = computed(() => sanitizeRichText(props.transaction?.description))

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function revoke() {
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

// The file is behind the API's auth, so fetch it with the access token and show it from a blob URL.
async function load() {
  const txn = props.transaction
  if (!txn?.attachment) return
  const current = ++request
  revoke()
  zoomed.value = false
  error.value = ''
  loading.value = true
  const url = `${apiBase}/transactions/${txn.id}/attachment`
  const get = () => {
    const token = getToken()
    return fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {}, credentials: 'include' })
  }
  try {
    let res = await get()
    if (res.status === 401 && await tryRefresh()) res = await get()
    if (!res.ok) throw new Error(res.status === 404 ? 'This file is no longer available.' : 'Could not load the file.')
    const blob = await res.blob()
    if (current !== request) return
    objectUrl.value = URL.createObjectURL(blob)
  } catch (e: any) {
    if (current === request) error.value = e?.message || 'Could not load the file.'
  } finally {
    if (current === request) loading.value = false
  }
}

function openInTab() {
  if (objectUrl.value) window.open(objectUrl.value, '_blank', 'noopener')
}

function close() {
  emit('update:modelValue', false)
}

watch(() => [props.modelValue, props.transaction?.id], ([open]) => {
  if (open) void load()
  else {
    request++
    revoke()
  }
})
useEventListener(window, 'keydown', (event: KeyboardEvent) => {
  if (props.modelValue && event.key === 'Escape') close()
})
onBeforeUnmount(revoke)
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
