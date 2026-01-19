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
            top: state.ymargin - state.lenghtyMargin - 30 + 'px',
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
          top: state.ymargin + state.lenghtyMargin + 'px',
          left: blueCursorPos - 10 + 'px',
          height: state.canvasH - state.ymargin - state.lenghtyMargin + 'px',
        }"
      >
        <div class="cursor-line blue-line"></div>
      </div>

      <!-- 绿色游标 -->
      <div
        ref="greenCursor"
        class="cursor-line-container"
        :style="{
          top: state.ymargin + state.lenghtyMargin + 'px',
          left: greenCursorPos - 10 + 'px',
          height: state.canvasH - state.ymargin - state.lenghtyMargin + 'px',
        }"
      >
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

interface Props {
  fileDirectory: string
  fileName: string
  randomTime: string
}

const props = withDefaults(defineProps<Props>(), {
  fileDirectory: '',
  fileName: '',
  randomTime: '',
})

// ==================== Refs ====================
const waveCanvas = ref<HTMLCanvasElement | null>(null)
const rulerCanvas = ref<HTMLCanvasElement | null>(null)
const waveCanvasContainer = ref<HTMLDivElement | null>(null)
const blueCursor = ref<HTMLDivElement | null>(null)
const greenCursor = ref<HTMLDivElement | null>(null)

// ==================== 响应式数据 ====================
const waveData = reactive<Partial<WaveDataType>>({})

const state = reactive({
  context: null as CanvasRenderingContext2D | null,
  rulerContext: null as CanvasRenderingContext2D | null,
  formatter: null as ValueFormatter | null,
  valueArr: [] as ValueData[],

  // 尺寸相关
  canvasW: 0,
  canvasH: 0,
  rulerH: 0,
  xmargin: 15, // 设置x边距
  ymargin: 50, // 设置y边距
  tspixel: 8, // 设置时间的像素间隔
  gap: 90, // 通道之间的间隔
  gapcp: 90,
  lenghtyMargin: 20, //刻度跟y边距之间的距离

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
  valueColor: '#fff',
  beginTime: '',
  canvasDivH: 0,
  stack: [] as number[],
})

const timestampLabels = reactive(
  [] as {
    label: string
    index: number
    offset: number
  }[],
)

type ChannelInfo = {
  index: number
  name: string
  color: string
  offset: number
  rms?: number // 有效值 root mean square
  instant?: number // 瞬时值 instantaneous value
}

const channelsInfo = reactive([] as ChannelInfo[])
const channelsValue = reactive([] as ChannelInfo[])

const blueCursorPos = ref<number>(0)
const greenCursorPos = ref<number>(0)

// 根据游标位置获取数据索引
function getIndexByCursorPosition(cur: number): number {
  let pos = 0
  if (state.pix > 1) {
    pos = state.pagestart + Math.floor((cur - state.xmargin) / state.pix)
  } else {
    pos = state.pagestart + (cur - state.xmargin) * state.pixneg
  }
  return pos
}

// ==================== 数据获取 ====================
async function getData() {
  try {
    const { data: res } = await getWaveCanvas('/tmp', 'file')
    console.log(res)
    if (res.flag) {
      loadWaveData(res.result)
    }
  } catch (error) {
    console.error('Failed to load wave data:', error)
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
function init(): void {
  if (!waveCanvas.value || !rulerCanvas.value || !waveCanvasContainer.value) {
    console.error('Canvas or container element not found')
    return
  }

  const parentElement = waveCanvasContainer.value.parentElement
  const grandParentElement = parentElement?.parentElement

  state.canvasW = grandParentElement?.offsetWidth || 0
  state.canvasH = (grandParentElement?.offsetHeight || 0) - 20

  console.log('Window Width:', state.canvasW, 'Window Height:', state.canvasH)

  state.rulerH = state.canvasH
  state.gapcp = state.gap

  waveCanvas.value.width = state.canvasW
  rulerCanvas.value.width = state.canvasW
  rulerCanvas.value.height = state.rulerH

  waveCanvasContainer.value.style.width = state.canvasW + 'px'
  waveCanvasContainer.value.style.height = state.canvasH - 70 + 'px'

  state.context = waveCanvas.value.getContext('2d')
  state.rulerContext = rulerCanvas.value.getContext('2d')
}

// ==================== 事件监听设置 ====================
const setupEventListeners = (): void => {
  if (!waveCanvasContainer.value || !blueCursor.value || !greenCursor.value) return

  // 点击波形容器设置蓝色游标
  waveCanvasContainer.value.addEventListener('click', (event: MouseEvent) => {
    const parentRect = waveCanvasContainer.value!.parentElement!.getBoundingClientRect()
    blueCursorPos.value = Math.floor(event.clientX - parentRect.left)
    drawvalue(waveData as WaveDataType, true)
  })

  // 蓝色游标拖动
  setupCursorDrag(blueCursor.value, true)

  // 绿色游标拖动
  setupCursorDrag(greenCursor.value, false)

  // 初始化游标位置
  blueCursorPos.value = 50
  greenCursorPos.value = 250
}

// ==================== 游标拖动功能 ====================
const setupCursorDrag = (cursorElement: HTMLDivElement, isBlue: boolean): void => {
  cursorElement.addEventListener('mousedown', () => {
    let isMoving = true
    drawvalue(waveData as WaveDataType, isBlue)

    const handleMouseMove = (event: MouseEvent) => {
      if (!isMoving) return

      const parentRect = waveCanvasContainer.value!.parentElement!.getBoundingClientRect()
      const newPos = Math.floor(event.clientX - parentRect.left)

      if (isBlue) {
        blueCursorPos.value = newPos
      } else {
        greenCursorPos.value = newPos
      }

      drawvalue(waveData as WaveDataType, isBlue)
    }

    const handleMouseUp = () => {
      isMoving = false
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  })
}

// ==================== 数据加载 ====================
const loadWaveData = (result: WaveDataType): void => {
  Object.assign(waveData, result)

  if (result.chns && result.chns.length > 0) {
    const chns = result.chns
    let height = state.gap

    for (let j = 0; j < chns.length; j++) {
      height = state.gap * (j + 2)
    }

    state.formatter = new ValueFormatter(result)
    state.canvasH = height + state.ymargin

    if (waveCanvas.value) {
      waveCanvas.value.height = state.canvasH
    }

    if (getIndexByCursorPosition(blueCursorPos.value) >= result.ts.length) {
      blueCursorPos.value = state.xmargin
    }
    if (getIndexByCursorPosition(greenCursorPos.value) >= result.ts.length) {
      greenCursorPos.value = state.xmargin + 200
    }
    const time1 = result.ts[getIndexByCursorPosition(blueCursorPos.value)]
    const time2 = result.ts[getIndexByCursorPosition(greenCursorPos.value)]
    if (typeof time1 === 'number' && typeof time2 === 'number') {
      state.timeDiff = time2 - time1
    } else {
      state.timeDiff = 0
    }
    state.beginTime = result.beginTime.slice(0, result.beginTime.length - 3)

    parameter(result)
    rulerParameter(result)
  }
}

// ==================== 参数设置 ====================
const parameter = (result: WaveDataType): void => {
  if (!state.context) return

  state.context.clearRect(state.xmargin, state.ymargin, state.canvasW, state.canvasH)
  drawwave(result, state.context!)
}

const rulerParameter = (result: WaveDataType): void => {
  if (!state.rulerContext) return

  state.rulerContext.clearRect(0, 0, state.canvasW, state.rulerH)
  drawTimestampTick(state.rulerContext!, state.canvasW, state.rulerH)
  caculateTimestampLabels(result)
}

const drawwave = (result: WaveDataType, cont: CanvasRenderingContext2D): void => {
  const chns = result.chns
  const allSelector = result.allSelector
  const maxsv: number[] = [] // 电压最大值数组
  const maxsi: number[] = [] // 电流最大值数组
  let max = 0
  let boo = true

  if (chns.length >= 4) {
    boo = false
    for (let i = 0; i < chns.length; i++) {
      const chn = chns[i]
      if (!chn) continue
      if (chn.uu === 'V' && Array.isArray(chn.y)) {
        maxsv.push(Math.max(...chn.y))
      } else if (chn.uu === 'A' && Array.isArray(chn.y)) {
        maxsi.push(Math.max(...chn.y))
      } else {
        max = 10
      }
    }
  }

  const maxi = Math.max(...maxsi, 0)
  const maxv = Math.max(...maxsv, 0)

  let curs = blueCursorPos.value
  if (state.valueColor === 'green') curs = greenCursorPos.value

  state.valueArr = state.formatter!.getValueDataByIndex(getIndexByCursorPosition(curs), false)

  channelsInfo.splice(0, channelsInfo.length)
  channelsValue.splice(0, channelsValue.length)
  for (let i = 0; i < chns.length; i++) {
    const bad = (state.gapcp - state.gap) / state.gapcp
    let line = i * state.gap + state.ymargin + 65 * bad
    if (line < state.ymargin + 100) {
      line = i * 100 + state.ymargin
    }

    let valueObj: ValueData | undefined = undefined
    for (let k = 0; k < state.valueArr.length; k++) {
      if (state.valueArr[k]?.index === i) {
        valueObj = state.valueArr[k]
      }
    }

    let channelColor = ''
    if (allSelector[i]?.AD === 'A') {
      if (allSelector[i]?.phase === 'A') {
        channelColor = 'rgb(255,255,0)'
      } else if (allSelector[i]?.phase === 'B') {
        channelColor = 'rgb(0,255,0)'
      } else if (allSelector[i]?.phase === 'C') {
        channelColor = 'rgb(255,0,0)'
      } else if (allSelector[i]?.phase === 'N') {
        channelColor = 'rgb(0,255,255)'
      }
    } else {
      channelColor = 'rgb(255,128,0)'
    }

    const channel = chns[i]
    if (typeof channel === 'undefined') {
      continue
    }

    const channelInfo: ChannelInfo = {
      index: i,
      name: channel.name,
      color: channelColor,
      offset: line - 18,
    }
    channelsInfo.push(channelInfo)

    const channelValue: ChannelInfo = {
      index: i,
      name: channel.name,
      color: channelColor,
      offset: line - 18,
    }
    channelsValue.push(channelValue)
    if (valueObj) {
      const value = GetCurrentValue(valueObj)

      channelValue.rms = value.a
      channelValue.instant = value.b
    }

    drawwavecan(cont, state.xmargin, line + 1, state.canvasW - state.xmargin, line + 1, '#fff')
    drawwavecan(cont, state.xmargin, line - 15, state.canvasW - state.xmargin, line - 15, '#fff')
    drawwavecanvas(
      cont,
      state.xmargin,
      line + state.gap / 2 - 7,
      state.canvasW - 5,
      line + state.gap / 2 - 7,
    )

    const ypoint = channel.y
    if (boo) {
      max = Math.max(...ypoint)
    }

    if (channel.uu === 'V') {
      max = maxv
    } else if (channel.uu === 'A') {
      max = maxi
    } else {
      max = Math.max(...ypoint) * 2
    }

    cont.beginPath()
    cont.lineWidth = 1
    cont.strokeStyle = channelColor

    const yvalue = line + state.gap / 2 - 7
    const ratio = state.gap / 3

    cont.moveTo(state.xmargin, yvalue)

    let k = 0
    for (let j = state.pagestart; j < ypoint.length; j++) {
      const currentY = ypoint[j]
      const nextY = ypoint[j + 1]
      if (currentY === undefined || nextY === undefined) continue
      if (j % state.pixneg === 0) {
        let endy = yvalue
        // const y = yvalue

        if (nextY) {
          // y = yvalue + (currentY / max) * ratio
          endy = yvalue - (nextY / max) * ratio
        } else if (currentY) {
          // y = yvalue + (currentY / max) * ratio
        }

        if (k <= state.canvasW - state.xmargin * 2) {
          cont.lineTo(k + 1 + state.xmargin, endy)
          state.pagelast = j
        }
        k += state.pix
      }
    }
    cont.stroke()
    cont.closePath()
  }
}

// ==================== 绘制虚线 ====================
const drawwavecanvas = (
  cont: CanvasRenderingContext2D,
  x: number,
  y: number,
  endx: number,
  endy: number,
): void => {
  cont.beginPath()
  cont.lineWidth = 1
  cont.strokeStyle = '#bbb'
  for (let i = 0; i < endx - state.xmargin * 2; i++) {
    if (i % (state.tspixel * 2) === 0) {
      cont.moveTo(x + i, y)
      cont.lineTo(x + i + state.tspixel, endy)
      cont.stroke()
    }
  }
  cont.closePath()
}

// ==================== 绘制线条 ====================
const drawwavecan = (
  cont: CanvasRenderingContext2D,
  x: number,
  y: number,
  endx: number,
  endy: number,
  color: string,
  lineWidth: number = 1,
): void => {
  cont.beginPath()
  cont.lineWidth = lineWidth
  cont.strokeStyle = color
  cont.moveTo(x, y)
  cont.lineTo(endx, endy)
  cont.stroke()
  cont.closePath()
}

// 计算时间轴标签
function caculateTimestampLabels(result: WaveDataType): void {
  const ts = result.ts
  let k = 0
  let tf = true

  timestampLabels.splice(0, timestampLabels.length)
  for (let j = state.pagestart; j < ts.length; j++) {
    if (ts[j + 1]) {
      if (j % 100 === 0) {
        tf = false
        if (k <= state.canvasW - state.xmargin * 2) {
          if (j % (state.pixneg * 100) === 0) {
            tf = true
            const currentTime = ts[j]
            if (typeof currentTime === 'number') {
              const time = currentTime / 1000
              timestampLabels.push({
                label: time.toString() + 'ms',
                index: j,
                offset: k,
              })
            }
          }
        }
      }

      if (tf) {
        if (k === blueCursorPos.value) {
          blueCursorPos.value = blueCursorPos.value === 0 ? j + 150 : blueCursorPos.value
          greenCursorPos.value = greenCursorPos.value === 0 ? j + 300 : greenCursorPos.value
        }
        if (k === blueCursorPos.value + 10) {
          state.cursoradd = j
        }
        if (k === blueCursorPos.value - 10) {
          state.cursorsub = j
        }
        if (k === greenCursorPos.value + 10) {
          state.cursoradd1 = j
        }
        if (k === greenCursorPos.value - 10) {
          state.cursorsub1 = j
        }
        k += state.pix
      }
    }
  }
}

// 绘制时间刻度
function drawTimestampTick(cont: CanvasRenderingContext2D, x: number, y: number): void {
  cont.beginPath()
  cont.lineWidth = 2
  cont.strokeStyle = '#006686'
  cont.moveTo(state.xmargin, state.ymargin)
  cont.lineTo(state.xmargin, y)
  cont.lineTo(x - state.xmargin, y)
  cont.lineTo(x - state.xmargin, state.ymargin)
  cont.stroke()
  cont.moveTo(state.xmargin, state.ymargin + state.lenghtyMargin)
  cont.lineTo(x - state.xmargin, state.ymargin + state.lenghtyMargin)
  cont.stroke()
  cont.closePath()

  for (let i = state.xmargin; i < x - state.xmargin; i++) {
    if (i % 5 === 0) {
      if (i % state.tspixel === 0) {
        mscoordinates(cont, i, state.ymargin + state.lenghtyMargin, 8)
      }
    } else {
      if (i % state.tspixel === 0) {
        mscoordinates(cont, i, state.ymargin + state.lenghtyMargin, 4)
      }
    }
  }
}

// ==================== 绘制毫秒坐标 ====================
const mscoordinates = (
  cont: CanvasRenderingContext2D,
  x: number,
  y: number,
  value: number,
): void => {
  cont.beginPath()
  cont.lineWidth = 2
  cont.strokeStyle = '#006686'
  cont.moveTo(x, y)
  cont.lineTo(x, y - value)
  cont.stroke()
  cont.closePath()
}

// ==================== 移动游标重新绘制有效值 ====================
const drawvalue = (result: WaveDataType, boo: boolean): void => {
  const greenIndex = getIndexByCursorPosition(greenCursorPos.value)
  const blueIndex = getIndexByCursorPosition(blueCursorPos.value)
  const time1 = result.ts[greenIndex]
  const time2 = result.ts[blueIndex]
  if (typeof time1 === 'undefined' || typeof time2 === 'undefined') {
    return
  }

  state.timeDiff = time1 - time2

  const chns = result.chns
  let index = blueIndex
  state.valueColor = 'blue'
  if (!boo) {
    index = greenIndex
    state.valueColor = 'green'
  }

  const valueArr = state.formatter!.getValueDataByIndex(index, false)
  for (let i = 0; i < chns.length; i++) {
    const channel = chns[i]
    if (typeof channel === 'undefined') {
      continue
    }

    if (channel.analyse) {
      const currentValue = GetCurrentValue(valueArr[i]!)

      // find the same channel in channelsValue to update its rms and instant values
      for (let j = 0; j < channelsValue.length; j++) {
        const channelValue = channelsValue[j]
        if (typeof channelValue === 'undefined') {
          continue
        }
        if (channelValue.index === i) {
          channelValue.rms = currentValue.a
          channelValue.instant = currentValue.b
        }
      }
    }
  }
}

// ==================== 缩放控制 ====================
const horizontalZoom = (flag: boolean): void => {
  if (flag) {
    if (state.pix < 5) {
      if (state.pixneg > 1) {
        state.pixneg--
      } else {
        state.pix++
      }
    }
  } else {
    if (state.pix > -5) {
      if (state.pix > 1) {
        state.pix--
      } else {
        state.pixneg++
      }
    }
  }
  loadWaveData(waveData as WaveDataType)
}

const verticalZoom = (step: number): void => {
  if (step > 0) {
    if (state.gap < 150) state.gap += step
  } else {
    if (state.gap > 70) state.gap += step
  }
  loadWaveData(waveData as WaveDataType)
}

const reStore = (): void => {
  state.pixneg = 1
  state.pix = 1
  state.gap = 90
  state.pagestart = 0
  state.pagelast = state.stack[0] || 0
  state.stack = []
  loadWaveData(waveData as WaveDataType)
}

// 导出公开方法（可选）
defineExpose({
  horizontalZoom,
  verticalZoom,
  reStore,
  loadWaveData,
})
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
