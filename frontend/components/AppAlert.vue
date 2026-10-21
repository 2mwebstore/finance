<template>
  <div class="app-alert" :class="[`is-${type}`, { 'is-compact': compact }]" :role="type === 'error' || type === 'warning' ? 'alert' : 'status'">
    <span class="alert-icon" aria-hidden="true"><component :is="icon" :size="compact ? 15 : 17" :stroke-width="2.5" /></span>
    <div class="alert-copy">
      <strong v-if="title">{{ title }}</strong>
      <p v-if="message || $slots.default"><slot>{{ sentence(message) }}</slot></p>
      <ul v-if="items.length" class="alert-list">
        <li v-for="item in items" :key="item">{{ sentence(item) }}</li>
      </ul>
      <div v-if="$slots.action" class="alert-action"><slot name="action" /></div>
    </div>
    <button v-if="dismissible" type="button" class="alert-close" aria-label="Dismiss" @click="emit('dismiss')"><X :size="15" /></button>
  </div>
</template>

<script setup lang="ts">
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-vue-next'
import { sentence, type AlertType } from '~/composables/useToast'

/** Inline message, mostly for form errors next to the button that failed. Pop-up results use useToast(). */
const props = withDefaults(defineProps<{
  type?: AlertType
  title?: string
  message?: string
  /** Several problems at once, e.g. every field a form check found. */
  items?: string[]
  dismissible?: boolean
  /** Single line, smaller icon: for tight spots like dialogs. */
  compact?: boolean
}>(), {
  type: 'error',
  title: '',
  message: '',
  items: () => [],
  dismissible: false,
  compact: false,
})
const emit = defineEmits<{ dismiss: [] }>()

const icon = computed(() => ({ success: CheckCircle2, error: XCircle, warning: AlertTriangle, info: Info })[props.type])
</script>

<style scoped>
.app-alert {
  --tone: #e2445c; --tone-2: #ff7a8c; --tone-ink: #9f1f35; --tone-soft: #fff1f3; --tone-line: #f8ccd4;
  position: relative; display: flex; align-items: flex-start; gap: 11px; overflow: hidden;
  border: 1px solid var(--tone-line); border-radius: 14px; padding: 11px 14px 11px 12px;
  background: radial-gradient(120% 140% at 0% 0%, var(--tone-soft) 0%, #fff 72%);
  box-shadow: 0 6px 18px -10px var(--tone);
  animation: alert-in .32s cubic-bezier(.2, .9, .25, 1.2);
}
.app-alert::before { content: ''; position: absolute; top: 0; bottom: 0; left: 0; width: 4px; background: linear-gradient(180deg, var(--tone-2), var(--tone)); }
.app-alert.is-error { animation: alert-in .32s cubic-bezier(.2, .9, .25, 1.2), alert-shake .42s .1s ease-in-out; }
.app-alert.is-success { --tone: #109c6f; --tone-2: #37d39c; --tone-ink: #0b6b4c; --tone-soft: #e9faf3; --tone-line: #bfeedb; }
.app-alert.is-warning { --tone: #e08a0b; --tone-2: #fbbf24; --tone-ink: #8a5306; --tone-soft: #fff7e6; --tone-line: #f6dca8; }
.app-alert.is-info { --tone: #1f6fe5; --tone-2: #5b9bff; --tone-ink: #154a9a; --tone-soft: #eef5ff; --tone-line: #cfe0fb; }
.alert-icon { display: grid; flex: 0 0 auto; width: 30px; height: 30px; margin-left: 2px; place-items: center; border-radius: 10px; color: #fff; background: linear-gradient(140deg, var(--tone-2), var(--tone)); box-shadow: 0 5px 12px -4px var(--tone); }
.alert-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 2px; padding-top: 1px; }
.alert-copy strong { color: var(--tone-ink); font-size: 13px; font-weight: 750; line-height: 1.3; }
.alert-copy p { margin: 0; color: #4c5f7d; font-size: 13px; line-height: 1.45; overflow-wrap: anywhere; }
.alert-copy strong + p { color: #5f7190; font-size: 12.5px; }
.app-alert:not(:has(strong)) .alert-copy { justify-content: center; min-height: 30px; }
.app-alert:not(:has(strong)) .alert-copy p { color: var(--tone-ink); font-weight: 600; }
.alert-list { display: flex; flex-direction: column; gap: 3px; margin: 3px 0 0; padding: 0; list-style: none; }
.alert-list li { position: relative; padding-left: 13px; color: #4c5f7d; font-size: 12.5px; line-height: 1.4; overflow-wrap: anywhere; }
.alert-list li::before { content: ''; position: absolute; top: .55em; left: 2px; width: 5px; height: 5px; border-radius: 50%; background: var(--tone); }
.alert-action { margin-top: 6px; }
.alert-action :slotted(button) { border: 0; border-radius: 8px; padding: 6px 12px; color: #fff; background: linear-gradient(140deg, var(--tone-2), var(--tone)); font-size: 12px; font-weight: 700; cursor: pointer; }
.alert-close { display: grid; flex: 0 0 auto; width: 26px; height: 26px; margin: 2px -4px 0 0; place-items: center; border: 0; border-radius: 8px; color: #8796ad; background: transparent; cursor: pointer; }
.alert-close:hover { color: var(--tone-ink); background: rgb(255 255 255 / 70%); }
.app-alert.is-compact { align-items: center; gap: 9px; border-radius: 12px; padding: 8px 12px 8px 10px; }
.app-alert.is-compact .alert-icon { width: 26px; height: 26px; border-radius: 8px; }
.app-alert.is-compact .alert-copy { min-height: 0; padding-top: 0; }
.app-alert.is-compact .alert-copy p { font-size: 12.5px; }
@keyframes alert-in { from { opacity: 0; transform: translateY(-6px) scale(.98); } }
@keyframes alert-shake { 20%, 60% { transform: translateX(-4px); } 40%, 80% { transform: translateX(4px); } }
@media (prefers-reduced-motion: reduce) { .app-alert, .app-alert.is-error { animation: none; } }
</style>
