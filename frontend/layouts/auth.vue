<template>
  <div class="auth-shell">
    <div class="auth-glow auth-glow-one"></div>
    <div class="auth-glow auth-glow-two"></div>
    <div class="auth-layout">
      <div class="auth-brand"><span class="auth-brand-mark"><ChartNoAxesCombined :size="26" /></span><span>Income &amp; Expense</span></div>
      <div class="auth-card"><slot /></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChartNoAxesCombined } from 'lucide-vue-next'
</script>

<style scoped>
.auth-shell { position: relative; display: flex; min-height: 100vh; min-height: 100dvh; overflow: hidden; align-items: center; justify-content: center; padding: 28px; color: #fff; background: linear-gradient(135deg, #0a234b 0%, #102f5d 52%, #1769e8 160%); }
.auth-layout { position: relative; z-index: 1; display: flex; width: min(400px, 100%); flex-direction: column; gap: 24px; }
.auth-brand { display: flex; align-items: center; justify-content: center; gap: 10px; font-size: 15px; font-weight: 750; }
.auth-brand-mark { display: grid; width: 46px; height: 46px; place-items: center; border-radius: 13px; background: linear-gradient(140deg, #3388ff, #1461e7); box-shadow: 0 8px 18px rgb(0 0 0 / 20%); }
.auth-card :deep(form) { padding: 30px; border: 1px solid #e4ebf5; border-radius: 18px; box-shadow: 0 20px 55px rgb(3 21 52 / 25%); }
.auth-card :deep(form + *) { color: #fff; }
.auth-glow { position: absolute; width: 420px; height: 420px; border-radius: 50%; background: rgb(47 132 255 / 18%); filter: blur(3px); }
.auth-glow-one { top: -200px; right: -100px; }
.auth-glow-two { bottom: -250px; left: -160px; background: rgb(31 193 145 / 10%); }
@media (max-width: 720px) { .auth-shell { padding: 18px; } .auth-layout { gap: 20px; } .auth-card :deep(form) { padding: 22px; } }
</style>
