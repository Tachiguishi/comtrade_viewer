<template>
  <div class="waveViewer">
    <div v-show="waveData.chns && waveData.chns.length > 0">
      <div class="option-buttons">
        <button @click="horizontalZoom(true)" type="button">水平放大</button>
        <button @click="horizontalZoom(false)" type="button">水平缩小</button>
        <!-- <button @click="verticalZoom(10)" type="button">垂直放大</button> -->
        <!-- <button @click="verticalZoom(-10)" type="button">垂直缩小</button> -->
        <button @click="reStore()" type="button">还原波形</button>
      </div>

      <div class="header-container">
        <!-- title -->
        <div
          class="header-title"
          :style="{
            left: state.xmargin + 'px',
          }"
        >
          录波文件：{{ props.fileName }}
          <span>开始时间：{{ state.beginTime }}</span>
          <span>时标差：{{ state.timeDiff / 1000 }}ms</span>
        </div>
        <!-- timestamp labels -->
        <div>
          <div
            class="timeLabel"
            v-for="time in timestampLabels"
            :key="time.index"
            :style="{
              top: state.ymargin - 18 + 'px',
              left: time.offset + 'px',
            }"
          >
            {{ time.label }}
          </div>
        </div>
        <!-- timestamp tick -->
        <canvas ref="rulerCanvas"></canvas>
      </div>

      <!-- 波形容器 -->
      <div ref="waveCanvasContainer" id="waveDiv" class="wave-container">
        <div class="channel-info">
          <div
            v-for="channel in channelsInfo"
            :key="channel.index"
            :style="{
              position: 'absolute',
              top: channel.offset + 'px',
              left: state.xmargin + 10 + 'px',
              color: channel.color,
            }"
          >
            {{ channel.name }}
          </div>
          <div
            v-for="channel in channelsValue"
            :key="channel.index + '_values'"
            :style="{
              position: 'absolute',
              top: channel.offset + 'px',
              right: '30px',
              color: channel.color,
            }"
          >
            有效值: {{ channel.rms }} 瞬时值: {{ channel.instant }}
          </div>
        </div>
        <canvas ref="waveCanvas" id="waveContent"></canvas>
      </div>

      <!-- 蓝色游标 -->
      <div
        ref="blueCursor"
        class="cursor-line-container"
        :style="{
          top: state.ymargin + state.rulerYMarginGap + 'px',
          left: blueCursorPos - 10 + 'px',
          height: state.canvasH - state.ymargin - state.rulerYMarginGap + 'px',
        }"
      >
        <div class="cursor-line blue-line"></div>
      </div>

      <!-- 绿色游标 -->
      <div
        ref="greenCursor"
        class="cursor-line-container"
        :style="{
          top: state.ymargin + state.rulerYMarginGap + 'px',
          left: greenCursorPos - 10 + 'px',
          height: state.canvasH - state.ymargin - state.rulerYMarginGap + 'px',
        }"
      >
        <div class="cursor-line green-line"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, nextTick } from 'vue'
import { getWaveCanvas } from '@/api'
import type { WaveDataType, ValueData } from '@/utils/comtrade.ts'
import { ValueFormatter, GetCurrentValue } from '@/utils/comtrade.ts'

// ==================== 常量定义 ====================

/** 默认X边距 */
const DEFAULT_X_MARGIN = 15
/** 默认Y边距 */
const DEFAULT_Y_MARGIN = 50
/** 时间像素间隔 */
const TIME_PIXEL_INTERVAL = 8
/** 通道间隔 */
const DEFAULT_CHANNEL_GAP = 90
/** 刻度与Y边距之间的距离 */
const RULER_Y_MARGIN_GAP = 10
/** 最小通道间隔 */
// const MIN_CHANNEL_GAP = 70
/** 最大通道间隔 */
// const MAX_CHANNEL_GAP = 150
/** 最大缩放级别 */
const MAX_ZOOM_LEVEL = 5
/** 最小缩放级别 */
const MIN_ZOOM_LEVEL = -5
/** 游标线宽度 */
const CURSOR_LINE_WIDTH = 2
/** 时间戳标签间隔 */
const TIMESTAMP_LABEL_INTERVAL = 100
/** 虚线间隔 */
const DASH_LINE_INTERVAL = 2

/** 相位颜色映射 */
const PHASE_COLORS = {
  A: 'rgb(255,255,0)', // 黄色
  B: 'rgb(0,255,0)', // 绿色
  C: 'rgb(255,0,0)', // 红色
  N: 'rgb(0,255,255)', // 青色
  DIGITAL: 'rgb(255,128,0)', // 橙色
} as const

/** 界面颜色 */
const UI_COLORS = {
  BACKGROUND: '#000',
  TEXT: '#fff',
  TITLE: '#4ae3ed',
  TIME_LABEL: '#78d1dc',
  RULER: '#006686',
  DASH_LINE: '#bbb',
} as const

// ==================== 类型定义 ====================

/** 组件属性 */
interface Props {
  /** 文件目录 */
  fileDirectory: string
  /** 文件名 */
  fileName: string
  /** 随机时间戳，用于触发刷新 */
  randomTime: string
}

/** 通道信息（用于显示） */
type ChannelInfo = {
  /** 通道索引 */
  index: number
  /** 通道名称 */
  name: string
  /** 显示颜色 */
  color: string
  /** 垂直偏移量 */
  offset: number
  /** 有效值 (RMS) */
  rms?: number
  /** 瞬时值 */
  instant?: number
}

/** 时间戳标签 */
type TimestampLabel = {
  /** 标签文本 */
  label: string
  /** 数据索引 */
  index: number
  /** 水平偏移量 */
  offset: number
}

/** 画布状态 */
type CanvasState = {
  /** 2D渲染上下文 */
  context: CanvasRenderingContext2D | null
  /** 刻度尺渲染上下文 */
  rulerContext: CanvasRenderingContext2D | null
  /** 值格式化器 */
  formatter: ValueFormatter | null
  /** 当前值数组 */
  valueArr: ValueData[]

  // 尺寸相关
  /** 画布宽度 */
  canvasW: number
  /** 画布高度 */
  canvasH: number
  /** 刻度尺高度 */
  rulerH: number
  /** X边距 */
  xmargin: number
  /** Y边距 */
  ymargin: number
  /** 时间像素间隔 */
  tspixel: number
  /** 通道间隔 */
  gap: number
  /** 通道间隔副本 */
  gapcp: number
  /** 刻度与Y边距间距 */
  rulerYMarginGap: number

  // 游标相关
  /** 蓝色游标加偏移 */
  cursoradd: number
  /** 蓝色游标减偏移 */
  cursorsub: number
  /** 绿色游标加偏移 */
  cursoradd1: number
  /** 绿色游标减偏移 */
  cursorsub1: number

  // 缩放相关
  /** 正向缩放像素 */
  pix: number
  /** 负向缩放像素 */
  pixneg: number
  /** 页面起始索引 */
  pagestart: number
  /** 页面结束索引 */
  pagelast: number

  // 其他
  /** 时间差（微秒） */
  timeDiff: number
  /** 当前值的颜色 */
  valueColor: string
  /** 开始时间 */
  beginTime: string
  /** 画布容器高度 */
  canvasDivH: number
  /** 堆栈 */
  stack: number[]
}

const props = withDefaults(defineProps<Props>(), {
  fileDirectory: '',
  fileName: '',
  randomTime: '',
})

// ==================== Refs ====================
/** 波形画布元素 */
const waveCanvas = ref<HTMLCanvasElement | null>(null)
/** 刻度尺画布元素 */
const rulerCanvas = ref<HTMLCanvasElement | null>(null)
/** 波形画布容器 */
const waveCanvasContainer = ref<HTMLDivElement | null>(null)
/** 蓝色游标元素 */
const blueCursor = ref<HTMLDivElement | null>(null)
/** 绿色游标元素 */
const greenCursor = ref<HTMLDivElement | null>(null)

/** 蓝色游标位置 */
const blueCursorPos = ref<number>(0)
/** 绿色游标位置 */
const greenCursorPos = ref<number>(0)

// ==================== 响应式数据 ====================

/** 波形数据 */
const waveData = reactive<Partial<WaveDataType>>({})

/** 画布状态 */
const state = reactive<CanvasState>({
  context: null,
  rulerContext: null,
  formatter: null,
  valueArr: [],

  // 尺寸相关
  canvasW: 0,
  canvasH: 0,
  rulerH: 0,
  xmargin: DEFAULT_X_MARGIN,
  ymargin: DEFAULT_Y_MARGIN,
  tspixel: TIME_PIXEL_INTERVAL,
  gap: DEFAULT_CHANNEL_GAP,
  gapcp: DEFAULT_CHANNEL_GAP,
  rulerYMarginGap: RULER_Y_MARGIN_GAP,

  // 游标相关
  cursoradd: 0,
  cursorsub: 0,
  cursoradd1: 0,
  cursorsub1: 0,

  // 缩放相关
  pix: 1,
  pixneg: 1,
  pagestart: 0,
  pagelast: 0,

  // 其他
  timeDiff: 0,
  valueColor: UI_COLORS.TEXT,
  beginTime: '',
  canvasDivH: 0,
  stack: [],
})

/** 时间戳标签数组 */
const timestampLabels = reactive<TimestampLabel[]>([])

/** 通道信息数组 */
const channelsInfo = reactive<ChannelInfo[]>([])

/** 通道值数组 */
const channelsValue = reactive<ChannelInfo[]>([])

// ==================== 工具函数 ====================

/**
 * 根据通道相位信息获取显示颜色
 * @param adType 信号类型 ('A'=模拟量, 'D'=数字量)
 * @param phase 相位 ('A', 'B', 'C', 'N')
 * @returns RGB颜色字符串
 */
function getChannelColor(adType?: string, phase?: string): string {
  if (adType !== 'A') {
    return PHASE_COLORS.DIGITAL
  }

  switch (phase) {
    case 'A':
      return PHASE_COLORS.A
    case 'B':
      return PHASE_COLORS.B
    case 'C':
      return PHASE_COLORS.C
    case 'N':
      return PHASE_COLORS.N
    default:
      return PHASE_COLORS.DIGITAL
  }
}

/**
 * 根据游标的像素位置计算数据索引
 * @param cursorPixelPos 游标的像素位置
 * @returns 数据索引
 */
function getDataIndexByCursorPosition(cursorPixelPos: number): number {
  let dataIndex = 0

  if (state.pix > 1) {
    // 放大模式：每个数据点占多个像素
    dataIndex = state.pagestart + Math.floor((cursorPixelPos - state.xmargin) / state.pix)
  } else {
    // 缩小模式：多个数据点共享一个像素
    dataIndex = state.pagestart + (cursorPixelPos - state.xmargin) * state.pixneg
  }

  return dataIndex
}

// ==================== 数据获取 ====================

/**
 * 从服务器获取波形数据
 */
async function getData() {
  try {
    const { data: response } = await getWaveCanvas('/tmp', 'file')
    console.log(response)

    if (response.flag) {
      loadWaveData(response.result)
    }
  } catch (error) {
    console.error('加载波形数据失败:', error)
  }
}

// ==================== 监听器 ====================
watch([() => props.fileDirectory, () => props.fileName, () => props.randomTime], () => getData())

// ==================== 生命周期 ====================
onMounted(() => {
  nextTick(() => {
    init()
    setupEventListeners()
    getData()
  })
})

// ==================== 初始化 ====================

/**
 * 初始化画布和渲染上下文
 */
function init(): void {
  if (!waveCanvas.value || !rulerCanvas.value || !waveCanvasContainer.value) {
    console.error('Canvas or container element not found')
    return
  }

  const parentElement = waveCanvasContainer.value.parentElement
  const grandParentElement = parentElement?.parentElement

  state.canvasW = grandParentElement?.offsetWidth || 0
  state.canvasH = grandParentElement?.offsetHeight || 0

  console.log('Window Width:', state.canvasW, 'Window Height:', state.canvasH)

  state.rulerH = state.canvasH
  state.gapcp = state.gap

  waveCanvas.value.width = state.canvasW
  rulerCanvas.value.width = state.canvasW
  rulerCanvas.value.height = state.rulerH

  waveCanvasContainer.value.style.width = `${state.canvasW}px`
  waveCanvasContainer.value.style.height = `${state.canvasH - state.ymargin}px`

  state.context = waveCanvas.value.getContext('2d')
  state.rulerContext = rulerCanvas.value.getContext('2d')
}

// ==================== 事件监听设置 ====================

/**
 * 设置所有事件监听器
 */
const setupEventListeners = (): void => {
  if (!waveCanvasContainer.value || !blueCursor.value || !greenCursor.value) {
    console.error('所需元素未找到')
    return
  }

  // 点击波形容器设置蓝色游标位置
  waveCanvasContainer.value.addEventListener('click', (event: MouseEvent) => {
    const parentRect = waveCanvasContainer.value!.parentElement!.getBoundingClientRect()
    blueCursorPos.value = Math.floor(event.clientX - parentRect.left)
    updateChannelValues(waveData as WaveDataType, true)
  })

  // 设置蓝色游标拖动
  setupCursorDrag(blueCursor.value, true)

  // 设置绿色游标拖动
  setupCursorDrag(greenCursor.value, false)

  // 初始化游标位置
  blueCursorPos.value = 50
  greenCursorPos.value = 250
}

// ==================== 游标拖动功能 ====================

/**
 * 设置游标拖动功能
 * @param cursorElement 游标元素
 * @param isBlue 是否为蓝色游标
 */
const setupCursorDrag = (cursorElement: HTMLDivElement, isBlue: boolean): void => {
  cursorElement.addEventListener('mousedown', () => {
    let isDragging = true
    updateChannelValues(waveData as WaveDataType, isBlue)

    const handleMouseMove = (event: MouseEvent) => {
      if (!isDragging || !waveCanvasContainer.value) return

      const parentRect = waveCanvasContainer.value.parentElement!.getBoundingClientRect()
      const newPosition = Math.floor(event.clientX - parentRect.left)

      if (isBlue) {
        blueCursorPos.value = newPosition
      } else {
        greenCursorPos.value = newPosition
      }

      updateChannelValues(waveData as WaveDataType, isBlue)
    }

    const handleMouseUp = () => {
      isDragging = false
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  })
}

// ==================== 数据加载 ====================

/**
 * 加载波形数据并初始化相关状态
 * @param result 波形数据
 */
const loadWaveData = (result: WaveDataType): void => {
  Object.assign(waveData, result)

  if (!result.chns || result.chns.length === 0) {
    return
  }

  const channels = result.chns
  let totalHeight = state.gap

  // 计算总高度
  for (let j = 0; j < channels.length; j++) {
    totalHeight = state.gap * (j + 2)
  }

  state.formatter = new ValueFormatter(result)
  state.canvasH = totalHeight + state.ymargin

  if (waveCanvas.value) {
    waveCanvas.value.height = state.canvasH
  }

  // 验证游标位置
  if (getDataIndexByCursorPosition(blueCursorPos.value) >= result.ts.length) {
    blueCursorPos.value = state.xmargin
  }
  if (getDataIndexByCursorPosition(greenCursorPos.value) >= result.ts.length) {
    greenCursorPos.value = state.xmargin + 200
  }

  // 计算时间差
  const time1 = result.ts[getDataIndexByCursorPosition(blueCursorPos.value)]
  const time2 = result.ts[getDataIndexByCursorPosition(greenCursorPos.value)]

  if (typeof time1 === 'number' && typeof time2 === 'number') {
    state.timeDiff = time2 - time1
  } else {
    state.timeDiff = 0
  }

  state.beginTime = result.beginTime.slice(0, result.beginTime.length - 3)

  renderWaveform(result)
  renderRuler(result)
}

/**
 * 渲染波形
 */
const renderWaveform = (result: WaveDataType): void => {
  if (!state.context) return

  state.context.clearRect(state.xmargin, state.ymargin, state.canvasW, state.canvasH)
  drawwave(result, state.context)
}

/**
 * 渲染刻度尺
 */
const renderRuler = (result: WaveDataType): void => {
  if (!state.rulerContext) return

  state.rulerContext.clearRect(0, 0, state.canvasW, state.rulerH)
  drawTimestampTick(state.rulerContext, state.canvasW, state.rulerH)
  calculateTimestampLabels(result)
}

// ==================== 绘图函数 ====================

/**
 * 计算通道的最大值比例
 * @param channels 通道数组
 * @returns 最大电压和最大电流
 */
function calculateMaxValues(channels: WaveDataType['chns']): {
  maxVoltage: number
  maxCurrent: number
} {
  const voltageMaxValues: number[] = []
  const currentMaxValues: number[] = []

  for (const channel of channels) {
    if (!channel || !Array.isArray(channel.y)) continue

    if (channel.uu === 'V') {
      voltageMaxValues.push(Math.max(...channel.y))
    } else if (channel.uu === 'A') {
      currentMaxValues.push(Math.max(...channel.y))
    }
  }

  return {
    maxVoltage: Math.max(...voltageMaxValues, 0),
    maxCurrent: Math.max(...currentMaxValues, 0),
  }
}

/**
 * 绘制波形
 * @param result 波形数据
 * @param context 2D渲染上下文
 */
const drawwave = (result: WaveDataType, context: CanvasRenderingContext2D): void => {
  const channels = result.chns
  const allSelector = result.allSelector

  // 计算最大值
  const { maxVoltage, maxCurrent } = calculateMaxValues(channels)
  const shouldUseGlobalMax = channels.length < 4

  // 获取当前游标位置的值
  const currentCursorPos = state.valueColor === 'green' ? greenCursorPos.value : blueCursorPos.value
  state.valueArr = state.formatter!.getValueDataByIndex(
    getDataIndexByCursorPosition(currentCursorPos),
    false,
  )

  // 清空通道信息
  channelsInfo.splice(0, channelsInfo.length)
  channelsValue.splice(0, channelsValue.length)

  // 绘制每个通道
  for (let i = 0; i < channels.length; i++) {
    const channel = channels[i]
    if (!channel) continue

    // 计算通道线的垂直位置
    const channelLineY = calculateChannelLineY(i)

    // 获取通道颜色
    const channelColor = getChannelColor(allSelector[i]?.AD, allSelector[i]?.phase)

    // 创建通道信息
    const channelInfo = createChannelInfo(channel, i, channelColor, channelLineY)
    channelsInfo.push(channelInfo)

    // 创建通道值信息
    const channelValueInfo = createChannelValueInfo(channel, i, channelColor, channelLineY)
    channelsValue.push(channelValueInfo)

    // 绘制通道基准线
    drawChannelBaselines(context, channelLineY)

    // 绘制波形
    drawChannelWaveform(
      context,
      channel,
      channelLineY,
      channelColor,
      maxVoltage,
      maxCurrent,
      shouldUseGlobalMax,
    )
  }
}

/**
 * 计算通道线的垂直位置
 * @param channelIndex 通道索引
 * @returns Y坐标
 */
function calculateChannelLineY(channelIndex: number): number {
  const gapRatio = (state.gapcp - state.gap) / state.gapcp
  let lineY = channelIndex * state.gap + state.ymargin + 65 * gapRatio

  if (lineY < state.ymargin + 100) {
    lineY = channelIndex * 100 + state.ymargin
  }

  return lineY
}

/**
 * 创建通道信息对象
 */
function createChannelInfo(
  channel: WaveDataType['chns'][0],
  index: number,
  color: string,
  lineY: number,
): ChannelInfo {
  return {
    index,
    name: channel!.name,
    color,
    offset: lineY - 18,
  }
}

/**
 * 创建通道值信息对象
 */
function createChannelValueInfo(
  channel: WaveDataType['chns'][0],
  index: number,
  color: string,
  lineY: number,
): ChannelInfo {
  const valueObj = state.valueArr.find((v) => v?.index === index)
  const channelValue: ChannelInfo = {
    index,
    name: channel!.name,
    color,
    offset: lineY - 18,
  }

  if (valueObj) {
    const value = GetCurrentValue(valueObj)
    channelValue.rms = value.a
    channelValue.instant = value.b
  }

  return channelValue
}

/**
 * 绘制通道的基准线（水平线和虚线）
 */
function drawChannelBaselines(context: CanvasRenderingContext2D, lineY: number): void {
  // 绘制上下边界线
  drawLine(
    context,
    state.xmargin,
    lineY + 1,
    state.canvasW - state.xmargin,
    lineY + 1,
    UI_COLORS.TEXT,
  )
  drawLine(
    context,
    state.xmargin,
    lineY - 15,
    state.canvasW - state.xmargin,
    lineY - 15,
    UI_COLORS.TEXT,
  )

  // 绘制中心虚线
  const centerY = lineY + state.gap / 2 - 7
  drawDashedLine(context, state.xmargin, centerY, state.canvasW - 5, centerY)
}

/**
 * 绘制通道波形
 */
function drawChannelWaveform(
  context: CanvasRenderingContext2D,
  channel: WaveDataType['chns'][0],
  lineY: number,
  color: string,
  maxVoltage: number,
  maxCurrent: number,
  shouldUseGlobalMax: boolean,
): void {
  if (!channel) return

  const yPoints = channel.y
  let maxValue = 10

  // 确定最大值
  if (!shouldUseGlobalMax) {
    if (channel.uu === 'V') {
      maxValue = maxVoltage
    } else if (channel.uu === 'A') {
      maxValue = maxCurrent
    } else {
      maxValue = Math.max(...yPoints) * 2
    }
  } else {
    maxValue = Math.max(...yPoints)
  }

  if (maxValue === 0) maxValue = 1 // 防止除以零

  context.beginPath()
  context.lineWidth = 1
  context.strokeStyle = color

  const centerY = lineY + state.gap / 2 - 7
  const ratio = state.gap / 3

  context.moveTo(state.xmargin, centerY)

  let pixelX = 0
  for (let j = state.pagestart; j < yPoints.length; j++) {
    const currentY = yPoints[j]
    const nextY = yPoints[j + 1]

    if (currentY === undefined || nextY === undefined) continue

    if (j % state.pixneg === 0) {
      const normalizedY = centerY - (nextY / maxValue) * ratio

      if (pixelX <= state.canvasW - state.xmargin * 2) {
        context.lineTo(pixelX + 1 + state.xmargin, normalizedY)
        state.pagelast = j
      }

      pixelX += state.pix
    }
  }

  context.stroke()
  context.closePath()
}

/**
 * 绘制实线
 */
const drawLine = (
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  endX: number,
  endY: number,
  color: string,
  lineWidth: number = 1,
): void => {
  context.beginPath()
  context.lineWidth = lineWidth
  context.strokeStyle = color
  context.moveTo(x, y)
  context.lineTo(endX, endY)
  context.stroke()
  context.closePath()
}

/**
 * 绘制虚线
 */
const drawDashedLine = (
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  endX: number,
  endY: number,
): void => {
  context.beginPath()
  context.lineWidth = 1
  context.strokeStyle = UI_COLORS.DASH_LINE

  for (let i = 0; i < endX - state.xmargin * 2; i++) {
    if (i % (state.tspixel * DASH_LINE_INTERVAL) === 0) {
      context.moveTo(x + i, y)
      context.lineTo(x + i + state.tspixel, endY)
      context.stroke()
    }
  }

  context.closePath()
}

/**
 * 计算时间轴标签的位置和文本
 * @param result 波形数据
 */
function calculateTimestampLabels(result: WaveDataType): void {
  const timeStamps = result.ts
  let pixelOffset = 0
  let shouldAddLabel = true

  timestampLabels.splice(0, timestampLabels.length)

  for (let j = state.pagestart; j < timeStamps.length; j++) {
    const nextTime = timeStamps[j + 1]
    if (!nextTime) continue

    if (j % TIMESTAMP_LABEL_INTERVAL === 0) {
      shouldAddLabel = false

      if (pixelOffset <= state.canvasW - state.xmargin * 2) {
        if (j % (state.pixneg * TIMESTAMP_LABEL_INTERVAL) === 0) {
          shouldAddLabel = true
          const currentTime = timeStamps[j]

          if (typeof currentTime === 'number') {
            const timeInMs = currentTime / 1000
            timestampLabels.push({
              label: `${timeInMs}ms`,
              index: j,
              offset: pixelOffset,
            })
          }
        }
      }
    }

    if (shouldAddLabel) {
      // 更新游标相关位置
      updateCursorOffsets(pixelOffset, j)
      pixelOffset += state.pix
    }
  }
}

/**
 * 更新游标偏移量
 */
function updateCursorOffsets(pixelOffset: number, dataIndex: number): void {
  if (pixelOffset === blueCursorPos.value) {
    blueCursorPos.value = blueCursorPos.value === 0 ? dataIndex + 150 : blueCursorPos.value
    greenCursorPos.value = greenCursorPos.value === 0 ? dataIndex + 300 : greenCursorPos.value
  }

  if (pixelOffset === blueCursorPos.value + 10) {
    state.cursoradd = dataIndex
  }
  if (pixelOffset === blueCursorPos.value - 10) {
    state.cursorsub = dataIndex
  }
  if (pixelOffset === greenCursorPos.value + 10) {
    state.cursoradd1 = dataIndex
  }
  if (pixelOffset === greenCursorPos.value - 10) {
    state.cursorsub1 = dataIndex
  }
}

/**
 * 绘制时间轴刻度尺
 * @param context 2D渲染上下文
 * @param width 画布宽度
 * @param height 画布高度
 */
function drawTimestampTick(context: CanvasRenderingContext2D, width: number, height: number): void {
  // 绘制边框
  context.beginPath()
  context.lineWidth = CURSOR_LINE_WIDTH
  context.strokeStyle = UI_COLORS.RULER
  context.moveTo(state.xmargin, state.ymargin)
  context.lineTo(state.xmargin, height)
  context.lineTo(width - state.xmargin, height)
  context.lineTo(width - state.xmargin, state.ymargin)
  context.stroke()

  // 绘制水平分隔线
  context.moveTo(state.xmargin, state.ymargin + state.rulerYMarginGap)
  context.lineTo(width - state.xmargin, state.ymargin + state.rulerYMarginGap)
  context.stroke()
  context.closePath()

  // 绘制刻度线
  for (let i = state.xmargin; i < width - state.xmargin; i++) {
    if (i % 5 === 0) {
      if (i % state.tspixel === 0) {
        drawTickMark(context, i, state.ymargin + state.rulerYMarginGap, 8)
      }
    } else {
      if (i % state.tspixel === 0) {
        drawTickMark(context, i, state.ymargin + state.rulerYMarginGap, 4)
      }
    }
  }
}

/**
 * 绘制刻度线
 */
const drawTickMark = (
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  height: number,
): void => {
  context.beginPath()
  context.lineWidth = CURSOR_LINE_WIDTH
  context.strokeStyle = UI_COLORS.RULER
  context.moveTo(x, y)
  context.lineTo(x, y - height)
  context.stroke()
  context.closePath()
}

/**
 * 根据游标位置更新通道的有效值和瞬时值
 * @param result 波形数据
 * @param isBlue 是否为蓝色游标
 */
const updateChannelValues = (result: WaveDataType, isBlue: boolean): void => {
  const greenIndex = getDataIndexByCursorPosition(greenCursorPos.value)
  const blueIndex = getDataIndexByCursorPosition(blueCursorPos.value)

  const time1 = result.ts[greenIndex]
  const time2 = result.ts[blueIndex]

  if (typeof time1 === 'undefined' || typeof time2 === 'undefined') {
    return
  }

  // 更新时间差
  state.timeDiff = time1 - time2

  const channels = result.chns
  const currentIndex = isBlue ? blueIndex : greenIndex
  state.valueColor = isBlue ? 'blue' : 'green'

  const valueArr = state.formatter!.getValueDataByIndex(currentIndex, false)

  for (let i = 0; i < channels.length; i++) {
    const channel = channels[i]
    if (!channel || !channel.analyse) continue

    const currentValue = GetCurrentValue(valueArr[i]!)

    // 查找并更新通道值
    for (let j = 0; j < channelsValue.length; j++) {
      const channelValue = channelsValue[j]
      if (channelValue && channelValue.index === i) {
        channelValue.rms = currentValue.a
        channelValue.instant = currentValue.b
        break
      }
    }
  }
}

// ==================== 缩放控制 ====================

/**
 * 水平缩放控制
 * @param shouldZoomIn true=放大, false=缩小
 */
function horizontalZoom(shouldZoomIn: boolean): void {
  if (shouldZoomIn) {
    if (state.pix < MAX_ZOOM_LEVEL) {
      if (state.pixneg > 1) {
        state.pixneg--
      } else {
        state.pix++
      }
    }
  } else {
    if (state.pix > MIN_ZOOM_LEVEL) {
      if (state.pix > 1) {
        state.pix--
      } else {
        state.pixneg++
      }
    }
  }
  loadWaveData(waveData as WaveDataType)
}

// /**
//  * 垂直缩放控制
//  * @param step 步长（正数=放大, 负数=缩小）
//  */
// function verticalZoom(step: number): void {
//   if (step > 0) {
//     if (state.gap < MAX_CHANNEL_GAP) {
//       state.gap += step
//     }
//   } else {
//     if (state.gap > MIN_CHANNEL_GAP) {
//       state.gap += step
//     }
//   }
//   loadWaveData(waveData as WaveDataType)
// }

/**
 * 还原波形到默认状态
 */
const reStore = (): void => {
  state.pixneg = 1
  state.pix = 1
  state.gap = DEFAULT_CHANNEL_GAP
  state.pagestart = 0
  state.pagelast = state.stack[0] || 0
  state.stack = []
  loadWaveData(waveData as WaveDataType)
}
</script>

<style scoped>
.waveViewer {
  position: relative;
  width: 100%;
  height: 100%;
  user-select: none;
  background-color: #000;
}

.header-container {
  z-index: 886;
  position: absolute;
  color: #fff;
}

.wave-container {
  z-index: 887;
  position: absolute;
  top: 50px;
  overflow-y: auto;
  overflow-x: hidden;
}

.cursor-line-container {
  z-index: 888;
  position: absolute;
  width: 20px;
  float: left;
  display: block;
}

.option-buttons {
  z-index: 900;
  display: flex;
  gap: 10px;
  position: absolute;
  right: 15px;
  top: 0;
}

.cursor-line {
  position: absolute;
  height: 100%;
  width: 2px;
  left: 10px;
  float: left;
}
.blue-line {
  background: blue;
}
.green-line {
  background: green;
}

.header-title {
  position: absolute;
  color: #4ae3ed;
  display: flex;
  gap: 20px;
  top: 0px;
}
.timeLabel {
  position: absolute;
  font-size: 11px;
  color: #78d1dc;
}
.channel-info {
  font-size: 0.7968vw;
}
</style>
