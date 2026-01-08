<template>
  <div class="dataset-list">
    <h3>Datasets</h3>
    <div v-if="datasetStore.loading && !datasetStore.datasets.length">Loading...</div>
    <ul v-else>
      <li
        v-for="ds in datasetStore.datasets"
        :key="ds.datasetId"
        :class="{ active: ds.datasetId === datasetStore.currentId }"
        @click="select(ds.datasetId)"
      >
        <div class="name">{{ ds.name }}</div>
        <div class="meta">{{ formatSize(ds.sizeBytes) }}</div>
      </li>
    </ul>

    <div v-if="datasetStore.metadata" class="channels">
      <h4>Channels</h4>
      <div class="channel-list">
        <label v-for="ch in datasetStore.metadata.channels" :key="ch.id" class="channel-item">
          <input
            type="checkbox"
            :checked="viewStore.selectedChannels.includes(ch.id)"
            @change="() => viewStore.toggleChannel(ch.id)"
          />
          <span :class="ch.type">{{ ch.name }}</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()

onMounted(() => {
  datasetStore.refreshList()
})

async function select(id: string) {
  await datasetStore.loadMetadata(id)
  // Auto-select first few channels if none selected
  if (datasetStore.metadata && viewStore.selectedChannels.length === 0) {
    const defaults = datasetStore.metadata.channels.slice(0, 3).map((c) => c.id)
    viewStore.setChannels(defaults)
  }
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.dataset-list ul {
  list-style: none;
  padding: 0;
  margin: 0;
}
li {
  padding: 6px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 2px;
}
li:hover {
  background: #f5f5f5;
}
li.active {
  background: #e3f2fd;
  color: #1976d2;
}
.name {
  font-weight: 500;
  word-break: break-all;
}
.meta {
  font-size: 11px;
  color: #888;
}
.channels {
  margin-top: 15px;
  border-top: 1px solid #eee;
  pt: 10px;
}
.channel-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.channel-item {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}
.channel-item .analog {
  color: #2c3e50;
}
.channel-item .digital {
  color: #d35400;
}
</style>
