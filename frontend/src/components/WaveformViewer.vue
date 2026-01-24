<template>
  <div class="viewer-container">
    <n-card :bordered="false">
      <template #header>
        <n-space align="center" justify="space-between">
          <n-space :size="14" wrap>
            <n-tag type="info" size="small">站点: {{ datasetStore.metadata?.station }}</n-tag>
            <n-tag type="info" size="small">设备: {{ datasetStore.metadata?.relay }}</n-tag>
            <n-tag type="info" size="small">版本: {{ datasetStore.metadata?.version }}</n-tag>
            <n-tag type="info" size="small">{{
              datasetStore.metadata?.dataFileType.toUpperCase()
            }}</n-tag>
            <n-tag type="info" size="small"
              >额定频率: {{ datasetStore.metadata?.frequency }}Hz</n-tag
            >
            <n-tooltip v-if="datasetStore.metadata?.sampleRates" placement="bottom">
              <template #trigger>
                <n-tag type="info" size="small">采样数量: {{ sampleCount }}</n-tag>
              </template>
              <div>
                <div v-for="(rate, idx) in datasetStore.metadata.sampleRates" :key="idx">
                  {{ rate.sampRate }}Hz -> {{ rate.lastSampleNum }}
                </div>
              </div>
            </n-tooltip>
            <n-tag type="default" size="small">开始: {{ startTime }}</n-tag>
          </n-space>
          <n-space>
            <n-button-group>
              <n-button
                :type="xAxesType === XAxesType.Index ? 'primary' : 'default'"
                @click="xAxesType = XAxesType.Index"
                size="small"
              >
                索引模式
              </n-button>
              <n-button
                :type="xAxesType === XAxesType.Time ? 'primary' : 'default'"
                @click="xAxesType = XAxesType.Time"
                size="small"
              >
                时间模式
              </n-button>
            </n-button-group>
            <n-button type="primary" @click="refreshData()" :loading="loading"> 刷新视图 </n-button>
          </n-space>
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
import { ref, onMounted, watch, onUnmounted, shallowRef, reactive, computed } from 'vue'
import * as echarts from 'echarts'
import { NCard, NSpace, NTag, NButton, NButtonGroup, NSpin, NAlert } from 'naive-ui'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'
import { getWaveforms, type ChannelValue } from '../api'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const chartRef = ref<HTMLElement>()
const chartInstance = shallowRef<echarts.ECharts>()
const loading = ref(false)
const sampleCount = ref(0)
let channelValues = reactive<Array<ChannelValue>>([])
let timestamps = reactive<Array<number>>([])
let xWindow = reactive<{ start: number; end: number }>({ start: 0, end: 0 })

enum XAxesType {
  Time = 'time',
  Index = 'index',
}

// Track initial data bounds for keeping X-axis consistent
let initialWindow = { start: 0, end: 0 }
const lastWindow = { startIndex: 0, endIndex: 0, startTime: 0, endTime: 0 }
const xAxesType = ref<XAxesType>(XAxesType.Index)

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    chartInstance.value.on('dataZoom', handleDataZoom)
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

// startTime
const startTime = computed(() => {
  if (datasetStore.metadata?.startTime) {
    // YYYY-MM-DD HH:mm:ss
    return new Date(datasetStore.metadata.startTime)
      .toISOString()
      .replace('T', ' ')
      .replace('Z', '')
  }
  return ''
})

// Watchers to trigger update
watch(
  [() => viewStore.selectedAnalogChannels, () => viewStore.selectedDigitalChannels],
  () => {
    refreshData()
  },
  { deep: true },
)

watch(
  () => datasetStore.currentId,
  () => {
    // Reset initial window when dataset metadata changes
    initialWindow = { start: 0, end: 0 }
    refreshData()
  },
)

watch(
  () => xAxesType.value,
  () => {
    // Reset initial window when xAxesType changes
    initialWindow = { start: 0, end: 0 }
    renderChart()
  },
)

function renderChart() {
  // Store initial window on first load
  if (initialWindow.start === 0 && initialWindow.end === 0) {
    if (xAxesType.value === XAxesType.Index)
      initialWindow = { start: 0, end: timestamps.length - 1 }
    else {
      initialWindow = { start: timestamps[0]!, end: timestamps[timestamps.length - 1]! }
    }
  }

  // Calculate zoom percentages if this is a zoomed refresh
  const span = initialWindow.end - initialWindow.start
  let zoomStartPct: number = 0
  let zoomEndPct: number = 100
  if (xAxesType.value === XAxesType.Index) {
    zoomStartPct = ((xWindow.start - initialWindow.start) / span) * 100
    zoomEndPct = ((xWindow.end - initialWindow.start) / span) * 100
  } else {
    zoomStartPct = ((timestamps[xWindow.start]! - initialWindow.start) / span) * 100
    zoomEndPct = ((timestamps[xWindow.end]! - initialWindow.start) / span) * 100
  }

  // sampleCount.value = timestamps.length
  const seriesCount = channelValues.length
  const axesIndices = Array.from({ length: seriesCount }, (_, i) => i)

  sampleCount.value = timestamps.length

  // 预留顶部/底部空间给标题/缩放器，按百分比垂直堆叠各 grid
  const plotAreaPct = 95 // 95% 高度作为绘图区
  const topMarginPct = 4
  const perGridPct = plotAreaPct / Math.max(seriesCount, 1)
  const LEFT_MARGIN_PX = 50
  const RIGHT_MARGIN_PX = 30

  const grids = channelValues.map((_, i) => ({
    left: LEFT_MARGIN_PX,
    right: RIGHT_MARGIN_PX,
    top: `${topMarginPct + i * perGridPct}%`,
    height: `${perGridPct - 4}%`,
  }))

  const xAxes = channelValues.map((_, i) => ({
    min: initialWindow.start,
    max: initialWindow.end,
    gridIndex: i,
    axisLabel: {
      show: i === seriesCount - 1,
      formatter: (value: number) => {
        if (xAxesType.value === XAxesType.Index) {
          return timestamps[value] + ' ms' || ''
        }
        return value + ' ms'
      },
    },
    axisTick: { show: false },
    axisLine: { show: false },
    splitLine: { show: true },
  }))

  const yAxes = channelValues.map((s, i) => {
    const isDigital = s.type === 'digital'
    const yAxisConfig: Record<string, unknown> = {
      scale: !isDigital, // 开关量不使用自动缩放
      gridIndex: i,
      name: s.unit ? `${s.name} (${s.unit})` : s.name,
      nameTextStyle: {
        align: 'left' as const,
        padding: [0, 0, -5, -LEFT_MARGIN_PX],
      },
      axisLabel: {
        show: true,
      },
      splitLine: { show: true },
    }

    // 为开关量配置Y轴：只显示0和1
    if (isDigital) {
      yAxisConfig.min = -0.1
      yAxisConfig.max = 1.1
      yAxisConfig.type = 'value'
      yAxisConfig.axisLabel = {
        formatter: (value: number) => {
          if (value === 0) return '0'
          if (value === 1) return '1'
          return ''
        },
        showMinLabel: true,
        showMaxLabel: true,
      }
      yAxisConfig.splitLine = { show: false }
      yAxisConfig.axisTick = {
        show: true,
        length: 3,
      }
    }

    return yAxisConfig
  })

  // 获取图表容器宽度
  const DEFAULT_CHART_WIDTH = 800
  const chartWidth = chartRef.value?.clientWidth || DEFAULT_CHART_WIDTH

  // 为每个子图添加底部边框线
  const graphicElements = channelValues.flatMap((_, i) => {
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
        // 根据channelValues中的顺序排序显示
        const sortedParams = Array.isArray(params)
          ? params.slice().sort((a, b) => {
              const seriesA = channelValues.findIndex((s) => s.name === a.seriesName)
              const seriesB = channelValues.findIndex((s) => s.name === b.seriesName)
              return seriesA - seriesB
            })
          : []
        if (sortedParams.length === 0) return ''
        // show x value at top
        let xValue = Array.isArray(sortedParams[0]?.data)
          ? sortedParams[0].data[0]
          : sortedParams[0]?.data
        if (xAxesType.value === XAxesType.Index) {
          // 显示为时间戳
          xValue = timestamps[xValue as number] || 0
        }
        return (
          `${xValue} ms<br/>` +
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
    series: channelValues.map((s, i) => {
      // 根据通道类型选择不同的渲染方式
      const isDigital = s.type === 'digital'

      return {
        name: s.name,
        type: 'line',
        step: isDigital ? 'start' : false, // 开关量使用阶梯图
        showSymbol: false,
        xAxisIndex: i,
        yAxisIndex: i,
        data: s.y.map((y, k) => {
          if (xAxesType.value === XAxesType.Index) {
            return [s.times[k], y]
          }
          return [timestamps[s.times[k]!], y]
        }),
        animation: false,
        smooth: isDigital ? false : true, // 只给模拟量平滑处理
        lineStyle: {
          type: isDigital ? 'solid' : 'solid',
        },
      }
    }),
    dataZoom: [
      {
        type: 'inside',
        xAxisIndex: axesIndices,
        start: zoomStartPct,
        end: zoomEndPct,
      },
      {
        type: 'slider',
        xAxisIndex: axesIndices,
        bottom: 0,
        start: zoomStartPct,
        end: zoomEndPct,
      },
    ],
  }

  chartInstance.value?.setOption(option, { notMerge: true })
}

async function refreshData(startTime?: number, endTime?: number) {
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
    // For initial load without time range, use default downsampling
    // For zoomed view with time range, fetch detailed data
    const data = await getWaveforms(
      datasetStore.currentId,
      viewStore.selectedAnalogChannels,
      viewStore.selectedDigitalChannels,
      startTime,
      endTime,
    )

    timestamps = data.times
    xWindow = data.window

    // 规范化数字通道数据：确保值只有0和1
    channelValues = data.series.map((s) => {
      if (s.type === 'digital') {
        // 对于数字通道，规范化值为0或1
        const normalizedY = s.y.map((val) => (val !== 0 ? 1 : 0))
        return { ...s, y: normalizedY }
      }
      return s
    })

    renderChart()
    lastWindow.startIndex = data.window.start
    lastWindow.endIndex = data.window.end
    lastWindow.startTime = timestamps[data.window.start] || timestamps[0] || 0
    lastWindow.endTime = timestamps[data.window.end] || timestamps[timestamps.length - 1] || 0
    datasetStore.error = ''
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    datasetStore.error = msg
  } finally {
    loading.value = false
  }
}

// Handle dataZoom events to load detailed data for zoomed region
function handleDataZoom(params: unknown) {
  // Guard: check if chart and data are available
  if (!chartInstance.value || timestamps.length === 0) return

  const p = params as echarts.ECElementEvent
  // Guard: validate batch data exists
  if (!p.batch || !Array.isArray(p.batch) || p.batch.length === 0) return

  const dz = p.batch[0]
  // Guard: validate dz has numeric start/end
  if (typeof dz.start !== 'number' || typeof dz.end !== 'number') return

  const startPct = dz.start
  const endPct = dz.end
  const span = initialWindow.end - initialWindow.start

  // Guard: prevent division by zero
  if (span === 0) return

  // Convert percentages to absolute values
  const absStart = (startPct / 100) * span + initialWindow.start
  const absEnd = (endPct / 100) * span + initialWindow.start

  // Determine zoom ranges in both index and time domains
  const MIN_POINTS = 500
  let startIdx: number
  let endIdx: number
  let startTimeVal: number
  let endTimeVal: number

  if (xAxesType.value === XAxesType.Time) {
    // Time mode: absStart/absEnd are time values
    startIdx = binarySearchIndex(timestamps, absStart)
    endIdx = binarySearchIndex(timestamps, absEnd)

    // Enforce minimum zoom width by index count
    if (endIdx - startIdx + 1 < MIN_POINTS) {
      // Try to extend to the right first
      endIdx = Math.min(startIdx + MIN_POINTS - 1, timestamps.length - 1)
      // If near the end, shift start left to maintain width
      startIdx = Math.max(0, endIdx - (MIN_POINTS - 1))
    }

    startTimeVal = timestamps[startIdx] || timestamps[0] || 0
    endTimeVal = timestamps[endIdx] || timestamps[timestamps.length - 1] || 0

    // Thresholds based on time range
    const lastRangeTime = Math.abs(lastWindow.endTime - lastWindow.startTime)
    if (lastRangeTime === 0) return
    const currRangeTime = Math.abs(endTimeVal - startTimeVal)
    const zoomRangePct = ((currRangeTime - lastRangeTime) / lastRangeTime) * 100
    const offsetRangePct =
      ((Math.abs(startTimeVal - lastWindow.startTime) + Math.abs(endTimeVal - lastWindow.endTime)) /
        lastRangeTime) *
      100

    const threshold = 10
    if (zoomRangePct >= threshold || offsetRangePct > threshold) {
      refreshData(startIdx, endIdx)
    }
  } else {
    // Index mode: absStart/absEnd are index values
    startIdx = Math.max(0, Math.floor(absStart))
    endIdx = Math.min(timestamps.length - 1, Math.ceil(absEnd))

    // Enforce minimum zoom width by index count
    if (endIdx - startIdx + 1 < MIN_POINTS) {
      endIdx = Math.min(startIdx + MIN_POINTS - 1, timestamps.length - 1)
      startIdx = Math.max(0, endIdx - (MIN_POINTS - 1))
    }

    const lastRangeIdx = Math.abs(lastWindow.endIndex - lastWindow.startIndex)
    if (lastRangeIdx === 0) return
    const currRangeIdx = Math.abs(endIdx - startIdx)
    const zoomRangePct = ((currRangeIdx - lastRangeIdx) / lastRangeIdx) * 100
    const offsetRangePct =
      ((Math.abs(startIdx - lastWindow.startIndex) + Math.abs(endIdx - lastWindow.endIndex)) /
        lastRangeIdx) *
      100

    const threshold = 10
    if (zoomRangePct >= threshold || offsetRangePct > threshold) {
      // Always pass time values to backend
      refreshData(startIdx, endIdx)
    }
  }
}

// Binary search to find the index for a given timestamp value
function binarySearchIndex(arr: Array<number>, target: number): number {
  let left = 0
  let right = arr.length - 1

  while (left <= right) {
    const mid = Math.floor((left + right) / 2)
    if (arr[mid]! < target) {
      left = mid + 1
    } else {
      right = mid - 1
    }
  }

  return Math.min(left, arr.length - 1)
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
