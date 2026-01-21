<template>
  <n-config-provider :theme="null">
    <n-message-provider>
      <template v-if="showShell">
        <n-layout style="height: 100vh">
          <n-layout-header bordered class="app-header">
            <div class="app-header-content">
              <h1 class="app-title">ComTrade Viewer</h1>
              <n-menu
                mode="horizontal"
                :options="menuOptions"
                :value="activeKey"
                @update:value="handleMenuSelect"
              />
            </div>
          </n-layout-header>
          <n-layout-content style="height: calc(100vh - 64px)">
            <router-view />
          </n-layout-content>
        </n-layout>
      </template>
      <template v-else>
        <router-view />
      </template>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { h, computed } from 'vue'
import type { Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { CloudUploadOutline, BarChartOutline } from '@vicons/ionicons5'
import { useAuthStore } from './stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  {
    label: '波形列表',
    key: 'upload',
    icon: renderIcon(CloudUploadOutline),
  },
  {
    label: '波形查看',
    key: 'viewer',
    icon: renderIcon(BarChartOutline),
  },
  {
    label: '波形查看2',
    key: 'canvas',
    icon: renderIcon(BarChartOutline),
  },
]

const activeKey = computed(() => {
  const path = route.path
  if (path.startsWith('/upload')) return 'upload'
  if (path.startsWith('/viewer')) return 'viewer'
  if (path.startsWith('/canvas')) return 'canvas'
  return 'upload'
})

const showShell = computed(() => route.name !== 'login' && authStore.isAuthenticated)

function handleMenuSelect(key: string) {
  router.push(`/${key}`)
}
</script>

<style scoped>
.app-header {
  height: 48px;
  margin-top: 5px;
  padding: 0 24px;
  display: flex;
  align-items: center;
}

.app-header-content {
  display: flex;
  align-items: center;
  width: 100%;
  justify-content: space-between;
}

.app-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  white-space: nowrap;
  padding: 0 0 8px 0;
}
</style>
