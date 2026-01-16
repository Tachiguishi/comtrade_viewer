<template>
  <div class="viewer-container">
    <n-card :bordered="false">
      <template #header>
        <n-space align="center" justify="space-between">
          <n-space :size="14" wrap>
            <n-tag type="info" size="small">站点: {{ datasetStore.metadata?.station }}</n-tag>
            <n-tag type="info" size="small">设备: {{ datasetStore.metadata?.relay }}</n-tag>
            <n-tag type="info" size="small">版本: {{ datasetStore.metadata?.version }}</n-tag>
            <n-tag type="info" size="small"
              >数据类型: {{ datasetStore.metadata?.dataFileType.toUpperCase() }}</n-tag
            >
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
            <n-tag type="default" size="small">开始: {{ datasetStore.metadata?.startTime }}</n-tag>
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
    const data = await getWaveforms(
      datasetStore.currentId,
      viewStore.selectedAnalogChannels,
      viewStore.selectedDigitalChannels,
    )

    // 规范化数字通道数据：确保值只有0和1
    const normalizedSeries = data.series.map((s) => {
      if (s.type === 'digital') {
        // 对于数字通道，规范化值为0或1
        const normalizedY = s.y.map((val) => (val !== 0 ? 1 : 0))
        return { ...s, y: normalizedY }
      }
      return s
    })

    // sampleCount.value = data.times.length
    const seriesCount = normalizedSeries.length
    const axesIndices = Array.from({ length: seriesCount }, (_, i) => i)

    sampleCount.value = data.downsample.originalPoints

    // 预留顶部/底部空间给标题/缩放器，按百分比垂直堆叠各 grid
    const plotAreaPct = 95 // 95% 高度作为绘图区
    const topMarginPct = 4
    const perGridPct = plotAreaPct / Math.max(seriesCount, 1)
    const LEFT_MARGIN_PX = 50
    const RIGHT_MARGIN_PX = 30

    const grids = normalizedSeries.map((_, i) => ({
      left: LEFT_MARGIN_PX,
      right: RIGHT_MARGIN_PX,
      top: `${topMarginPct + i * perGridPct}%`,
      height: `${perGridPct - 4}%`,
    }))

    const xAxes = normalizedSeries.map((_, i) => ({
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

    const yAxes = normalizedSeries.map((s, i) => {
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
          interval: 1,
          showMinLabel: true,
          showMaxLabel: true,
        }
        yAxisConfig.splitLine = { show: false }
        yAxisConfig.axisTick = {
          show: true,
          interval: 1,
          length: 3,
        }
      }

      return yAxisConfig
    })

    // 获取图表容器宽度
    const DEFAULT_CHART_WIDTH = 800
    const chartWidth = chartRef.value?.clientWidth || DEFAULT_CHART_WIDTH

    // 为每个子图添加底部边框线
    const graphicElements = normalizedSeries.flatMap((_, i) => {
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
          // 根据normalizedSeries中的顺序排序显示
          const sortedParams = Array.isArray(params)
            ? params.slice().sort((a, b) => {
                const seriesA = normalizedSeries.findIndex((s) => s.name === a.seriesName)
                const seriesB = normalizedSeries.findIndex((s) => s.name === b.seriesName)
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
      series: normalizedSeries.map((s, i) => {
        // 根据通道类型选择不同的渲染方式
        const isDigital = s.type === 'digital'

        return {
          name: s.name,
          type: 'line',
          step: isDigital ? 'start' : false, // 开关量使用阶梯图
          showSymbol: false,
          xAxisIndex: i,
          yAxisIndex: i,
          data: s.y.map((y, k) => [s.times[k], y]),
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
