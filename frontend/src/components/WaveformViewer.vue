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
          <n-space>
            <n-button @click="showAnnotationModal = true" :disabled="loading">
              <template #icon>
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M12 5v14M5 12h14" />
                </svg>
              </template>
              添加标注
            </n-button>
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

    <!-- Annotation Modal -->
    <n-modal
      v-model:show="showAnnotationModal"
      preset="card"
      title="添加标注"
      style="width: 600px"
      :bordered="false"
      :segmented="{ content: 'soft' }"
    >
      <n-form ref="annotationFormRef" :model="annotationForm" :rules="annotationRules">
        <n-form-item label="类型" path="type">
          <n-radio-group v-model:value="annotationForm.type">
            <n-radio value="marker">标记点</n-radio>
            <n-radio value="region">区域</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item v-if="annotationForm.type === 'marker'" label="时间 (ms)" path="time">
          <n-input-number v-model:value="annotationForm.time" :step="0.1" style="width: 100%" />
        </n-form-item>
        <template v-if="annotationForm.type === 'region'">
          <n-form-item label="开始时间 (ms)" path="startTime">
            <n-input-number
              v-model:value="annotationForm.startTime"
              :step="0.1"
              style="width: 100%"
            />
          </n-form-item>
          <n-form-item label="结束时间 (ms)" path="endTime">
            <n-input-number
              v-model:value="annotationForm.endTime"
              :step="0.1"
              style="width: 100%"
            />
          </n-form-item>
        </template>
        <n-form-item label="标签" path="label">
          <n-input v-model:value="annotationForm.label" placeholder="输入标签名称" />
        </n-form-item>
        <n-form-item label="颜色" path="color">
          <n-color-picker v-model:value="annotationForm.color" :modes="['hex']" />
        </n-form-item>
        <n-form-item label="描述" path="description">
          <n-input
            v-model:value="annotationForm.description"
            type="textarea"
            placeholder="输入描述信息（可选）"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAnnotationModal = false">取消</n-button>
          <n-button type="primary" @click="handleSaveAnnotation" :loading="annotationSaving">
            保存
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Annotations List Drawer -->
    <n-drawer v-model:show="showAnnotationsDrawer" :width="400" placement="right">
      <n-drawer-content title="标注列表">
        <n-space vertical>
          <n-card v-for="ann in annotations" :key="ann.id" size="small" :bordered="true" hoverable>
            <template #header>
              <n-space align="center">
                <div
                  :style="{
                    width: '12px',
                    height: '12px',
                    borderRadius: '50%',
                    backgroundColor: ann.color || '#1890ff',
                  }"
                ></div>
                <span>{{ ann.label }}</span>
              </n-space>
            </template>
            <template #header-extra>
              <n-space>
                <n-button text @click="handleEditAnnotation(ann)">
                  <template #icon>
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    >
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                    </svg>
                  </template>
                </n-button>
                <n-button text @click="handleDeleteAnnotation(ann.id!)">
                  <template #icon>
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    >
                      <polyline points="3 6 5 6 21 6" />
                      <path
                        d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                      />
                    </svg>
                  </template>
                </n-button>
              </n-space>
            </template>
            <n-space vertical size="small">
              <div v-if="ann.type === 'marker'">
                <n-text depth="3">时间: </n-text>
                <n-text>{{ ann.time }} ms</n-text>
              </div>
              <div v-else-if="ann.type === 'region'">
                <n-text depth="3">时间范围: </n-text>
                <n-text>{{ ann.startTime }} ~ {{ ann.endTime }} ms</n-text>
              </div>
              <div v-if="ann.description">
                <n-text depth="3">描述: </n-text>
                <n-text>{{ ann.description }}</n-text>
              </div>
            </n-space>
          </n-card>
          <n-empty v-if="annotations.length === 0" description="暂无标注" />
        </n-space>
        <template #footer>
          <n-button block @click="showAnnotationModal = true"> 添加新标注 </n-button>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Floating Action Button -->
    <n-button circle type="primary" size="large" class="fab" @click="showAnnotationsDrawer = true">
      <template #icon>
        <n-badge :value="annotations.length" :max="99">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
            <line x1="16" y1="13" x2="8" y2="13" />
            <line x1="16" y1="17" x2="8" y2="17" />
            <polyline points="10 9 9 9 8 9" />
          </svg>
        </n-badge>
      </template>
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted, shallowRef } from 'vue'
import * as echarts from 'echarts'
import { useMessage, useDialog, type FormInst, type FormRules } from 'naive-ui'
import { useDatasetStore } from '../stores/dataset'
import { useViewStore } from '../stores/view'
import {
  getWaveforms,
  getAnnotations,
  createAnnotation,
  updateAnnotation,
  deleteAnnotation,
  type Annotation,
} from '../api'

const datasetStore = useDatasetStore()
const viewStore = useViewStore()
const message = useMessage()
const dialog = useDialog()
const chartRef = ref<HTMLElement>()
const chartInstance = shallowRef<echarts.ECharts>()
const loading = ref(false)
const sampleCount = ref(0)

// Annotations state
const annotations = ref<Annotation[]>([])
const showAnnotationModal = ref(false)
const showAnnotationsDrawer = ref(false)
const annotationSaving = ref(false)
const editingAnnotationId = ref<string | null>(null)
const annotationFormRef = ref<FormInst | null>(null)

// Annotation form
const annotationForm = ref<Annotation>({
  type: 'marker',
  time: 0,
  startTime: 0,
  endTime: 0,
  label: '',
  color: '#FF5722FF',
  description: '',
})

const annotationRules: FormRules = {
  label: [{ required: true, message: '请输入标签名称', trigger: 'blur' }],
  time: [
    {
      required: true,
      type: 'number',
      message: '请输入时间',
      trigger: 'blur',
    },
  ],
  startTime: [
    {
      required: true,
      type: 'number',
      message: '请输入开始时间',
      trigger: 'blur',
    },
  ],
  endTime: [
    {
      required: true,
      type: 'number',
      message: '请输入结束时间',
      trigger: 'blur',
    },
  ],
}

// Track initial data bounds for keeping X-axis consistent
let initialWindow = { start: 0, end: 0 }
const lastWindow = { start: 0, end: 0 }

onMounted(() => {
  if (chartRef.value) {
    chartInstance.value = echarts.init(chartRef.value)
    chartInstance.value.on('dataZoom', handleDataZoom)
    window.addEventListener('resize', resizeChart)

    refreshData()
    loadAnnotations()
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
    loadAnnotations()
  },
)

watch(annotations, () => {
  updateChartAnnotations()
})

async function loadAnnotations() {
  if (!datasetStore.currentId) return
  try {
    annotations.value = await getAnnotations(datasetStore.currentId)
  } catch (e) {
    console.error('Failed to load annotations:', e)
  }
}

function handleEditAnnotation(ann: Annotation) {
  editingAnnotationId.value = ann.id || null
  annotationForm.value = { ...ann }
  showAnnotationModal.value = true
}

async function handleSaveAnnotation() {
  if (!annotationFormRef.value) return
  await annotationFormRef.value.validate(async (errors) => {
    if (errors) return

    if (!datasetStore.currentId) {
      message.error('未选择数据集')
      return
    }

    annotationSaving.value = true
    try {
      if (editingAnnotationId.value) {
        // Update existing annotation
        await updateAnnotation(
          datasetStore.currentId,
          editingAnnotationId.value,
          annotationForm.value,
        )
        message.success('标注已更新')
      } else {
        // Create new annotation
        await createAnnotation(datasetStore.currentId, annotationForm.value)
        message.success('标注已添加')
      }

      await loadAnnotations()
      showAnnotationModal.value = false
      resetAnnotationForm()
    } catch (e) {
      message.error('保存标注失败')
      console.error(e)
    } finally {
      annotationSaving.value = false
    }
  })
}

async function handleDeleteAnnotation(annId: string) {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除这个标注吗？',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      if (!datasetStore.currentId) return
      try {
        await deleteAnnotation(datasetStore.currentId, annId)
        message.success('标注已删除')
        await loadAnnotations()
      } catch (e) {
        message.error('删除标注失败')
        console.error(e)
      }
    },
  })
}

function resetAnnotationForm() {
  editingAnnotationId.value = null
  annotationForm.value = {
    type: 'marker',
    time: initialWindow.start || 0,
    startTime: initialWindow.start || 0,
    endTime: initialWindow.end || 0,
    label: '',
    color: '#FF5722FF',
    description: '',
  }
}

function updateChartAnnotations() {
  if (!chartInstance.value) return

  const option = chartInstance.value.getOption() as echarts.EChartsOption
  if (!option || !option.series) return

  // Create markLine and markArea data for annotations
  const markLineData: echarts.MarkLineComponentOption['data'] = []
  const markAreaData: echarts.MarkAreaComponentOption['data'] = []

  annotations.value.forEach((ann) => {
    if (ann.type === 'marker' && ann.time !== undefined) {
      markLineData.push({
        name: ann.label,
        xAxis: ann.time,
        label: {
          formatter: ann.label,
          position: 'end',
        },
        lineStyle: {
          color: ann.color || '#FF5722',
          width: 2,
          type: 'solid',
        },
      })
    } else if (ann.type === 'region' && ann.startTime !== undefined && ann.endTime !== undefined) {
      markAreaData.push([
        {
          name: ann.label,
          xAxis: ann.startTime,
          itemStyle: {
            color: ann.color ? ann.color + '30' : '#FF572230',
          },
          label: {
            formatter: ann.label,
            position: 'top',
          },
        },
        {
          xAxis: ann.endTime,
        },
      ])
    }
  })

  // Update series with markLine and markArea
  if (Array.isArray(option.series)) {
    option.series = option.series.map((s, idx) => {
      const series = { ...s }
      // Only add annotations to the first series to avoid duplication
      if (idx === 0) {
        series.markLine = {
          symbol: ['none', 'none'],
          data: markLineData,
          animation: false,
        }
        series.markArea = {
          data: markAreaData,
          animation: false,
        }
      }
      return series
    })
  }

  chartInstance.value.setOption(option)
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

    // Store initial window on first load
    if (initialWindow.start === 0 && initialWindow.end === 0) {
      initialWindow = { start: data.window.start, end: data.window.end }
    }

    // Calculate zoom percentages if this is a zoomed refresh
    const span = initialWindow.end - initialWindow.start
    const zoomStartPct = ((data.timeRange.start - initialWindow.start) / span) * 100
    const zoomEndPct = ((data.timeRange.end - initialWindow.start) / span) * 100

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
      min: initialWindow.start,
      max: initialWindow.end,
      gridIndex: i,
      axisLabel: {
        show: i === seriesCount - 1,
        formatter: (value: number) => {
          if (i !== seriesCount - 1) {
            return value.toFixed(0)
          }
          return value.toFixed(0) + ' ms'
        },
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
    lastWindow.start = startTime !== undefined ? startTime : data.window.start
    lastWindow.end = endTime !== undefined ? endTime : data.window.end
    datasetStore.error = ''

    // Update annotations after chart is rendered
    updateChartAnnotations()
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    datasetStore.error = msg
  } finally {
    loading.value = false
  }
}

// Handle dataZoom events to load detailed data for zoomed region
function handleDataZoom(params: unknown) {
  if (!chartInstance.value) return

  const p = params as echarts.ECElementEvent
  // Get current zoom range from xAxis
  const dz =
    p.batch && Array.isArray(p.batch) && p.batch.length > 0 ? p.batch[0] : { start: 0, end: 100 }
  const startPct: number = typeof dz.start === 'number' ? dz.start : 0
  const endPct: number = typeof dz.end === 'number' ? dz.end : 100
  const span = initialWindow.end - initialWindow.start
  const zoomStart = initialWindow.start + (startPct / 100) * span
  const zoomEnd = initialWindow.start + (endPct / 100) * span

  // Only reload if zoomed significantly (more than 10% of range)
  const threshold = (lastWindow.end - lastWindow.start) * 0.1

  const zoomRange = zoomEnd - zoomStart
  const offsetRange = Math.max(
    Math.abs(zoomStart - lastWindow.start),
    Math.abs(zoomEnd - lastWindow.end),
  )
  if ((zoomRange < threshold && zoomRange > 0) || offsetRange > threshold) {
    // Zoomed in significantly, request detailed data
    refreshData(zoomStart, zoomEnd)
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
.fab {
  position: fixed;
  bottom: 32px;
  right: 32px;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>
