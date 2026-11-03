<template>
  <!-- Put under a list: asks for more when it comes near the bottom of the screen. -->
  <div ref="sentinel" class="flex min-h-6 items-center justify-center py-2 text-xs text-muted" aria-live="polite">
    <span v-if="loading" class="inline-flex items-center gap-2"><span class="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-border border-t-primary" aria-hidden="true"></span> Loading more…</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ hasMore: boolean; loading: boolean; count: number }>()
const emit = defineEmits<{ more: [] }>()

const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

function watchSentinel() {
  observer?.disconnect()
  if (!sentinel.value || !props.hasMore) return
  // A new look each time: when the rows loaded still leave the end of the list on screen,
  // the observer says so again and the next page comes.
  observer = new IntersectionObserver((entries) => {
    if (entries.some(entry => entry.isIntersecting) && props.hasMore && !props.loading) emit('more')
  }, { rootMargin: '400px 0px' })
  observer.observe(sentinel.value)
}

onMounted(watchSentinel)
watch(() => [props.hasMore, props.loading, props.count], () => nextTick(watchSentinel))
onBeforeUnmount(() => observer?.disconnect())
</script>
