<template>
  <img v-if="src" :src="src" :alt="alt" :class="imgClass" />
  <span v-else class="grid h-full min-h-24 w-full place-items-center text-muted" role="img" :aria-label="failed ? 'Image not available' : 'Loading image'">
    <ImageOff v-if="failed" :size="22" />
    <LoaderCircle v-else :size="18" class="animate-spin" />
  </span>
</template>

<script setup lang="ts">
import { ImageOff, LoaderCircle } from 'lucide-vue-next'
import { useEntryImageFiles } from '~/composables/useEntryImageFiles'

/** A picture kept in RP's History (thumb: its small copy, for the list). */
const props = withDefaults(defineProps<{ id: string; alt: string; thumb?: boolean; imgClass?: string }>(), { thumb: false, imgClass: '' })
const emit = defineEmits<{ loaded: [url: string] }>()

const { fileUrl } = useEntryImageFiles()
const src = ref('')
const failed = ref(false)

async function load() {
  src.value = ''
  failed.value = false
  try {
    src.value = await fileUrl(props.id, props.thumb)
    emit('loaded', src.value)
  } catch {
    failed.value = true
  }
}

watch(() => [props.id, props.thumb], load, { immediate: true })
</script>
