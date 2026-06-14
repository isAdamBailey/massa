<script setup lang="ts">
const route = useRoute()
const auth = useAuthStore()

const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/settings', label: 'Settings' }
]

async function onLogout() {
  await auth.logout()
  await navigateTo('/login')
}
</script>

<template>
  <header class="flex flex-wrap items-center justify-between gap-3 rounded-md bg-slate px-4 py-3 sm:px-5">
    <NuxtLink
      to="/"
      class="flex items-center gap-2.5 text-wordmark font-mono tabular-nums tracking-tight text-mist"
    >
      <span class="relative flex h-2 w-2 shrink-0">
        <span class="absolute inset-0 rounded-full bg-verdigris animate-glow-pulse" />
        <span class="relative h-2 w-2 rounded-full bg-verdigris" />
      </span>
      Massa
    </NuxtLink>

    <nav
      aria-label="Primary"
      class="order-3 flex w-full gap-1 rounded-sm bg-graphite p-1 text-label sm:order-none sm:w-auto"
    >
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex-1 rounded-sm px-3 py-1.5 text-center transition-colors duration-150 sm:flex-none"
        :class="route.path === item.to ? 'bg-verdigris text-carbon' : 'text-mist hover:bg-graphite-hover'"
      >
        {{ item.label }}
      </NuxtLink>
    </nav>

    <button
      type="button"
      class="rounded-sm bg-graphite px-4 py-2 text-label text-mist transition-colors duration-150 hover:bg-graphite-hover"
      @click="onLogout"
    >
      Log out
    </button>
  </header>
</template>
