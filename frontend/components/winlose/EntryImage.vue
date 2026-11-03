<template>
  <button type="button" class="inline-flex h-7 items-center gap-1 rounded-lg border border-border bg-white px-2 text-[11px] font-semibold text-navy hover:bg-canvas disabled:opacity-60" :disabled="busy" aria-label="Capture as an image" data-capture-skip @click="capture">
    <Camera :size="13" /> {{ busy ? 'Capturing…' : 'Image' }}
  </button>

  <AppModal v-model="open" title="Entry image" :subtitle="title" width="720px">
    <div class="overflow-auto rounded-lg border border-border bg-canvas p-2">
      <img v-if="url" :src="url" :alt="title" class="mx-auto block h-auto max-w-full" />
    </div>
    <template #footer>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-navy" @click="download"><Download :size="15" /> Download</button>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="sharing" @click="share"><Send :size="15" /> {{ sharing ? 'Sharing…' : 'Share to Telegram' }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { Camera, Download, Send } from 'lucide-vue-next'
import { useToast } from '~/composables/useToast'

/**
 * Takes a picture of an entry's card (the element with the given id) and opens it in a pop-up to
 * download or share to Telegram. Anything inside the card marked `data-capture-skip` (buttons,
 * reactions) is left out, and the group's name is put before the title (in the picture only). Sharing uses the phone's share sheet, where Telegram is one of the
 * choices; where the browser can't share files the image is copied so it can be pasted in Telegram.
 */
const props = defineProps<{ targetId: string; title: string; group?: string }>()
const toast = useToast()

const open = ref(false)
const busy = ref(false)
const sharing = ref(false)
const url = ref('')
let blob: Blob | null = null

const fileName = computed(() => `${props.title.trim().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '') || 'entry'}.png`)

watch(open, (isOpen) => {
  if (!isOpen && url.value) {
    URL.revokeObjectURL(url.value)
    url.value = ''
    blob = null
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
    restore = () => {
      label.remove()
      scrollers.forEach((el, i) => { el.style.overflow = overflows[i] ?? '' })
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
}

async function share() {
  if (!blob) return
  const file = new File([blob], fileName.value, { type: 'image/png' })
  sharing.value = true
  try {
    if (navigator.canShare?.({ files: [file] })) {
      await navigator.share({ files: [file], title: props.title })
      return
    }
    // No file sharing here (most desktop browsers): copy the image to paste into Telegram.
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
    toast.success('Paste it in a Telegram chat, or use Download.', { title: 'Image copied' })
    window.open('https://web.telegram.org/', '_blank', 'noopener')
  } catch (e: any) {
    if (e?.name !== 'AbortError') toast.error('Use Download and send the file in Telegram.', { title: 'Couldn’t share the image' })
  } finally {
    sharing.value = false
  }
}
</script>
