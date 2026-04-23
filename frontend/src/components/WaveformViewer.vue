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
const cursorSampleIndex = ref<number | null>(null)
const CURSOR_LOG_PREFIX = '[WaveformCursor]'

enum XAxesType {
  Time = 'time',
  Index = 'index',
}

// Track initial data bounds for keeping X-axis consistent
let initialWindow = { start: 0, end: 0 }
const lastWindow = { startIndex: 0, endIndex: 0, startTime: 0, endTime: 0 }
const xAxesType = ref<XAxesType>(XAxesType.Index)
const LEFT_MARGIN_PX = 50
const RIGHT_MARGIN_PX = 30
const TOP_MARGIN_PCT = 4
const PLOT_AREA_PCT = 95

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    chartInstance.value.on('dataZoom', handleDataZoom)
    chartInstance.value.getZr().on('click', handleChartClick)
    console.log(`${CURSOR_LOG_PREFIX} zr click listener attached`)
    window.addEventListener('resize', resizeChart)

    refreshData()
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeChart)
  chartInstance.value?.getZr().off('click', handleChartClick)
  chartInstance.value?.dispose()
})

function resizeChart() {
  chartInstance.value?.resize()
}

function findNearestIndex(arr: number[], target: number): number {
  if (arr.length === 0) return -1
  let left = 0
  let right = arr.length - 1

  while (left <= right) {
    const mid = Math.floor((left + right) / 2)
    if (arr[mid] === target) return mid
    if (arr[mid]! < target) left = mid + 1
    else right = mid - 1
  }

  if (left <= 0) return 0
  if (left >= arr.length) return arr.length - 1
  return Math.abs(arr[left]! - target) < Math.abs(arr[left - 1]! - target) ? left : left - 1
}

function getSamplesPerCycle(sampleIndex: number): number {
  const frequency = datasetStore.metadata?.frequency || 0
  const sampleRates = datasetStore.metadata?.sampleRates || []
  if (frequency <= 0 || sampleRates.length === 0) return 0

  let selectedRate = sampleRates[sampleRates.length - 1]?.sampRate || 0
  for (const rate of sampleRates) {
    if (sampleIndex <= rate.lastSampleNum) {
      selectedRate = rate.sampRate
      break
    }
  }

  if (selectedRate <= 0) return 0
  return Math.max(1, Math.round(selectedRate / frequency))
}

function computeCycleRms(channel: ChannelValue, sampleIndex: number): string {
  if (channel.type !== 'analog' || channel.y.length === 0 || channel.times.length === 0) return '-'

  const samplesPerCycle = getSamplesPerCycle(sampleIndex)
  if (samplesPerCycle <= 0) return '-'

  const centerPos = findNearestIndex(channel.times, sampleIndex)
  if (centerPos < 0) return '-'

  const halfCycle = Math.floor(samplesPerCycle / 2)
  const rawStart = Math.max(0, centerPos - halfCycle)
  const end = Math.min(channel.y.length - 1, rawStart + samplesPerCycle - 1)
  const start = Math.max(0, end - samplesPerCycle + 1)

  let sumSquares = 0
  let count = 0
  for (let i = start; i <= end; i++) {
    const value = channel.y[i]
    if (typeof value !== 'number' || Number.isNaN(value)) continue
    sumSquares += value * value
    count++
  }

  if (count === 0) return '-'
  const rms = Math.sqrt(sumSquares / count)
  return Number.isFinite(rms) ? rms.toFixed(3) : '-'
}

function formatRelativeMs(delta: number): string {
  const normalized = Math.abs(delta) < 1e-9 ? 0 : delta
  const sign = normalized > 0 ? '+' : ''
  return `${sign}${normalized.toFixed(3)}`
}

function getCursorAxisValue(): number | null {
  if (cursorSampleIndex.value === null || timestamps.length === 0) {
    console.log(`${CURSOR_LOG_PREFIX} getCursorAxisValue skipped`, {
      cursorSampleIndex: cursorSampleIndex.value,
      timestampsLength: timestamps.length,
    })
    return null
  }
  const idx = Math.max(0, Math.min(cursorSampleIndex.value, timestamps.length - 1))
  const axisValue = xAxesType.value === XAxesType.Index ? idx : (timestamps[idx] ?? null)
  console.log(`${CURSOR_LOG_PREFIX} getCursorAxisValue`, {
    xAxesType: xAxesType.value,
    cursorSampleIndex: cursorSampleIndex.value,
    normalizedIndex: idx,
    axisValue,
  })
  return axisValue
}

function getPointFromClickEvent(evt: unknown): [number, number] | null {
  const e = evt as {
    offsetX?: number
    offsetY?: number
    zrX?: number
    zrY?: number
    event?: { offsetX?: number; offsetY?: number; zrX?: number; zrY?: number }
  }

  const x = e.offsetX ?? e.zrX ?? e.event?.offsetX ?? e.event?.zrX
  const y = e.offsetY ?? e.zrY ?? e.event?.offsetY ?? e.event?.zrY

  console.log(`${CURSOR_LOG_PREFIX} raw click event`, evt)
  console.log(`${CURSOR_LOG_PREFIX} extracted click point`, { x, y })

  if (typeof x !== 'number' || typeof y !== 'number') {
    console.log(`${CURSOR_LOG_PREFIX} invalid click point type`)
    return null
  }
  if (!Number.isFinite(x) || !Number.isFinite(y)) {
    console.log(`${CURSOR_LOG_PREFIX} click point is not finite`)
    return null
  }
  return [x, y]
}

function getCurrentAxisWindow(): { start: number; end: number } | null {
  if (timestamps.length === 0) return null

  if (xAxesType.value === XAxesType.Index) {
    const start = Number.isFinite(xWindow.start) ? xWindow.start : initialWindow.start
    const end = Number.isFinite(xWindow.end) ? xWindow.end : initialWindow.end
    return { start, end }
  }

  const startIdx = Math.max(0, Math.min(xWindow.start, timestamps.length - 1))
  const endIdx = Math.max(0, Math.min(xWindow.end, timestamps.length - 1))
  return {
    start: timestamps[startIdx] ?? timestamps[0] ?? 0,
    end: timestamps[endIdx] ?? timestamps[timestamps.length - 1] ?? 0,
  }
}

function handleChartClick(evt: unknown) {
  console.log(`${CURSOR_LOG_PREFIX} handleChartClick triggered`, {
    hasChart: !!chartInstance.value,
    timestampsLength: timestamps.length,
    channelCount: channelValues.length,
  })
  if (!chartInstance.value || timestamps.length === 0 || channelValues.length === 0) {
    console.log(`${CURSOR_LOG_PREFIX} handleChartClick early return by guard`)
    return
  }

  const point = getPointFromClickEvent(evt)
  if (!point) {
    console.log(`${CURSOR_LOG_PREFIX} handleChartClick failed to parse point`)
    return
  }

  const chartWidth = chartRef.value?.clientWidth || 0
  const chartHeight = chartRef.value?.clientHeight || 0
  const xStartPx = LEFT_MARGIN_PX
  const xEndPx = chartWidth - RIGHT_MARGIN_PX
  const yStartPx = (TOP_MARGIN_PCT / 100) * chartHeight
  const yEndPx = ((TOP_MARGIN_PCT + PLOT_AREA_PCT - 4) / 100) * chartHeight

  if (chartWidth <= 0 || chartHeight <= 0 || xEndPx <= xStartPx) {
    console.log(`${CURSOR_LOG_PREFIX} invalid chart size for click mapping`, {
      chartWidth,
      chartHeight,
      xStartPx,
      xEndPx,
    })
    return
  }

  if (point[0] < xStartPx || point[0] > xEndPx || point[1] < yStartPx || point[1] > yEndPx) {
    console.log(`${CURSOR_LOG_PREFIX} click outside plotting area`, {
      point,
      xStartPx,
      xEndPx,
      yStartPx,
      yEndPx,
    })
    return
  }

  const windowRange = getCurrentAxisWindow()
  if (!windowRange) {
    console.log(`${CURSOR_LOG_PREFIX} current axis window unavailable`)
    return
  }

  const ratio = (point[0] - xStartPx) / (xEndPx - xStartPx)
  const clampedRatio = Math.max(0, Math.min(1, ratio))
  const axisValue = windowRange.start + clampedRatio * (windowRange.end - windowRange.start)

  console.log(`${CURSOR_LOG_PREFIX} pixel->axis result`, {
    point,
    ratio,
    clampedRatio,
    axisValue,
    windowRange,
    xAxesType: xAxesType.value,
  })

  const nextIndex =
    xAxesType.value === XAxesType.Index
      ? Math.round(axisValue)
      : binarySearchIndex(timestamps, axisValue)

  console.log(`${CURSOR_LOG_PREFIX} computed nextIndex`, {
    axisValue,
    nextIndex,
    firstTimestamp: timestamps[0],
    lastTimestamp: timestamps[timestamps.length - 1],
  })

  cursorSampleIndex.value = Math.max(0, Math.min(nextIndex, timestamps.length - 1))
  console.log(`${CURSOR_LOG_PREFIX} cursor updated`, {
    cursorSampleIndex: cursorSampleIndex.value,
    cursorTime: timestamps[cursorSampleIndex.value] ?? null,
  })
  renderChart()
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
    cursorSampleIndex.value = null
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
  console.log(`${CURSOR_LOG_PREFIX} renderChart`, {
    xAxesType: xAxesType.value,
    cursorSampleIndex: cursorSampleIndex.value,
    timestampsLength: timestamps.length,
    channelCount: channelValues.length,
  })
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

  const grids = channelValues.map((_, i) => ({
    left: LEFT_MARGIN_PX,
    right: RIGHT_MARGIN_PX,
    top: `${topMarginPct + i * perGridPct}%`,
    height: `${perGridPct - 4}%`,
  }))

  const xAxes = channelValues.map((_, i) => ({
    type: 'value' as const,
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

  // 绘制游标线（graphic使用像素坐标，不是坐标轴数据值）
  const cursorAxisValue = getCursorAxisValue()
  const graphicElements = []
  if (typeof cursorAxisValue === 'number') {
    const windowRange = getCurrentAxisWindow()
    const chartHeight = chartRef.value?.clientHeight || 0
    const chartWidth = chartRef.value?.clientWidth || 0
    const xStartPx = LEFT_MARGIN_PX
    const xEndPx = chartWidth - RIGHT_MARGIN_PX
    const xSpan = (windowRange?.end ?? 0) - (windowRange?.start ?? 0)
    const ratio = xSpan === 0 ? 0 : (cursorAxisValue - (windowRange?.start ?? 0)) / xSpan
    const clampedRatio = Math.max(0, Math.min(1, ratio))
    const cursorPixelX = xStartPx + clampedRatio * (xEndPx - xStartPx)

    const cursorTopY = (topMarginPct / 100) * chartHeight
    const cursorBottomY = ((topMarginPct + plotAreaPct - 4) / 100) * chartHeight

    console.log(`${CURSOR_LOG_PREFIX} adding cursor line at axis value`, {
      cursorAxisValue,
      cursorPixelX,
      windowRange,
      ratio,
      clampedRatio,
      cursorTopY,
      cursorBottomY,
      chartWidth,
      chartHeight,
      initialWindow,
    })

    graphicElements.push({
      type: 'line',
      shape: {
        x1: cursorPixelX,
        y1: cursorTopY,
        x2: cursorPixelX,
        y2: cursorBottomY,
      },
      style: {
        stroke: 'red', // #0A3D91, #0B5FFF, #1E3A8A
        lineWidth: 1.5,
        lineDash: [4, 4],
      },
      z: 10,
    })
  }

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
      // 鼠标悬停时显示所有通道的瞬时值和RMS，且按照channelValues中的顺序显示
      formatter: function (params) {
        // 根据channelValues中的顺序排序
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
        let hoverIndex = typeof xValue === 'number' ? xValue : 0
        let hoverTime = 0
        if (xAxesType.value === XAxesType.Index) {
          // 显示为时间戳
          hoverIndex = Math.max(0, Math.min(Number(xValue) || 0, timestamps.length - 1))
          hoverTime = timestamps[hoverIndex] || 0
          xValue = hoverTime
        } else {
          hoverTime = Number(xValue) || 0
          hoverIndex = binarySearchIndex(timestamps, hoverTime)
        }
        const cursorTime =
          cursorSampleIndex.value === null
            ? null
            : (timestamps[Math.max(0, Math.min(cursorSampleIndex.value, timestamps.length - 1))] ??
              null)
        const relativeInfo =
          cursorTime === null ? '' : ` (相对游标: ${formatRelativeMs(hoverTime - cursorTime)} ms)`
        return (
          `${xValue} ms${relativeInfo}<br/>` +
          sortedParams
            .map((item) => {
              const seriesName = typeof item.seriesName === 'string' ? item.seriesName : ''
              const channel = channelValues.find((s) => s.name === seriesName)
              const isDigital = channel?.type === 'digital'

              const yValue = Array.isArray(item.data) ? item.data[1] : item.data
              // 对模拟量保留3位小数，对开关量直接显示0或1
              const formatY = typeof yValue === 'number' && !isDigital ? yValue.toFixed(3) : yValue
              const rmsValue = channel ? computeCycleRms(channel, hoverIndex) : '-'

              const rmsPart = isDigital
                ? ''
                : `<span style="display:inline-block;min-width:110px;">有效值=${rmsValue}</span>`
              return `<span style="display:inline-block;margin-right:6px;width:8px;height:8px;border-radius:50%;background:${item.color};"></span>
                      <span style="display:inline-block;min-width:150px;">${seriesName}:</span>
                      <span style="display:inline-block;min-width:130px;">瞬时值=${formatY}</span>${rmsPart}`
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

    if (timestamps.length > 0 && cursorSampleIndex.value === null) {
      cursorSampleIndex.value = Math.max(0, Math.min(data.window.start, timestamps.length - 1))
      console.log(`${CURSOR_LOG_PREFIX} init cursor on refreshData`, {
        cursorSampleIndex: cursorSampleIndex.value,
        cursorTime: timestamps[cursorSampleIndex.value] ?? null,
        window: data.window,
      })
    }

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
