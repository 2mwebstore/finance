<template>
  <Teleport to="body">
    <div v-if="modelValue" class="app-modal-backdrop" @click.self="close">
      <section class="app-modal" :style="{ '--app-modal-width': width }" role="dialog" aria-modal="true" :aria-labelledby="titleId">
        <span class="sheet-handle" aria-hidden="true"></span>
        <header class="app-modal-header">
          <span v-if="$slots.icon" class="app-modal-icon"><slot name="icon" /></span>
          <div class="app-modal-heading">
            <p v-if="eyebrow" class="app-modal-eyebrow">{{ eyebrow }}</p>
            <h2 :id="titleId">{{ title }}</h2>
            <p v-if="subtitle" class="app-modal-subtitle">{{ subtitle }}</p>
          </div>
          <button type="button" class="app-modal-close" aria-label="Close" :disabled="busy" @click="close"><X :size="19" /></button>
        </header>
        <!-- As a form, Enter submits and the footer's submit button works. -->
        <component :is="form ? 'form' : 'div'" class="app-modal-content" @submit.prevent="emit('submit')">
          <div class="app-modal-body"><slot /></div>
          <footer v-if="$slots.footer" class="app-modal-footer"><slot name="footer" /></footer>
        </component>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useEventListener } from '@vueuse/core'
import { X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  eyebrow?: string
  subtitle?: string
  /** Wrap body + footer in a <form> and emit `submit`. */
  form?: boolean
  /** While saving: the modal can't be closed. */
  busy?: boolean
  width?: string
}>(), {
  eyebrow: '',
  subtitle: '',
  form: false,
  busy: false,
  width: '560px',
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; submit: [] }>()

const titleId = `app-modal-${Math.random().toString(36).slice(2, 9)}`

function close() {
  if (!props.busy) emit('update:modelValue', false)
}

useEventListener(window, 'keydown', (event: KeyboardEvent) => {
  if (props.modelValue && event.key === 'Escape') close()
})
</script>

<style scoped>
.app-modal-backdrop { position: fixed; inset: 0; z-index: 100; display: flex; align-items: center; justify-content: center; padding: 20px; background: rgb(8 24 48 / 55%); backdrop-filter: blur(4px); animation: app-modal-fade .18s ease-out; }
.app-modal { position: relative; display: flex; width: min(var(--app-modal-width), 100%); max-height: min(92vh, 820px); flex-direction: column; overflow: hidden; border: 1px solid #e2eaf5; border-radius: 18px; background: #fff; box-shadow: 0 24px 75px rgb(8 24 48 / 28%); animation: app-modal-pop .22s cubic-bezier(.2, .9, .25, 1); }
.sheet-handle { display: none; }
.app-modal-header { position: relative; display: flex; align-items: center; gap: 12px; padding: 20px 22px 16px; border-bottom: 1px solid #edf1f7; }
.app-modal-icon { display: grid; flex: 0 0 auto; width: 44px; height: 44px; place-items: center; border-radius: 13px; color: #1769e8; background: #eef5ff; }
.app-modal-heading { flex: 1; min-width: 0; }
.app-modal-eyebrow { margin: 0 0 4px; color: #8c9ab0; font-size: 10px; font-weight: 800; letter-spacing: .13em; }
.app-modal-header h2 { margin: 0; color: #172b4e; font-size: 19px; font-weight: 750; letter-spacing: -.02em; }
.app-modal-subtitle { margin: 4px 0 0; color: #7083a2; font-size: 13px; }
.app-modal-close { display: grid; flex: 0 0 auto; width: 36px; height: 36px; place-items: center; border: 0; border-radius: 10px; color: #73849d; background: #f3f6fb; cursor: pointer; }
.app-modal-close:hover:not(:disabled) { color: #1e3b66; background: #e9eff8; }
.app-modal-content { display: flex; min-height: 0; flex: 1; flex-direction: column; }
.app-modal-body { overflow-y: auto; padding: 18px 22px; }
.app-modal-footer { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 22px 18px; border-top: 1px solid #edf1f7; background: #fff; }
.app-modal-footer :slotted(button) { min-width: 110px; min-height: 44px; border-radius: 10px; padding: 0 18px; font-size: 14px; font-weight: 700; cursor: pointer; }
@media (max-width: 540px) {
  .app-modal-backdrop { align-items: flex-end; padding: 0; }
  .app-modal { width: 100%; max-height: 94dvh; border: 0; border-radius: 22px 22px 0 0; animation: app-modal-sheet .24s cubic-bezier(.2, .9, .25, 1); }
  .sheet-handle { position: absolute; top: 8px; left: 50%; display: block; width: 38px; height: 4px; border-radius: 999px; background: #d6dfeb; transform: translateX(-50%); }
  .app-modal-header { padding: 22px 18px 14px; }
  .app-modal-body { padding: 16px 18px; }
  .app-modal-footer { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); padding: 12px 18px calc(12px + env(safe-area-inset-bottom)); }
  .app-modal-footer :slotted(button) { min-width: 0; min-height: 48px; }
}
@keyframes app-modal-fade { from { opacity: 0; } }
@keyframes app-modal-pop { from { opacity: 0; transform: scale(.96) translateY(8px); } }
@keyframes app-modal-sheet { from { transform: translateY(100%); } }
</style>
