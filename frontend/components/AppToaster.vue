<template>
  <Teleport to="body">
    <div class="toaster" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="toast"
          :class="`is-${toast.type}`"
          :role="toast.type === 'error' ? 'alert' : 'status'"
          :style="{ '--toast-duration': `${toast.duration}ms` }"
        >
          <span class="toast-glow" aria-hidden="true"></span>
          <span class="toast-icon" aria-hidden="true"><component :is="icons[toast.type]" :size="19" :stroke-width="2.5" /></span>
          <div class="toast-copy">
            <strong>{{ toast.title }}</strong>
            <p v-if="toast.message">{{ toast.message }}</p>
          </div>
          <button type="button" class="toast-close" aria-label="Dismiss" @click="dismiss(toast.id)"><X :size="16" /></button>
          <!-- The bar empties over the toast's duration (paused on hover); when it ends, the toast closes. -->
          <span class="toast-timer" aria-hidden="true" @animationend="dismiss(toast.id)"></span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-vue-next'
import { useToast } from '~/composables/useToast'

const { toasts, dismiss } = useToast()
const icons = { success: CheckCircle2, error: XCircle, warning: AlertTriangle, info: Info }
</script>

<style scoped>
.toaster { position: fixed; top: 18px; right: 18px; z-index: 200; display: flex; width: min(380px, calc(100vw - 24px)); flex-direction: column; gap: 10px; pointer-events: none; }
.toast {
  --tone: #e2445c; --tone-2: #ff7a8c; --tone-ink: #9f1f35;
  position: relative; display: flex; align-items: flex-start; gap: 12px; overflow: hidden;
  border: 1px solid rgb(226 234 245 / 90%); border-radius: 16px; padding: 14px 12px 16px 14px;
  background: rgb(255 255 255 / 96%); box-shadow: 0 18px 44px -12px rgb(15 35 70 / 30%), 0 2px 6px rgb(15 35 70 / 6%);
  backdrop-filter: blur(10px); pointer-events: auto;
}
.toast.is-success { --tone: #109c6f; --tone-2: #37d39c; --tone-ink: #0b6b4c; }
.toast.is-warning { --tone: #e08a0b; --tone-2: #fbbf24; --tone-ink: #8a5306; }
.toast.is-info { --tone: #1f6fe5; --tone-2: #5b9bff; --tone-ink: #154a9a; }
/* A soft splash of the tone behind the icon. */
.toast-glow { position: absolute; top: -40px; left: -40px; width: 130px; height: 130px; border-radius: 50%; background: radial-gradient(circle, var(--tone-2) 0%, transparent 68%); opacity: .2; pointer-events: none; }
.toast-icon { position: relative; display: grid; flex: 0 0 auto; width: 38px; height: 38px; place-items: center; border-radius: 12px; color: #fff; background: linear-gradient(140deg, var(--tone-2), var(--tone)); box-shadow: 0 8px 18px -6px var(--tone); animation: toast-icon-pop .5s .08s cubic-bezier(.3, 1.6, .5, 1) both; }
.toast-copy { position: relative; display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 3px; padding-top: 1px; }
.toast-copy strong { color: #14284b; font-size: 14px; font-weight: 750; line-height: 1.3; }
.toast-copy p { margin: 0; color: #5b6e8c; font-size: 13px; line-height: 1.45; overflow-wrap: anywhere; }
.toast-close { position: relative; display: grid; flex: 0 0 auto; width: 28px; height: 28px; place-items: center; border: 0; border-radius: 9px; color: #8a99af; background: transparent; cursor: pointer; }
.toast-close:hover { color: #1e3b66; background: #f0f4fa; }
.toast-timer { position: absolute; right: 0; bottom: 0; left: 0; height: 3px; background: linear-gradient(90deg, var(--tone), var(--tone-2)); transform-origin: left; animation: toast-timer var(--toast-duration) linear forwards; }
.toast:hover .toast-timer, .toast:focus-within .toast-timer { animation-play-state: paused; }
.toast.is-error { animation: toast-shake .45s .42s ease-in-out; }

.toast-enter-active { transition: opacity .3s ease, transform .42s cubic-bezier(.2, 1.1, .3, 1); }
.toast-leave-active { position: absolute; width: 100%; transition: opacity .22s ease, transform .22s ease; }
.toast-move { transition: transform .3s cubic-bezier(.2, .9, .3, 1); }
.toast-enter-from { opacity: 0; transform: translateX(40px) scale(.96); }
.toast-leave-to { opacity: 0; transform: translateX(30px) scale(.96); }

@keyframes toast-timer { from { transform: scaleX(1); } to { transform: scaleX(0); } }
@keyframes toast-icon-pop { from { transform: scale(.4) rotate(-12deg); } }
@keyframes toast-shake { 20%, 60% { transform: translateX(-5px); } 40%, 80% { transform: translateX(5px); } }

/* Phones: drop in from the top, full width. */
@media (max-width: 640px) {
  .toaster { top: calc(10px + env(safe-area-inset-top)); right: 12px; left: 12px; width: auto; }
  .toast-enter-from, .toast-leave-to { transform: translateY(-24px) scale(.97); }
}
@media (prefers-reduced-motion: reduce) {
  .toast, .toast-icon { animation: none; }
  .toast-enter-active, .toast-leave-active, .toast-move { transition: opacity .15s ease; }
  .toast-enter-from, .toast-leave-to { transform: none; }
}
</style>
