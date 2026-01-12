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
      <div class="channel-header">
        <div>Analog Channels ({{ datasetStore.metadata.analogChannelNum }})</div>
        <input v-model="analogFilter" type="text" placeholder="Filter..." class="filter-input" />
      </div>
      <div class="channel-list">
        <label
          v-for="ch in filteredAnalogChannels"
          :key="ch.id"
          class="channel-item"
          :title="analogTooltip(ch)"
        >
          <input
            type="checkbox"
            :checked="viewStore.selectedAnalogChannels.includes('A' + ch.id.toString())"
            @change="() => viewStore.toggleAnalogChannel(ch.id)"
          />
          <span class="analog">{{ ch.id + '.' + ch.name }}</span>
          <span class="unit"> {{ ch.unit }} </span>
        </label>
      </div>
      <div class="channel-header">
        <div>Digital Channels ({{ datasetStore.metadata.digitalChannelNum }})</div>
        <input v-model="digitalFilter" type="text" placeholder="Filter..." class="filter-input" />
      </div>
      <div class="channel-list">
        <label
          v-for="ch in filteredDigitalChannels"
          :key="ch.id"
          class="channel-item"
          :title="digitalTooltip(ch)"
        >
          <input
            type="checkbox"
            :checked="viewStore.selectedDigitalChannels.includes('D' + ch.id.toString())"
            @change="() => viewStore.toggleDigitalChannel(ch.id)"
          />
          <span class="digital">{{ ch.id + '.' + ch.name }}</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'
import type { AnalogChannelMeta, DigitalChannelMeta } from '../api'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const analogFilter = ref('')
const digitalFilter = ref('')

onMounted(() => {
  datasetStore.refreshList()
})

const filteredAnalogChannels = computed(() => {
  const channels = datasetStore.metadata?.analogChannels || []
  console.log('Filtering analog channels with filter:', analogFilter.value)
  if (!analogFilter.value) return channels
  return channels.filter(
    (ch) =>
      ch.name.toLowerCase().includes(analogFilter.value.toLowerCase()) ||
      ch.id.toString().toLowerCase().includes(analogFilter.value.toLowerCase()),
  )
})

const filteredDigitalChannels = computed(() => {
  const channels = datasetStore.metadata?.digitalChannels || []
  if (!digitalFilter.value) return channels
  return channels.filter(
    (ch) =>
      ch.name.toLowerCase().includes(digitalFilter.value.toLowerCase()) ||
      ch.id.toString().toLowerCase().includes(digitalFilter.value.toLowerCase()),
  )
})

async function select(id: string) {
  viewStore.clearChannelSelection()
  await datasetStore.loadMetadata(id)
  if (datasetStore.metadata) {
    viewStore.setMetaData(
      datasetStore.metadata.station,
      datasetStore.metadata.relay,
      datasetStore.metadata.version,
    )
    viewStore.setTimeRange(datasetStore.metadata.startTime, datasetStore.metadata.endTime)

    if (viewStore.selectedAnalogChannels.length === 0) {
      const defaults = datasetStore.metadata.analogChannels.slice(0, 3).map((c) => c.id)
      viewStore.setAnalogChannels(defaults)
    }
  }
}
function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const analogTooltip = (ch: AnalogChannelMeta) =>
  `Phase: ${ch.phase || '-'}\nCCBM: ${ch.ccbm || '-'}\nMultiplier: ${ch.multiplier}\nOffset: ${ch.offset}\nSkew: ${ch.skew}\nRange: [${ch.minValue}, ${ch.maxValue}]\nPrimary/Secondary: ${ch.primary}/${ch.secondary}\nPorS: ${ch.ps}`

const digitalTooltip = (ch: DigitalChannelMeta) =>
  `Phase: ${ch.phase || '-'}\nCCBM: ${ch.ccbm || '-'}\nY: ${ch.y}`
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
.channel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 0;
  font-weight: 500;
  font-size: 13px;
  border-bottom: 1px solid #eee;
}
.filter-input {
  flex: 1;
  padding: 4px 6px;
  font-size: 12px;
  border: 1px solid #ddd;
  border-radius: 3px;
  outline: none;
}
.filter-input:focus {
  border-color: #1976d2;
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
