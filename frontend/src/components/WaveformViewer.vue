<template>
  <div class="viewer-container">
    <n-card :bordered="false">
      <template #header>
        <n-space align="center" justify="space-between">
          <n-space :size="14" wrap>
            <n-tag type="info" size="small">站点: {{ viewStore.station }}</n-tag>
            <n-tag type="info" size="small">设备: {{ viewStore.relay }}</n-tag>
            <n-tag type="info" size="small">版本: {{ viewStore.version }}</n-tag>
            <n-tag type="info" size="small">数据类型: {{ viewStore.dataType.toUpperCase() }}</n-tag>
            <n-tag type="info" size="small">采样点数量: {{ sampleCount }}</n-tag>
            <n-tag type="default" size="small">开始: {{ viewStore.startTime }}</n-tag>
            <n-tag type="default" size="small">结束: {{ viewStore.endTime }}</n-tag>
          </n-space>
          <n-button type="primary" @click="refreshData" :loading="loading"> 刷新视图 </n-button>
        </n-space>
      </template>
      <div ref="chartRef" class="chart"></div>
      <n-spin v-if="loading" :show="loading" class="loading-overlay">
        <template #description>加载数据中...</template>
      </n-spin>
      <n-alert
        v-if="datasetStore.error"
        type="error"
        :title="datasetStore.error"
        class="error-banner"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted, shallowRef } from 'vue'
import * as echarts from 'echarts'
import { NCard, NSpace, NTag, NButton, NSpin, NAlert } from 'naive-ui'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'
import { getWaveforms } from '../api'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const chartRef = ref<HTMLElement>()
const chartInstance = shallowRef<echarts.ECharts>()
const loading = ref(false)
const sampleCount = ref(0)

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    window.addEventListener('resize', resizeChart)

    refreshData()
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

async function refreshData() {
  if (
    !datasetStore.currentId ||
    (viewStore.selectedAnalogChannels.length === 0 &&
      viewStore.selectedDigitalChannels.length === 0)
  ) {
    chartInstance.value?.clear()
    sampleCount.value = 0
    return
  }

  loading.value = true
  try {
    const data = await getWaveforms(datasetStore.currentId, [
      ...viewStore.selectedAnalogChannels,
      ...viewStore.selectedDigitalChannels,
    ])

    // sort series by channel type and number
    // data.series.sort((a, b) => {
    //   const typeA = a.type === 'analog' ? 0 : 1
    //   const typeB = b.type === 'analog' ? 0 : 1
    //   if (typeA !== typeB) return typeA - typeB
    //   return a.channel - b.channel
    // })

    // sampleCount.value = data.times.length
    const seriesCount = data.series.length
    const axesIndices = Array.from({ length: seriesCount }, (_, i) => i)

    sampleCount.value = data.downsample.originalPoints

    // 预留顶部/底部空间给标题/缩放器，按百分比垂直堆叠各 grid
    const plotAreaPct = 95 // 95% 高度作为绘图区
    const topMarginPct = 4
    const perGridPct = plotAreaPct / Math.max(seriesCount, 1)
    const LEFT_MARGIN_PX = 50
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
        padding: [0, 0, -20, -LEFT_MARGIN_PX],
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
        data: s.y.map((y, k) => [s.times[k], y]),
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
  height: 100%;
  /* padding: 16px; */
  position: relative;
}
:deep(.n-card-header) {
  padding: 10px 24px; /* 你想要的 padding 值 */
}
.chart {
  width: 100%;
  height: calc(100vh - 125px);
  min-height: 400px;
}
.loading-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 10;
}
.error-banner {
  position: absolute;
  bottom: 16px;
  left: 16px;
  right: 16px;
  z-index: 10;
}
</style>
