<template>
  <div class="channel-list">
    <div v-if="!datasetStore.metadata" class="no-data">
      <n-empty description="请先选择一个数据集" />
    </div>
    <n-space v-else vertical>
      <div class="channel-section">
        <n-divider title-placement="left">
          模拟通道 ({{ datasetStore.metadata.analogChannelNum }})
        </n-divider>
        <n-input
          v-model:value="analogFilter"
          placeholder="过滤通道..."
          clearable
          size="small"
          style="margin-bottom: 12px"
        />
        <n-scrollbar style="max-height: 40vh">
          <n-space vertical :size="4">
            <n-checkbox
              v-for="ch in filteredAnalogChannels"
              :key="ch.id"
              :checked="viewStore.selectedAnalogChannels.includes(ch.id)"
              @update:checked="() => viewStore.toggleAnalogChannel(ch.id)"
            >
              <n-tooltip :delay="500">
                <template #trigger>
                  <span style="display: flex">
                    <span class="channel-name analog">{{ ch.id }}. {{ ch.name }}</span>
                    <span class="channel-unit">{{ ch.unit }}</span>
                  </span>
                </template>
                <div>
                  <div>相位: {{ ch.phase || '-' }}</div>
                  <div>CCBM: {{ ch.ccbm || '-' }}</div>
                  <div>a: {{ ch.multiplier }}</div>
                  <div>b: {{ ch.offset }}</div>
                  <div>时间偏移: {{ ch.skew }}</div>
                  <div>范围: [{{ ch.minValue }}, {{ ch.maxValue }}]</div>
                  <div>一次/二次: {{ ch.primary }}/{{ ch.secondary }}</div>
                  <div>PS: {{ ch.ps }}</div>
                </div>
              </n-tooltip>
            </n-checkbox>
          </n-space>
        </n-scrollbar>
      </div>

      <div class="channel-section">
        <n-divider title-placement="left">
          数字通道 ({{ datasetStore.metadata.digitalChannelNum }})
        </n-divider>
        <n-input
          v-model:value="digitalFilter"
          placeholder="过滤通道..."
          clearable
          size="small"
          style="margin-bottom: 12px"
        />
        <n-scrollbar style="max-height: 37vh">
          <n-space vertical :size="4">
            <n-checkbox
              v-for="ch in filteredDigitalChannels"
              :key="ch.id"
              :checked="viewStore.selectedDigitalChannels.includes(ch.id)"
              @update:checked="() => viewStore.toggleDigitalChannel(ch.id)"
            >
              <n-tooltip :delay="500">
                <template #trigger>
                  <span class="channel-name digital">{{ ch.id }}. {{ ch.name }}</span>
                </template>
                <div>
                  <div>相位: {{ ch.phase || '-' }}</div>
                  <div>CCBM: {{ ch.ccbm || '-' }}</div>
                  <div>Y: {{ ch.y }}</div>
                </div>
              </n-tooltip>
            </n-checkbox>
          </n-space>
        </n-scrollbar>
      </div>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NSpace, NCheckbox, NTooltip, NDivider, NInput, NScrollbar, NEmpty } from 'naive-ui'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const analogFilter = ref('')
const digitalFilter = ref('')

const filteredAnalogChannels = computed(() => {
  const channels = datasetStore.metadata?.analogChannels || []
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
</script>

<style scoped>
.channel-list {
  height: 100%;
}
.no-data {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}
.n-divider {
  margin: 5px 0;
}
.channel-section {
  margin-bottom: 0;
}
.channel-name {
  font-size: 13px;
  margin-right: 8px;
}
.channel-name.analog {
  color: #2c3e50;
}
.channel-name.digital {
  color: #d35400;
}
.channel-unit {
  font-size: 12px;
  color: #888;
  margin-left: auto;
}
</style>
