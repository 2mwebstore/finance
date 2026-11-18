<template>
  <div class="space-y-4">
    <!-- Finds an agent by user or name, and puts each currency's agents A to Z or Z to A -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="min-w-0 flex-1 sm:min-w-[14rem] sm:max-w-sm"><ListSearch v-model="search" :delay="150" placeholder="Search agent…" /></div>
      <div class="flex items-center gap-0.5 rounded-lg bg-canvas p-1" role="radiogroup" aria-label="Sort agents">
        <span class="px-1.5 text-[11px] font-semibold text-muted">Sort</span>
        <button v-for="o in AGENT_SORTS" :key="o.value" type="button" role="radio" :aria-checked="sort === o.value" class="rounded-md px-2.5 py-1 text-xs font-semibold" :class="sort === o.value ? 'bg-white text-primary shadow-sm' : 'text-muted hover:text-navy'" @click="sort = o.value">{{ o.label }}</button>
      </div>
      <!-- A page's own filters (the Agents list: Branch, turned-off agents) -->
      <slot />
    </div>
    <!-- Only the agents in one currency, or whose code starts with SBK_, SBU_, 12K_ … (All: every one) -->
    <div v-if="currencies.length || prefixes.length" class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      <div v-if="currencies.length" class="flex flex-wrap items-center gap-1.5" role="radiogroup" aria-label="Currency">
        <button v-for="c in [{ value: '', label: 'All', count: total }, ...currencies]" :key="c.value || 'all'" type="button" role="radio" :aria-checked="currency === c.value" class="inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-semibold" :class="currency === c.value ? 'border-primary bg-blue-50 text-primary' : 'border-border bg-white text-muted hover:text-navy'" @click="currency = c.value">
          <span>{{ c.label }}</span><span class="font-normal opacity-70">{{ c.count }}</span>
        </button>
      </div>
      <span v-if="currencies.length && prefixes.length" class="h-5 w-px bg-border" aria-hidden="true" />
      <div v-if="prefixes.length" class="flex flex-wrap items-center gap-1.5" role="radiogroup" aria-label="Agents">
        <button v-for="p in [{ value: '', label: 'All', count: total }, ...prefixes]" :key="p.value || 'all'" type="button" role="radio" :aria-checked="prefix === p.value" class="inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-semibold" :class="prefix === p.value ? 'border-primary bg-blue-50 text-primary' : 'border-border bg-white text-muted hover:text-navy'" @click="prefix = p.value">
          <span :class="p.value ? 'font-mono' : ''">{{ p.label }}</span><span class="font-normal opacity-70">{{ p.count }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AGENT_SORTS, type AgentChoice, type AgentSort } from '~/composables/useAgentFilter'

/** Data entry's agent filters (New entry, Update and an entry's View page); see useAgentFilter. */
defineProps<{ currencies: AgentChoice[]; prefixes: AgentChoice[]; total: number }>()
const search = defineModel<string>('search', { required: true })
const sort = defineModel<AgentSort>('sort', { required: true })
const currency = defineModel<string>('currency', { required: true })
const prefix = defineModel<string>('prefix', { required: true })
</script>
