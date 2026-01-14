<template>
  <div class="waveViewer">
    <div class="option-buttons" v-show="waveData.chns && waveData.chns.length > 0">
      <button @click="horizontalZoom(true)" type="button">水平放大</button>
      <button @click="horizontalZoom(false)" type="button">水平缩小</button>
      <!-- <button @click="verticalZoom(10)" type="button">垂直放大</button>
      <button @click="verticalZoom(-10)" type="button">垂直缩小</button> -->
      <button @click="reStore()" type="button">还原波形</button>
    </div>
    <div class="wave-view" v-show="waveData.chns && waveData.chns.length > 0">
      <!-- 蓝色游标 -->
      <div ref="blueCursor" class="cursor-line-container">
        <div id="blue" class="cursor-line" style="background: blue"></div>
      </div>

      <!-- 绿色游标 -->
      <div ref="greenCursor" class="cursor-line-container">
        <div id="green" class="cursor-line" style="background: green"></div>
      </div>

      <!-- 标尺容器 -->
      <div ref="rulerDiv" id="rulerDiv" class="ruler-container">
        <div id="rulerTime"></div>
        <div id="nonius"></div>
        <canvas ref="rulerCanvas" id="ruler"></canvas>
      </div>

      <!-- 波形容器 -->
      <div ref="waveCanvasContainer" id="waveDiv" class="wave-container">
        <div id="waveName"></div>
        <div id="waveValue"></div>
        <canvas ref="waveCanvas" id="waveContent"></canvas>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, nextTick } from 'vue'
import { getWaveCanvas } from '@/api'
import type { WaveDataType, ValueData } from '@/utils/comtrade.ts'
import { ValueFormatter } from '@/utils/comtrade.ts'

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
const rulerDiv = ref<HTMLDivElement | null>(null)

// ==================== 响应式数据 ====================
const waveData = reactive<Partial<WaveDataType>>({})

const state = reactive({
  context: null as CanvasRenderingContext2D | null,
  rulerContext: null as CanvasRenderingContext2D | null,
  formatter: null as ValueFormatter | null,
  valueArr: [] as ValueData[],

  // 尺寸相关
  winW: 0,
  winH: 0,
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
  cursor: 50, // 游标的位置
  cursor1: 250, // 第二个游标的位置
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

// ==================== 计算屏幕位置 ====================
const getPointPos = (cur: number): number => {
  let pos = 0
  if (state.pix > 1) {
    pos = state.pagestart + Math.floor((cur - state.xmargin) / state.pix)
  } else {
    pos = state.pagestart + (cur - state.xmargin) * state.pixneg
  }
  return pos
}

// ==================== 数据获取 ====================
const getData = async (): Promise<void> => {
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
watch(() => props.fileDirectory, getData)
watch(() => props.randomTime, getData)
watch(() => props.fileName, getData)

// ==================== 生命周期 ====================
onMounted(() => {
  nextTick(() => {
    init()
    setupEventListeners()
    getData()
  })
})

// ==================== 初始化 ====================
const init = (): void => {
  if (!waveCanvas.value || !rulerCanvas.value || !waveCanvasContainer.value) {
    console.error('Canvas or container element not found')
    return
  }

  const parentElement = waveCanvasContainer.value.parentElement
  const grandParentElement = parentElement?.parentElement

  state.winW = grandParentElement?.offsetWidth || 0
  state.winH = (grandParentElement?.offsetHeight || 0) - 20

  console.log('Window Width:', state.winW, 'Window Height:', state.winH)

  state.canvasW = state.winW
  state.canvasH = state.winH
  state.rulerH = state.canvasH
  state.gapcp = state.gap

  waveCanvas.value.width = state.canvasW
  rulerCanvas.value.width = state.canvasW
  rulerCanvas.value.height = state.rulerH

  waveCanvasContainer.value.style.width = state.winW + 'px'
  waveCanvasContainer.value.style.height = state.winH - 70 + 'px'

  state.context = waveCanvas.value.getContext('2d')
  state.rulerContext = rulerCanvas.value.getContext('2d')
}

// ==================== 事件监听设置 ====================
const setupEventListeners = (): void => {
  if (!waveCanvasContainer.value || !blueCursor.value || !greenCursor.value) return

  // 点击波形容器设置蓝色游标
  waveCanvasContainer.value.addEventListener('click', (event: MouseEvent) => {
    const parentRect = waveCanvasContainer.value!.parentElement!.getBoundingClientRect()
    state.cursor = Math.floor(event.clientX - parentRect.left)
    blueCursor.value!.style.left = state.cursor - 10 + 'px'
    drawvalue(waveData as WaveDataType, true)
  })

  // 蓝色游标拖动
  setupCursorDrag(blueCursor.value, true)

  // 绿色游标拖动
  setupCursorDrag(greenCursor.value, false)

  // 初始化游标位置
  blueCursor.value.style.top = state.ymargin + state.lenghtyMargin + 'px'
  blueCursor.value.style.left = state.cursor - 10 + 'px'
  blueCursor.value.style.height = state.canvasH - state.ymargin - state.lenghtyMargin + 'px'

  greenCursor.value.style.top = state.ymargin + state.lenghtyMargin + 'px'
  greenCursor.value.style.left = state.cursor1 - 10 + 'px'
  greenCursor.value.style.height = state.canvasH - state.ymargin - state.lenghtyMargin + 'px'
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
        state.cursor = newPos
        blueCursor.value!.style.left = newPos - 10 + 'px'
      } else {
        state.cursor1 = newPos
        greenCursor.value!.style.left = newPos - 10 + 'px'
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

    if (getPointPos(state.cursor) >= result.ts.length) {
      state.cursor = state.xmargin
    }
    if (getPointPos(state.cursor1) >= result.ts.length) {
      state.cursor1 = state.xmargin + 200
    }
    const time1 = result.ts[getPointPos(state.cursor)]
    const time2 = result.ts[getPointPos(state.cursor1)]
    if (typeof time1 === 'number' && typeof time2 === 'number') {
      state.timeDiff = time2 - time1
    } else {
      state.timeDiff = 0
    }
    state.beginTime = result.beginTime.slice(0, result.beginTime.length - 3)

    parameter(result)
    rulerParameter(result)

    blueCursor.value?.style.setProperty('display', 'block')
    greenCursor.value?.style.setProperty('display', 'block')
  }
}

// ==================== 参数设置 ====================
const parameter = (result: WaveDataType): void => {
  if (!state.context) return

  state.context.clearRect(state.xmargin, state.ymargin, state.canvasW, state.canvasH)
  startdraw(result)
}

const rulerParameter = (result: WaveDataType): void => {
  if (!state.rulerContext) return

  state.rulerContext.clearRect(0, 0, state.canvasW, state.rulerH)
  rulerStartdraw(result)
}

// ==================== 绘制方法 ====================
const rulerStartdraw = (result: WaveDataType): void => {
  outline(state.rulerContext!, state.canvasW, state.rulerH)
  legend(result, state.rulerContext!)
}

const startdraw = (result: WaveDataType): void => {
  drawwave(result, state.context!)
}

const drawwave = (result: WaveDataType, cont: CanvasRenderingContext2D): void => {
  const chns = result.chns
  const allSelector = result.allSelector
  const maxsv: number[] = []
  const maxsi: number[] = []
  let max = 0
  let boo = true

  if (chns.length >= 4) {
    boo = false
    for (let i = 0; i < chns.length; i++) {
      if (chns[i].uu === 'V') {
        maxsv.push(Math.max(...chns[i].y))
      } else if (chns[i].uu === 'A') {
        maxsi.push(Math.max(...chns[i].y))
      } else {
        max = 10
      }
    }
  }

  const maxi = Math.max(...maxsi, 0)
  const maxv = Math.max(...maxsv, 0)
  let strName = ''
  let aParam = ''
  if (state.winW < 380) aParam = 'width:150px;'

  let curs = state.cursor
  if (state.valueColor === 'green') curs = state.cursor1

  state.valueArr = state.formatter!.getValueDataByIndex(getPointPos(curs), false)

  let str = ''

  for (let i = 0; i < chns.length; i++) {
    const bad = (state.gapcp - state.gap) / state.gapcp
    let line = i * state.gap + state.ymargin + 65 * bad
    if (line < state.ymargin + 100) {
      line = i * 100 + state.ymargin
    }

    let valueObj: ValueData | undefined = undefined
    for (let k = 0; k < state.valueArr.length; k++) {
      if (state.valueArr[k].index === i) {
        valueObj = state.valueArr[k]
      }
    }

    let channelColor = ''
    if (allSelector[i].AD === 'A') {
      if (allSelector[i].phase === 'A') {
        channelColor = 'rgb(255,255,0)'
      } else if (allSelector[i].phase === 'B') {
        channelColor = 'rgb(0,255,0)'
      } else if (allSelector[i].phase === 'C') {
        channelColor = 'rgb(255,0,0)'
      } else if (allSelector[i].phase === 'N') {
        channelColor = 'rgb(0,255,255)'
      }
    } else {
      channelColor = 'rgb(255,128,0)'
    }

    strName +=
      "<div style='position:absolute;" +
      aParam +
      'top:' +
      (line - 18) +
      'px;left:' +
      (state.xmargin + 10) +
      'px;font-size: 0.7968vw;color:' +
      channelColor +
      "'>" +
      chns[i].name +
      '</div>'

    if (valueObj) {
      let a = valueObj.valueStr
      let b = valueObj.valueSsz
      const re = /([0-9]+\.[0-9]{2})[0-9]*/
      a = a.replace(re, '$1')
      b = parseFloat(b.toFixed(2))

      const value = ' 有效值:' + a + ' 瞬时值:' + b
      str +=
        "<div style='position:absolute;top:" +
        (line - 18) +
        "px;right:30px;font-size: 0.7968vw;color:#4ae3ed'>" +
        value +
        '</div>'
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

    const ypoint = chns[i].y
    if (boo) {
      max = Math.max(...ypoint)
    }

    if (chns[i].uu === 'V') {
      max = maxv
    } else if (chns[i].uu === 'A') {
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
      if (ypoint[j + 1] !== undefined && j % state.pixneg === 0) {
        let endy = yvalue
        let y = yvalue

        if (ypoint[j + 1]) {
          y = yvalue + (ypoint[j] / max) * ratio
          endy = yvalue - (ypoint[j + 1] / max) * ratio
        } else if (ypoint[j]) {
          y = yvalue + (ypoint[j] / max) * ratio
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

  const waveNameEl = document.getElementById('waveName')
  const waveValueEl = document.getElementById('waveValue')
  if (waveNameEl) waveNameEl.innerHTML = strName
  if (waveValueEl) waveValueEl.innerHTML = str

  if (waveCanvas.value) {
    state.canvasDivH = waveCanvas.value.offsetHeight
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

// ==================== 绘制图例 ====================
const legend = (result: WaveDataType, cont: CanvasRenderingContext2D): void => {
  const info =
    '录波文件名称：' +
    props.fileName +
    "<span style='margin-left: 20px;'>开始时间：" +
    state.beginTime +
    "</span><span style='margin-left: 20px;'>时标差：" +
    state.timeDiff / 1000 +
    'ms</span>'

  const str =
    "<div style='position:absolute;color: #4ae3ed;top:" +
    (state.ymargin - state.lenghtyMargin - 30) +
    'px;left:' +
    state.xmargin +
    "px;'>" +
    info +
    '</div>'

  const rulerTimeEl = document.getElementById('rulerTime')
  if (rulerTimeEl) rulerTimeEl.innerHTML = str

  const ts = result.ts
  let k = 0
  let tf = true
  let str2 = ''

  for (let j = state.pagestart; j < ts.length; j++) {
    if (ts[j + 1]) {
      if (j % 100 === 0) {
        tf = false
        if (k <= state.canvasW - state.xmargin * 2) {
          if (j % (state.pixneg * 100) === 0) {
            tf = true
            str2 +=
              "<div style='position:absolute;top:" +
              (state.ymargin - 18) +
              'px;left:' +
              k +
              "px;font-size: 11px;color: #78d1dc'>" +
              ts[j] / 1000 +
              'ms</div>'
          }
        }
      }

      if (tf) {
        if (k === state.cursor) {
          state.cursor = state.cursor === 0 ? j + 150 : state.cursor
          state.cursor1 = state.cursor1 === 0 ? j + 300 : state.cursor1
        }
        if (k === state.cursor + 10) {
          state.cursoradd = j
        }
        if (k === state.cursor - 10) {
          state.cursorsub = j
        }
        if (k === state.cursor1 + 10) {
          state.cursoradd1 = j
        }
        if (k === state.cursor1 - 10) {
          state.cursorsub1 = j
        }
        k += state.pix
      }
    }
  }

  const noniusEl = document.getElementById('nonius')
  if (noniusEl) noniusEl.innerHTML = str2
}

// ==================== 绘制轮廓 ====================
const outline = (cont: CanvasRenderingContext2D, x: number, y: number): void => {
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
  const timeDiff = result.ts[getPointPos(state.cursor1)] - result.ts[getPointPos(state.cursor)]
  const info =
    '录波文件名称：' +
    props.fileName +
    "<span style='margin-left: 20px;'>开始时间：" +
    state.beginTime +
    "</span><span style='margin-left: 20px;'>时标差：" +
    timeDiff / 1000 +
    'ms</span>'

  const str =
    "<div style='position:absolute; color:#4ae3ed;top:" +
    (state.ymargin - state.lenghtyMargin - 30) +
    'px;left:' +
    state.xmargin +
    "px;'>" +
    info +
    '</div>'

  const rulerTimeEl = document.getElementById('rulerTime')
  if (rulerTimeEl) rulerTimeEl.innerHTML = str

  const chns = result.chns
  let paramNum = state.cursor
  state.valueColor = 'blue'
  if (!boo) {
    paramNum = state.cursor1
    state.valueColor = 'green'
  }

  const valueArr = state.formatter!.getValueDataByIndex(getPointPos(paramNum), false)
  let str2 = ''

  for (let i = 0; i < chns.length; i++) {
    const bad = (state.gapcp - state.gap) / state.gapcp
    let line = i * state.gap + state.ymargin + 65 * bad
    if (line < state.ymargin + 100) {
      line = i * 100 + state.ymargin
    }

    if (chns[i].analyse) {
      let a = valueArr[i].valueStr
      let b = valueArr[i].valueSsz
      const re = /([0-9]+\.[0-9]{2})[0-9]*/

      a = a.replace(re, '$1')
      b = parseFloat(b.toFixed(2))

      const value = ' 有效值:' + a + ' 瞬时值:' + b
      str2 +=
        "<div style='position:absolute;top:" +
        (line - 18) +
        "px;right:30px;font-size: 0.7968vw;color:#4ae3ed'>" +
        value +
        '</div>'
    }
  }

  const waveValueEl = document.getElementById('waveValue')
  if (waveValueEl) waveValueEl.innerHTML = str2
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
}

.option-buttons {
  display: flex;
  gap: 10px;
  position: absolute;
  right: 15px;
  top: 0;
  z-index: 900;
}

.wave-view {
  position: relative;
  width: 100%;
  height: 100%;
  background-color: #000;
}

.cursor-line-container {
  z-index: 888;
  position: absolute;
  width: 20px;
  float: left;
  display: none;
}
.cursor-line {
  position: absolute;
  height: 100%;
  width: 2px;
  left: 10px;
  float: left;
}
.ruler-container {
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
</style>
