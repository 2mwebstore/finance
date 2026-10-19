<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h1 class="text-xl font-bold text-navy">Permission codes</h1>
      <div class="flex items-center gap-2">
        <span class="text-xs text-muted">{{ catalog.length }} codes</span>
        <button
          class="rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-medium hover:bg-canvas"
          :disabled="syncing"
          @click="onRefresh"
        >
          {{ syncing ? 'Syncing…' : 'Sync from API' }}
        </button>
      </div>
    </div>

    <div class="text-xs text-muted flex flex-wrap gap-3">
      <span v-if="catalogSynced" class="text-income">● Synced from API</span>
      <span v-else class="text-muted">○ Using static catalog</span>
      <span v-if="catalogSyncError" class="text-expense">{{ catalogSyncError }}</span>
      <span v-if="store.lastSyncedAt">Last: {{ formatTime(store.lastSyncedAt) }}</span>
    </div>

    <p class="text-sm text-muted">
      Codes are enforced by the API. This list is loaded from
      <code class="text-xs bg-canvas px-1 rounded">GET /api/v1/permissions</code>
      when your role allows it.
    </p>

    <div class="flex gap-2 overflow-x-auto pb-1">
      <button
        v-for="m in moduleOptions"
        :key="m"
        class="shrink-0 rounded-full px-3 py-1.5 text-xs font-medium border"
        :class="moduleFilter === m ? 'bg-primary text-white border-primary' : 'bg-white text-muted border-border'"
        @click="moduleFilter = m"
      >
        {{ m || 'All' }}
      </button>
    </div>

    <div class="bg-white rounded-xl border border-border overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-canvas text-left text-xs text-muted">
          <tr>
            <th class="px-3 py-2 font-medium">Code</th>
            <th class="px-3 py-2 font-medium hidden sm:table-cell">Name</th>
            <th class="px-3 py-2 font-medium">Module</th>
            <th class="px-3 py-2 font-medium hidden md:table-cell">You</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="row in filtered" :key="row.code">
            <td class="px-3 py-2 font-mono text-xs text-primary">{{ row.code }}</td>
            <td class="px-3 py-2 hidden sm:table-cell">{{ row.name }}</td>
            <td class="px-3 py-2 text-xs text-muted">{{ row.module }}</td>
            <td class="px-3 py-2 hidden md:table-cell">
              <span v-if="can(row.code)" class="text-[10px] text-income font-semibold">YES</span>
              <span v-else class="text-[10px] text-muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
const store = usePermissionsStore()
const { catalog, catalogSynced, catalogSyncError, refreshCatalog, can } = usePermissions()

const moduleFilter = ref('')
const syncing = ref(false)

const moduleOptions = computed(() => {
  const set = new Set(catalog.value.map((c) => c.module))
  return ['', ...Array.from(set).sort()]
})

const filtered = computed(() =>
  moduleFilter.value
    ? catalog.value.filter((c) => c.module === moduleFilter.value)
    : catalog.value,
)

function formatTime(d: string) {
  try {
    return new Date(d).toLocaleString()
  } catch {
    return d
  }
}

async function onRefresh() {
  syncing.value = true
  try {
    await refreshCatalog()
  } finally {
    syncing.value = false
  }
}

onMounted(() => {
  if (!catalogSynced.value) onRefresh()
})
</script>
