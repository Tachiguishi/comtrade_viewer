<template>
  <div class="dataset-list">
    <h4>Datasets</h4>
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
      <h5>Analog Channels</h5>
      <div class="channel-list">
        <label v-for="ch in datasetStore.metadata.analogChannels" :key="ch.id" class="channel-item">
          <input
            type="checkbox"
            :checked="viewStore.selectedAnalogChannels.includes(ch.id)"
            @change="() => viewStore.toggleAnalogChannel(ch.id)"
          />
          <span class="analog">{{ ch.id + '.' + ch.name }}</span>
          <span class="unit"> {{ ch.unit }} </span>
        </label>
      </div>
      <h5>Digital Channels</h5>
      <div class="channel-list">
        <label
          v-for="ch in datasetStore.metadata.digitalChannels"
          :key="ch.id"
          class="channel-item"
        >
          <input
            type="checkbox"
            :checked="viewStore.selectedDigitalChannels.includes(ch.id)"
            @change="() => viewStore.toggleDigitalChannel(ch.id)"
          />
          <span class="digital">{{ ch.id + '.' + ch.name }}</span>
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
  if (datasetStore.metadata && viewStore.selectedAnalogChannels.length === 0) {
    const defaults = datasetStore.metadata.analogChannels.slice(0, 3).map((c) => c.id)
    viewStore.setAnalogChannels(defaults)
  }
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.dataset-list {
  display: flex;
  flex-direction: column;
}
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
  margin-top: 0px;
  border-top: 1px solid #eee;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.channel-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  flex: 1 1 auto;
  min-height: 0;
  max-height: 45%;
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
.channel-item .unit {
  margin-left: auto;
  margin-right: 5px;
}
.channel-item .digital {
  color: #d35400;
}
</style>
