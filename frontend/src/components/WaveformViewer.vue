<template>
  <div class="viewer-container">
    <div class="toolbar">
      <span>Window: {{ viewStore.startMs.toFixed(1) }}ms - {{ viewStore.endMs.toFixed(1) }}ms</span>
      <button @click="refreshData">Refresh View</button>
    </div>
    <div ref="chartRef" class="chart"></div>
    <div v-if="loading" class="loading-overlay">Loading Data...</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted, shallowRef } from 'vue'
import * as echarts from 'echarts'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'
import { getWaveforms } from '../api'

const props = defineProps({})
const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const chartRef = ref<HTMLElement>()
const chartInstance = shallowRef<echarts.ECharts>()
const loading = ref(false)

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    window.addEventListener('resize', resizeChart)
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeChart)
  chartInstance.value?.dispose()
})

function resizeChart() {
  chartInstance.value?.resize()
}

// Watchers to trigger update
watch(
  [
    () => datasetStore.currentId,
    () => viewStore.selectedAnalogChannels,
    () => viewStore.selectedDigitalChannels,
  ],
  () => {
    refreshData()
  },
  { deep: true },
)

// Also watch window (debouncing recommended in real app, simplistic here)
// For now, update only on explicit refresh or selection change to avoid loop
// Or we implement chart zoom event handling to update store

async function refreshData() {
  if (
    !datasetStore.currentId ||
    (viewStore.selectedAnalogChannels.length === 0 &&
      viewStore.selectedDigitalChannels.length === 0)
  ) {
    chartInstance.value?.clear()
    return
  }

  loading.value = true
  try {
    const data = await getWaveforms(
      datasetStore.currentId,
      [...viewStore.selectedAnalogChannels, ...viewStore.selectedDigitalChannels],
      viewStore.startMs,
      viewStore.endMs,
    )

    const option: echarts.EChartsOption = {
      tooltip: { trigger: 'axis' },
      legend: { data: data.series.map((s) => s.channelId), bottom: 0 },
      grid: { left: 50, right: 30, top: 20, bottom: 60, containLabel: true },
      xAxis: {
        type: 'value',
        min: data.window.start,
        max: data.window.end,
        name: 'Time (s)',
      },
      yAxis: {
        type: 'value',
        scale: true,
      },
      series: data.series.map((s) => ({
        name: s.channelId,
        type: 'line',
        showSymbol: false,
        data: s.t.map((t, i) => [t, s.y[i]]),
        animation: false, // Performance
      })),
      dataZoom: [
        {
          type: 'inside',
          xAxisIndex: 0,
        },
        {
          type: 'slider',
          xAxisIndex: 0,
        },
      ],
    }

    chartInstance.value?.setOption(option, { notMerge: true })

    // Optional: sync zoom back to store
    // chartInstance.value?.on('dataZoom', ...)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.viewer-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
}
.toolbar {
  padding: 8px;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  gap: 10px;
  align-items: center;
}
.chart {
  flex: 1;
  min-height: 0;
  width: 100%;
}
.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}
</style>
