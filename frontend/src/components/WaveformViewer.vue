<template>
  <div class="viewer-container">
    <div class="toolbar">
      <span>station: {{ viewStore.station }}</span>
      <span>relay: {{ viewStore.relay }}</span>
      <span>version: {{ viewStore.version }}</span>
      <span>start: {{ viewStore.startTime }}</span>
      <span>end: {{ viewStore.endTime }}</span>
      <button @click="refreshData">Refresh View</button>
    </div>
    <div ref="chartRef" class="chart"></div>
    <div v-if="loading" class="loading-overlay">Loading Data...</div>
    <div v-if="datasetStore.error" class="error-banner">{{ datasetStore.error }}</div>
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
    const data = await getWaveforms(datasetStore.currentId, [
      ...viewStore.selectedAnalogChannels,
      ...viewStore.selectedDigitalChannels,
    ])

    // sort series by channel type and number
    data.series.sort((a, b) => {
      const typeA = a.type === 'analog' ? 0 : 1
      const typeB = b.type === 'analog' ? 0 : 1
      if (typeA !== typeB) return typeA - typeB
      return a.channel - b.channel
    })

    const seriesCount = data.series.length
    const axesIndices = Array.from({ length: seriesCount }, (_, i) => i)

    // 预留顶部/底部空间给标题/缩放器，按百分比垂直堆叠各 grid
    const plotAreaPct = 95 // 95% 高度作为绘图区
    const topMarginPct = 4
    const perGridPct = plotAreaPct / Math.max(seriesCount, 1)
    const LEFT_MARGIN_PX = 60
    const RIGHT_MARGIN_PX = 30

    const grids = data.series.map((_, i) => ({
      left: LEFT_MARGIN_PX,
      right: RIGHT_MARGIN_PX,
      top: `${topMarginPct + i * perGridPct}%`,
      height: `${perGridPct - 4}%`,
    }))

    const xAxes = data.series.map((_, i) => ({
      min: data.window.start,
      max: data.window.end,
      gridIndex: i,
      axisLabel: {
        show: i === seriesCount - 1,
      },
      axisTick: { show: false },
      axisLine: { show: false },
      splitLine: { show: true },
    }))

    const yAxes = data.series.map((s, i) => ({
      scale: true,
      gridIndex: i,
      name: s.unit ? `${s.name} (${s.unit})` : s.name,
      nameTextStyle: {
        align: 'left' as const,
        padding: [0, 0, 0, -LEFT_MARGIN_PX],
      },
      axisLabel: {
        show: true,
      },
      splitLine: { show: true },
    }))

    // 获取图表容器宽度
    const DEFAULT_CHART_WIDTH = 800
    const chartWidth = chartRef.value?.clientWidth || DEFAULT_CHART_WIDTH

    // 为每个子图添加底部边框线
    const graphicElements = data.series.flatMap((_, i) => {
      const bottomY = `${topMarginPct + (i + 1) * perGridPct - 4}%`

      return [
        {
          type: 'line',
          right: RIGHT_MARGIN_PX,
          top: bottomY,
          shape: {
            x1: 0,
            y1: 0,
            x2: chartWidth - RIGHT_MARGIN_PX,
            y2: 0,
          },
          style: {
            stroke: '#aaa',
            lineWidth: 1,
          },
          z: 0,
        },
      ]
    })

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
          link: [{ xAxisIndex: 'all' }],
          crossStyle: {
            color: '#999',
            width: 1,
            type: 'dashed',
          },
          label: {
            show: false,
          },
        },
        formatter: function (params) {
          // 根据data.series中的顺序排序显示
          const sortedParams = Array.isArray(params)
            ? params.slice().sort((a, b) => {
                const seriesA = data.series.findIndex((s) => s.name === a.seriesName)
                const seriesB = data.series.findIndex((s) => s.name === b.seriesName)
                return seriesA - seriesB
              })
            : []
          if (sortedParams.length === 0) return ''
          // show x value at top
          const xValue = Array.isArray(sortedParams[0]?.data)
            ? sortedParams[0].data[0]
            : sortedParams[0]?.data
          return (
            `${xValue}<br/>` +
            sortedParams
              .map((item) => {
                const yValue = Array.isArray(item.data) ? item.data[1] : item.data
                return `<span style="display:inline-block;margin-right:6px;width:8px;height:8px;border-radius:50%;background:${item.color};"></span>${item.seriesName}: ${yValue}`
              })
              .join('<br/>')
          )
        },
      },
      axisPointer: {
        link: [{ xAxisIndex: 'all' }],
      },
      grid: grids,
      xAxis: xAxes,
      yAxis: yAxes,
      graphic: graphicElements,
      series: data.series.map((s, i) => ({
        name: s.name,
        type: 'line',
        showSymbol: false,
        xAxisIndex: i,
        yAxisIndex: i,
        data: s.y.map((y, k) => [data.times[k], y]),
        animation: false,
      })),
      dataZoom: [
        {
          type: 'inside',
          xAxisIndex: axesIndices,
        },
        {
          type: 'slider',
          xAxisIndex: axesIndices,
          bottom: 0,
        },
      ],
    }

    chartInstance.value?.setOption(option, { notMerge: true })

    // Optional: sync zoom back to store
    // chartInstance.value?.on('dataZoom', ...)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    datasetStore.error = msg
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
.error-banner {
  position: absolute;
  bottom: 8px;
  left: 8px;
  right: 8px;
  background: #ffecec;
  color: #b00020;
  border: 1px solid #ffc4c4;
  padding: 6px 8px;
  border-radius: 4px;
}
</style>
