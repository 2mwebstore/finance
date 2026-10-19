<template>
  <div class="auth-shell">
    <div class="auth-glow auth-glow-one"></div>
    <div class="auth-glow auth-glow-two"></div>
    <div class="auth-layout">
      <div class="auth-intro">
        <div class="auth-brand"><span class="auth-brand-mark"><ChartNoAxesCombined :size="26" /></span><span>Income &amp; Expense</span></div>
        <p class="auth-eyebrow">BUSINESS FINANCE</p>
        <h1>Make every business decision with clarity.</h1>
        <p class="auth-copy">Track income, expenses, branches, and reports in one simple workspace.</p>
        <div class="auth-points"><span><Check :size="15" /> Real-time overview</span><span><Check :size="15" /> Secure access control</span></div>
      </div>
      <div class="auth-card"><slot /></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChartNoAxesCombined, Check } from 'lucide-vue-next'
</script>

<style scoped>
.auth-shell { position: relative; display: flex; min-height: 100vh; overflow: hidden; align-items: center; justify-content: center; padding: 28px; color: #fff; background: linear-gradient(135deg, #0a234b 0%, #102f5d 52%, #1769e8 160%); }
.auth-layout { position: relative; z-index: 1; display: grid; width: min(920px, 100%); grid-template-columns: 1fr 380px; align-items: center; gap: 80px; }
.auth-intro { max-width: 430px; }
.auth-brand { display: flex; align-items: center; gap: 10px; font-size: 15px; font-weight: 750; }
.auth-brand-mark { display: grid; width: 46px; height: 46px; place-items: center; border-radius: 13px; background: linear-gradient(140deg, #3388ff, #1461e7); box-shadow: 0 8px 18px rgb(0 0 0 / 20%); }
.auth-eyebrow { margin: 58px 0 10px; color: #8ab6fa; font-size: 10px; font-weight: 800; letter-spacing: .16em; }
.auth-intro h1 { margin: 0; font-size: clamp(31px, 4vw, 48px); line-height: 1.08; letter-spacing: -.045em; }
.auth-copy { max-width: 360px; margin: 18px 0 0; color: #b8c9e2; font-size: 14px; line-height: 1.65; }
.auth-points { display: flex; flex-wrap: wrap; gap: 14px; margin-top: 30px; color: #d5e2f5; font-size: 11px; }
.auth-points span { display: inline-flex; align-items: center; gap: 6px; }
.auth-points svg { color: #58d0aa; }
.auth-card :deep(form) { padding: 30px; border: 1px solid #e4ebf5; border-radius: 18px; box-shadow: 0 20px 55px rgb(3 21 52 / 25%); }
.auth-card :deep(form + *) { color: #fff; }
.auth-glow { position: absolute; width: 420px; height: 420px; border-radius: 50%; background: rgb(47 132 255 / 18%); filter: blur(3px); }
.auth-glow-one { top: -200px; right: -100px; }
.auth-glow-two { bottom: -250px; left: -160px; background: rgb(31 193 145 / 10%); }
@media (max-width: 720px) { .auth-shell { padding: 18px; align-items: flex-start; } .auth-layout { display: block; width: 100%; padding-top: 8vh; } .auth-intro { margin-bottom: 26px; text-align: center; } .auth-brand { justify-content: center; } .auth-eyebrow { margin-top: 30px; } .auth-intro h1 { font-size: 30px; } .auth-copy { margin-inline: auto; font-size: 13px; } .auth-points { justify-content: center; margin-top: 18px; } .auth-card :deep(form) { padding: 22px; } }
</style>
