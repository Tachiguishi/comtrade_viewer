<template>
  <div class="dataset-list">
    <n-spin :show="datasetStore.loading && !datasetStore.datasets.length">
      <n-list hoverable clickable>
        <n-list-item
          v-for="ds in datasetStore.datasets"
          :key="ds.datasetId"
          @click="select(ds.datasetId)"
        >
          <template #prefix>
            <n-icon size="15">
              <DocumentTextOutline />
            </n-icon>
          </template>
          <n-thing :title="ds.name">
            <template #header>
              <div style="display: flex; align-items: center; gap: 8px">
                <span>{{ ds.name }}</span>
                <n-space :size="4">
                  <n-tag size="small" type="info">{{ formatSize(ds.sizeBytes) }}</n-tag>
                  <n-tag v-if="ds.datasetId === datasetStore.currentId" size="small" type="success">
                    当前选中
                  </n-tag>
                </n-space>
              </div>
            </template>
          </n-thing>
        </n-list-item>
      </n-list>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NList, NListItem, NThing, NTag, NSpace, NSpin, NIcon } from 'naive-ui'
import { DocumentTextOutline } from '@vicons/ionicons5'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const router = useRouter()

onMounted(() => {
  datasetStore.refreshList()
})

async function select(id: string) {
  viewStore.clearChannelSelection()
  await datasetStore.loadMetadata(id)
  if (datasetStore.metadata) {
    if (viewStore.selectedAnalogChannels.length === 0) {
      const defaults = datasetStore.metadata.analogChannels.slice(0, 6).map((c) => c.id)
      viewStore.setAnalogChannels(defaults)
    }
  }
  // 切换到 viewer 页面
  router.push('/viewer')
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.dataset-list {
  height: 100%;
  overflow-y: auto;
}
</style>
