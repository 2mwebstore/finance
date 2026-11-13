<template>
  <span class="inline-flex flex-wrap items-center gap-0.5 sm:gap-1">
    <!-- Like Telegram: a pill for each reaction given (yours filled), with who gave it -->
    <button
      v-for="k in given" :key="k.value" type="button"
      class="reaction-pill inline-flex h-6 items-center gap-1 rounded-full pl-1 pr-1.5 text-xs font-semibold tabular-nums transition-colors disabled:opacity-60 sm:h-7 sm:pl-1.5 sm:pr-2"
      :class="[isMine(k.value) ? 'bg-primary text-white' : 'bg-[#e9effd] text-primary hover:bg-[#dce6fc]', popped === k.value ? 'reaction-pop' : '']"
      :aria-pressed="isMine(k.value)" :aria-label="`${k.label}, ${who(k.value).length}: ${who(k.value).map(r => r.user_name).join(', ')}`" :aria-disabled="!canReact || undefined" :disabled="busy"
      @click="canReact ? react(k.value, $event) : show($event, k.value)" @mouseenter="show($event, k.value)" @mouseleave="tip = null" @focus="show($event, k.value)" @blur="tip = null"
    >
      <span class="text-sm leading-none" aria-hidden="true">{{ k.icon }}</span>
      <!-- Up to three people as their initials, more as a number -->
      <span v-if="who(k.value).length <= 3" class="flex -space-x-1.5" aria-hidden="true">
        <span v-for="r in who(k.value)" :key="r.user_id" class="grid h-4 w-4 place-items-center rounded-full text-[7px] font-bold text-white ring-2 sm:h-5 sm:w-5 sm:text-[8px]" :class="[colorOf(r.user_id), isMine(k.value) ? 'ring-primary' : 'ring-[#e9effd]']">{{ initials(r.user_name) }}</span>
      </span>
      <span v-else>{{ who(k.value).length }}</span>
    </button>
    <!-- Add a reaction: a bar with ✅, ❌ and 📤 (report sent) -->
    <button v-if="canReact" ref="adder" type="button" class="grid h-6 w-6 place-items-center sm:h-7 sm:w-7 rounded-full text-slate-400 transition-colors hover:bg-[#e9effd] hover:text-primary disabled:opacity-60" :class="picking ? 'bg-[#e9effd] text-primary' : ''" :aria-expanded="picking" aria-label="Add a reaction" title="React" :disabled="busy" @click="togglePicker">
      <SmilePlus :size="16" />
    </button>

    <Teleport to="body">
      <!-- (fixed, so tables that scroll sideways don't cut these off) -->
      <div v-if="picking" ref="bar" class="reaction-bar fixed z-[80] flex -translate-x-1/2 items-center gap-0.5 rounded-full border border-border bg-white p-1 shadow-xl" :class="picking.below ? '' : '-translate-y-full'" :style="{ left: `${picking.x}px`, top: `${picking.y}px` }" role="menu" aria-label="React">
        <button
          v-for="k in KINDS" :key="k.value" type="button" role="menuitemradio" :aria-checked="isMine(k.value)" :aria-label="k.label"
          class="grid h-9 w-9 place-items-center rounded-full text-xl transition-transform hover:scale-125" :class="isMine(k.value) ? 'bg-[#e9effd]' : ''"
          @click="react(k.value); picking = null"
        >{{ k.icon }}</button>
      </div>
      <div v-if="tip" role="tooltip" class="pointer-events-none fixed z-[80] w-max max-w-[220px] -translate-x-1/2 -translate-y-full rounded-xl bg-navy/95 px-3 py-2 text-[11px] leading-snug text-white shadow-lg" :style="{ left: `${tip.x}px`, top: `${tip.y}px` }">
        <p class="mb-0.5 font-semibold">{{ KINDS.find(k => k.value === tip!.kind)?.icon }} {{ who(tip.kind).length }}</p>
        <p v-for="r in who(tip.kind)" :key="r.user_id" class="flex items-center gap-1.5">
          <span class="grid h-4 w-4 shrink-0 place-items-center rounded-full text-[7px] font-bold" :class="colorOf(r.user_id)">{{ initials(r.user_name) }}</span>{{ r.user_name || 'Someone' }}
        </p>
      </div>
    </Teleport>
  </span>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener } from '@vueuse/core'
import { SmilePlus } from 'lucide-vue-next'
import type { EntryReaction } from '~/types/api'
import { useToast } from '~/composables/useToast'

/**
 * ✅, ❌ and 📤 (report sent) on a data entry (or a group or agent in it), like Telegram's
 * reactions: a pill for each one given (yours filled) with who gave it, and a button that opens a bar
 * to add one. A person has one of ✅ / ❌ (the other changes it) and 📤 beside it; tapping one again
 * takes it back (needs React to Data Entry). Hovering a pill shows who.
 */
// url: where a reaction is sent (an entry's, or an agent's result in it).
const props = defineProps<{ url: string; reactions: EntryReaction[] }>()

type Kind = EntryReaction['reaction']
const KINDS: { value: Kind; icon: string; label: string }[] = [
  { value: 'yes', icon: '✅', label: 'Yes' },
  { value: 'no', icon: '❌', label: 'No' },
  { value: 'sent', icon: '📤', label: 'Report sent' },
]

const { api } = useApi()
const { user } = useAuth()
const { can } = usePermissions()
const canReact = computed(() => can('winlose.react'))
const toast = useToast()
const list = ref<EntryReaction[]>([...props.reactions])
watch(() => props.reactions, (r) => { list.value = [...r] })
// Yours: ✅ or ❌, and 📤 beside it.
const mineList = computed(() => list.value.filter(r => r.user_id === user.value?.id).map(r => r.reaction))
const isMine = (kind: Kind) => mineList.value.includes(kind)
const who = (kind: Kind) => list.value.filter(r => r.reaction === kind)
const given = computed(() => KINDS.filter(k => who(k.value).length))

// "Sokha Owner" → "SO"; each person keeps a colour.
const initials = (name: string) => name.trim().split(/\s+/).slice(0, 2).map(w => w[0]).join('').toUpperCase() || '?'
const COLORS = ['bg-sky-500', 'bg-emerald-500', 'bg-amber-500', 'bg-rose-500', 'bg-violet-500', 'bg-teal-500']
const colorOf = (id: string) => COLORS[[...id].reduce((n, c) => n + c.charCodeAt(0), 0) % COLORS.length]

const busy = ref(false)
const popped = ref<Kind | ''>('')
async function react(kind: Kind, e?: Event) {
  const pill = e?.currentTarget as HTMLElement | undefined // (gone from the event once it has been handled)
  busy.value = true
  try {
    list.value = (await api<{ reactions: EntryReaction[] }>(props.url, { method: 'POST', body: { reaction: kind } })).reactions
    if (isMine(kind)) {
      popped.value = kind
      setTimeout(() => { if (popped.value === kind) popped.value = '' }, 450)
    }
    if (pill?.isConnected) showAt(pill, kind) // on phones there's no hover: show who after a tap
  } catch (err: any) {
    toast.error(err?.data?.error || 'Could not save your reaction.')
  } finally {
    busy.value = false
  }
}

// The bar opens above the button (below it near the top of the screen).
const adder = ref<HTMLElement | null>(null)
const bar = ref<HTMLElement | null>(null)
const picking = ref<{ x: number; y: number; below: boolean } | null>(null)
function togglePicker() {
  tip.value = null
  if (picking.value) return void (picking.value = null)
  const r = adder.value?.getBoundingClientRect()
  if (!r) return
  const below = r.top < 64
  picking.value = { x: Math.min(Math.max(r.left + r.width / 2, 56), window.innerWidth - 56), y: below ? r.bottom + 6 : r.top - 6, below }
}
onClickOutside(bar, () => { picking.value = null }, { ignore: [adder] })

const tip = ref<{ kind: Kind; x: number; y: number } | null>(null)
const show = (e: Event, kind: Kind) => showAt(e.currentTarget as HTMLElement | null, kind)
function showAt(el: HTMLElement | null, kind: Kind) {
  const r = el?.getBoundingClientRect()
  if (!r || picking.value) return
  tip.value = { kind, x: Math.min(Math.max(r.left + r.width / 2, 116), window.innerWidth - 116), y: r.top - 6 }
}
useEventListener('scroll', () => { tip.value = null; picking.value = null }, { capture: true, passive: true })
useEventListener('keydown', (e: KeyboardEvent) => { if (e.key === 'Escape') picking.value = null })
</script>

<style scoped>
/* A reaction given pops, like Telegram's */
.reaction-pop { animation: reaction-pop .45s cubic-bezier(.34, 1.56, .64, 1); }
@keyframes reaction-pop {
  0% { transform: scale(.6); }
  60% { transform: scale(1.15); }
  100% { transform: scale(1); }
}
.reaction-bar { animation: reaction-bar .16s ease-out; }
@keyframes reaction-bar {
  from { opacity: 0; scale: .85; }
  to { opacity: 1; scale: 1; }
}
</style>
