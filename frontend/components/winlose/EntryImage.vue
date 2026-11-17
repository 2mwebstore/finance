<template>
  <button type="button" class="inline-flex h-7 items-center gap-1 rounded-lg border border-border bg-white px-2 text-[11px] font-semibold text-navy hover:bg-canvas disabled:opacity-60" :disabled="busy" aria-label="Capture as an image" data-capture-skip @click="capture">
    <Camera :size="13" /> {{ busy ? 'Capturing…' : 'Image' }}
  </button>

  <AppModal v-model="open" title="Entry image" :subtitle="title" width="720px">
    <div class="overflow-auto rounded-lg border border-border bg-canvas p-2">
      <img v-if="url" :src="url" :alt="title" class="mx-auto block h-auto max-w-full" />
    </div>
    <!-- Whether the picture is kept in RP's History (on its first Download or Share) -->
    <p v-if="entryId" class="mt-2 flex items-center gap-1.5 text-[11px]" :class="saveState === 'failed' ? 'text-red-600' : 'text-muted'" role="status">
      <History :size="12" class="shrink-0" />
      <template v-if="saveState === 'saving'">Saving to History…</template>
      <template v-else-if="saveState === 'saved'">Saved to History</template>
      <template v-else-if="saveState === 'failed'">Not saved to History. Download or share again to retry.</template>
      <template v-else>Download or Share saves it to History.</template>
    </p>
    <template #footer>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-navy" @click="download"><Download :size="15" /> Download</button>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="sharing" @click="share"><Send :size="15" /> {{ sharing ? 'Sharing…' : 'Share to Telegram' }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { Camera, Download, History, Send } from 'lucide-vue-next'
import type { WinLoseEntryImage } from '~/types/api'
import { useToast } from '~/composables/useToast'

/**
 * Takes a picture of an entry's card (the element with the given id) and opens it in a pop-up to
 * download or share to Telegram. Anything inside the card marked `data-capture-skip` (buttons,
 * reactions) is left out, what is marked `data-capture-show` is shown (the agent column's "AG"),
 * and the group's name is put before the title (in the picture only). Its agents are in a table per
 * currency, USD first, as on the page. Sharing uses the phone's share sheet, where Telegram is one
 * of the choices; where the browser can't share files the image is copied so it can be pasted in
 * Telegram.
 *
 * With an entry (Group summary), the picture's first Download or Share is kept on the server for
 * RP's History, with the data entry's and the group's titles; doing the other with the same picture
 * marks it as that too. It goes up in the background: sharing isn't held up.
 */
const props = defineProps<{ targetId: string; title: string; group?: string; entryId?: string; groupId?: string }>()
const toast = useToast()
const { api } = useApi()

const open = ref(false)
const busy = ref(false)
const sharing = ref(false)
const url = ref('')
let blob: Blob | null = null

const fileName = computed(() => `${props.title.trim().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '') || 'entry'}.png`)

// --- History: the picture kept on the server ---
type ImageAction = 'download' | 'share'
const saveState = ref<'' | 'saving' | 'saved' | 'failed'>('')
// The kept picture's id, once it is up; what was already sent for it.
let kept: Promise<string> | null = null
const marked = new Set<ImageAction>()

// A small copy of the picture's top for History's list (JPG, 720 px wide: sharp on phones).
async function thumbnail(picture: Blob): Promise<Blob | null> {
  try {
    const bitmap = await createImageBitmap(picture)
    const scale = Math.min(1, 720 / bitmap.width)
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(bitmap.width * scale)
    canvas.height = Math.min(Math.round(bitmap.height * scale), 900)
    const context = canvas.getContext('2d')
    if (!context) return null
    context.fillStyle = '#ffffff'
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(bitmap, 0, 0, Math.round(bitmap.width * scale), Math.round(bitmap.height * scale))
    bitmap.close()
    return await new Promise(resolve => canvas.toBlob(resolve, 'image/jpeg', 0.8))
  } catch {
    return null
  }
}

async function upload(picture: Blob, action: ImageAction): Promise<string> {
  const form = new FormData()
  form.append('file', picture, fileName.value)
  const small = await thumbnail(picture)
  if (small) form.append('thumb', small, 'thumb.jpg')
  form.append('action', action)
  if (props.groupId) form.append('group_id', props.groupId)
  const row = await api<WinLoseEntryImage>(`/winlose/entries/${props.entryId}/images`, { method: 'POST', body: form })
  return row.id
}

// Keeps the picture (its first action) or marks the other action on it; not awaited.
function keep(action: ImageAction) {
  const picture = blob
  if (!props.entryId || !picture || marked.has(action)) return
  marked.add(action)
  const first = !kept
  if (first) {
    saveState.value = 'saving'
    kept = upload(picture, action)
  }
  const job = first ? kept! : kept!.then(id => api(`/winlose/images/${id}/actions`, { method: 'POST', body: { action } }))
  job.then(() => {
    if (first && blob === picture) saveState.value = 'saved'
  }).catch(() => {
    marked.delete(action)
    if (!first || blob !== picture) return
    kept = null
    saveState.value = 'failed'
  })
}

watch(open, (isOpen) => {
  if (!isOpen && url.value) {
    URL.revokeObjectURL(url.value)
    url.value = ''
    blob = null
  }
  // A new picture is a new one in History.
  if (!isOpen) {
    kept = null
    marked.clear()
    saveState.value = ''
  }
})

async function capture() {
  const node = document.getElementById(props.targetId)
  if (!node) return
  busy.value = true
  // Puts the card back as it was, whether or not the picture was made.
  let restore = () => {}
  try {
    const { toBlob } = await import('html-to-image')
    // The group's name goes before the entry's title while the picture is taken.
    const heading = node.querySelector('h3')
    const label = document.createElement('span')
    label.textContent = `${props.group} · `
    label.className = 'font-bold text-navy'
    label.setAttribute('data-capture-label', '')
    if (props.group && heading) heading.prepend(label)
    // The card keeps its on-screen size and gets a margin around it: the padding is added outside
    // the width and height, or the right and bottom edges would be cut off.
    // The table's scroll area is opened up so no scrollbar is drawn.
    const scrollers = Array.from(node.querySelectorAll<HTMLElement>('.overflow-x-auto, .overflow-auto'))
    const overflows = scrollers.map(el => el.style.overflow)
    scrollers.forEach(el => { el.style.overflow = 'visible' })
    // What is left out of the table (agents' ✅ / ❌) is hidden while the picture is taken, so its
    // rows are as tall as without it.
    const hidden = Array.from(node.querySelectorAll<HTMLElement>('table [data-capture-skip]'))
    const displays = hidden.map(el => el.style.display)
    hidden.forEach(el => { el.style.display = 'none' })
    // What only the picture has ("AG" over the agents) is shown while it is taken.
    const shown = Array.from(node.querySelectorAll<HTMLElement>('[data-capture-show]'))
    shown.forEach(el => { el.style.display = 'inline' })
    restore = () => {
      label.remove()
      scrollers.forEach((el, i) => { el.style.overflow = overflows[i] ?? '' })
      hidden.forEach((el, i) => { el.style.display = displays[i] ?? '' })
      shown.forEach(el => { el.style.display = '' })
    }
    const width = node.offsetWidth, height = node.offsetHeight, margin = 16
    // Sharp (2x) unless the picture would be huge (100 agents): phone browsers refuse canvases
    // of more than about 16 megapixels, so the sharpness comes down to stay under 12.
    const pixelRatio = Math.max(1, Math.min(2, Math.sqrt(12_000_000 / ((width + margin * 2) * (height + margin * 2)))))
    const result = await toBlob(node, {
      pixelRatio,
      backgroundColor: '#ffffff',
      width: width + margin * 2,
      height: height + margin * 2,
      style: { boxSizing: 'content-box', width: `${width}px`, height: `${height}px`, padding: `${margin}px`, margin: '0' },
      filter: el => !(el instanceof HTMLElement && el.hasAttribute('data-capture-skip')),
    })
    if (!result) throw new Error('empty image')
    blob = result
    url.value = URL.createObjectURL(result)
    open.value = true
  } catch {
    toast.error('Please try again.', { title: 'Couldn’t make the image' })
  } finally {
    restore()
    busy.value = false
  }
}

function download() {
  if (!url.value) return
  const link = document.createElement('a')
  link.href = url.value
  link.download = fileName.value
  link.click()
  keep('download')
}

async function share() {
  if (!blob) return
  const file = new File([blob], fileName.value, { type: 'image/png' })
  sharing.value = true
  try {
    if (navigator.canShare?.({ files: [file] })) {
      await navigator.share({ files: [file], title: props.title })
      keep('share') // after the share sheet: a cancelled share isn't kept
      return
    }
    // No file sharing here (most desktop browsers): copy the image to paste into Telegram.
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
    keep('share')
    toast.success('Paste it in a Telegram chat, or use Download.', { title: 'Image copied' })
    window.open('https://web.telegram.org/', '_blank', 'noopener')
  } catch (e: any) {
    if (e?.name !== 'AbortError') toast.error('Use Download and send the file in Telegram.', { title: 'Couldn’t share the image' })
  } finally {
    sharing.value = false
  }
}
</script>
