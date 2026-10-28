<template>
  <span class="relative inline-grid shrink-0 place-items-center overflow-hidden rounded-xl bg-blue-50 text-primary" :class="sizeClass">
    <img v-if="src" :src="src" :alt="product.name" class="h-full w-full" :class="size === 'xl' ? 'max-h-[70vh] object-contain' : 'object-cover'" loading="lazy" referrerpolicy="no-referrer" @error="failed = true" />
    <Package v-else :size="iconSize" />
  </span>
</template>

<script setup lang="ts">
import { Package } from 'lucide-vue-next'

/**
 * A product's picture: a web address as it is, or an uploaded picture fetched with the access
 * token (the API keeps uploads behind sign-in) and shown from a blob URL. Falls back to an icon.
 */
const props = withDefaults(defineProps<{
  product: { id: string; name: string; has_image?: boolean; image_url?: string; updated_at?: string }
  size?: 'sm' | 'md' | 'lg' | 'xl'
}>(), { size: 'md' })

const { apiBase } = useApi()
const { getToken, tryRefresh } = useAuth()
const blobUrl = ref('')
const failed = ref(false)
const sizeClass = computed(() => ({ sm: 'h-9 w-9', md: 'h-11 w-11', lg: 'h-24 w-24', xl: 'min-h-[220px] w-full !bg-canvas' }[props.size]))
const iconSize = computed(() => ({ sm: 16, md: 18, lg: 30, xl: 48 }[props.size]))
const src = computed(() => failed.value ? '' : props.product.image_url || blobUrl.value)

// One fetch per picture version for the whole page session.
const cache = useState<Record<string, string>>('product-image-cache', () => ({}))

async function load() {
  failed.value = false
  blobUrl.value = ''
  if (props.product.image_url || !props.product.has_image) return
  const key = `${props.product.id}:${props.product.updated_at || ''}`
  if (cache.value[key]) { blobUrl.value = cache.value[key]!; return }
  const url = `${apiBase}/inventory/products/${props.product.id}/image`
  const get = () => {
    const token = getToken()
    return fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {}, credentials: 'include' })
  }
  try {
    let res = await get()
    if (res.status === 401 && await tryRefresh()) res = await get()
    if (!res.ok) throw new Error()
    const objectUrl = URL.createObjectURL(await res.blob())
    cache.value[key] = objectUrl
    blobUrl.value = objectUrl
  } catch {
    failed.value = true
  }
}

watch(() => [props.product.id, props.product.has_image, props.product.image_url, props.product.updated_at], load, { immediate: true })
</script>
