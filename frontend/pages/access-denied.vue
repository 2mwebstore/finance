<template>
  <div class="text-center py-20 px-4">
    <div class="text-4xl mb-3">🔒</div>
    <h1 class="text-xl font-bold text-navy">Access denied</h1>
    <p class="text-muted text-sm mt-2 max-w-sm mx-auto">
      Your role does not include permission to view this page. Contact an administrator if you need access.
    </p>
    <div class="mt-6 flex flex-wrap justify-center gap-3">
      <NuxtLink to="/dashboard" class="rounded-lg bg-primary text-white text-sm font-medium px-4 py-2">
        Dashboard
      </NuxtLink>
      <NuxtLink to="/more" class="rounded-lg border border-border bg-white text-sm font-medium px-4 py-2">
        My permissions
      </NuxtLink>
    </div>
  </div>
</template>
