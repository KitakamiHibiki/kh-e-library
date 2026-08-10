<script setup lang="ts">
import { RouterView } from 'vue-router'
import { useTheme } from '@/composables/useTheme'

const { loadTheme } = useTheme()
loadTheme()
</script>

<template>
  <!-- The Reader is kept alive so navigating to the completion page doesn't
       unmount it: returning to /read restores the parsed book + exact scroll
       position instantly instead of re-processing the file. Cached per full
       path (book id), evicted LRU by :max. -->
  <RouterView v-slot="{ Component }">
    <KeepAlive include="Reader" :max="3">
      <component :is="Component" :key="$route.fullPath" />
    </KeepAlive>
  </RouterView>
</template>

<style lang="less">
@import './styles/variables.less';
:root {
  --bg-primary: #ffffff;
  --bg-secondary: #f5f5f5;
  --text-primary: #303133;
  --text-secondary: #909399;
  --border-color: #e0e0e0;
}

html.dark {
  --bg-primary: #1a1a2e;
  --bg-secondary: #16213e;
  --text-primary: #e0e0e0;
  --text-secondary: #a0a0a0;
  --border-color: #333;
}

html.dark body {
  background: var(--bg-primary);
  color: var(--text-primary);
}

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
}
</style>
