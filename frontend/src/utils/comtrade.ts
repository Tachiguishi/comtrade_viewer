// ==================== 常量定义 ====================

/** 电力系统标准频率 (Hz) */
const POWER_FREQUENCY = 50

/** 角度偏移时间换算常数 */
const SKEW_TIME_CONVERSION = 20000

/** 电压单位转换阈值 */
const KILO_UNIT_THRESHOLD = 1

/** 电压单位转换系数 */
const KILO_UNIT_DIVISOR = 1000.0

/** 根号2常数，用于RMS计算 */
const SQRT_2 = Math.sqrt(2.0)

/** 默认小数位数 */
const DEFAULT_DECIMAL_PLACES = 2

/** RMS值小数位数 */
const RMS_DECIMAL_PLACES = 3

// ==================== 类型定义 ====================

/**
 * 通道数据
 * 包含波形数据及其相关参数
 */
export type ChannelData = {
  /** 通道名称 */
  name: string
  /** 单位 */
  uu: string
  /** 采样点数据数组 */
  y: number[]
  /** 线性转换系数a (实际值 = y * a + b) */
  a: number
  /** 线性转换系数b */
  b: number
  /** 时间偏移(微秒) */
  skew: number
  /** 一次/二次标志 ('P'表示一次值, 'S'表示二次值) */
  ps: string
  /** PT/CT变比 */
  ptct: number
  /** 是否参与分析 (0=否, 1=是) */
  analyse: number
  /** 显示颜色 */
  color: string
  /** 可选的光标位置 */
  cursor?: number
  /** 可选的第二光标位置 */
  cursor1?: number
}

/**
 * 采样信息
 * 描述某段数据的采样率和结束位置
 */
export type SampleInfo = {
  /** 采样率 (samples/second) */
  samp: number
  /** 结束采样点索引 */
  endsamp: number
}

/**
 * 全选选择器
 * 用于筛选和分组通道
 */
export type AllSelector = {
  /** 信号类型: 'A'=模拟量, 'D'=数字量 */
  AD: string
  /** 相位: 'A', 'B', 'C', 'N' */
  phase: string
}

/**
 * 波形数据类型
 * COMTRADE文件的主要数据结构
 */
export type WaveDataType = {
  /** 所有通道数据 */
  chns: ChannelData[]
  /** 时间戳数组 */
  ts: number[]
  /** 记录开始时间 */
  beginTime: string
  /** 采样信息数组 */
  sampleInfo: SampleInfo[]
  /** 选择器数组 */
  allSelector: AllSelector[]
}

/**
 * 计算后的通道值数据
 */
export type ValueData = {
  /** 通道名称 */
  name: string
  /** 通道索引 */
  index: number
  /** 格式化后的RMS值字符串 */
  valueStr: string
  /** 实际值 (经过a*y+b转换) */
  valueSsz: number
  /** 原始采样值 */
  valueCyz: number
}

/**
 * 相量测量类
 * 用于电力系统相量计算，支持实部虚部和RMS/角度表示法
 */
export class MeasPhasor {
  /** 实部 (A表示法) */
  private _realA: number = 0
  /** 虚部 (A表示法) */
  private _imagA: number = 0
  /** 实部 (P表示法) */
  private _realP: number = 0
  /** 虚部 (P表示法) */
  private _imagP: number = 0
  /** 零值标志 */
  private _zeroValue: number = 0

  // Getter 和 Setter
  get realA(): number {
    return this._realA
  }
  set realA(value: number) {
    this._realA = value
  }

  get imagA(): number {
    return this._imagA
  }
  set imagA(value: number) {
    this._imagA = value
  }

  get realP(): number {
    return this._realP
  }
  set realP(value: number) {
    this._realP = value
  }

  get imagP(): number {
    return this._imagP
  }
  set imagP(value: number) {
    this._imagP = value
  }

  get zeroValue(): number {
    return this._zeroValue
  }
  set zeroValue(value: number) {
    this._zeroValue = value
  }

  /**
   * 计算P表示法的RMS值
   * @returns RMS值
   */
  getRMSP(): number {
    return Math.sqrt((this._realP * this._realP + this._imagP * this._imagP) / 2)
  }

  /**
   * 计算A表示法的RMS值
   * @returns RMS值
   */
  getRMSA(): number {
    return Math.sqrt((this._realA * this._realA + this._imagA * this._imagA) / 2)
  }

  /**
   * 计算P表示法的相角
   * @returns 相角 (弧度)
   */
  getAngleP(): number {
    if (this._realP === 0.0 && this._imagP === 0.0) {
      return 0.0
    }
    return Math.atan2(this._imagP, this._realP)
  }

  /**
   * 计算A表示法的相角
   * @returns 相角 (弧度)
   */
  getAngleA(): number {
    if (this._realA === 0.0 && this._imagA === 0.0) {
      return 0.0
    }
    return Math.atan2(this._imagA, this._realA)
  }

  /**
   * 从RMS和角度转换为P表示法的实部虚部
   * @param rms RMS值
   * @param angle 相角 (弧度)
   */
  setFromPolarP(rms: number, angle: number): void {
    this._realP = rms * SQRT_2 * Math.cos(angle)
    this._imagP = rms * SQRT_2 * Math.sin(angle)
  }

  /**
   * 从RMS和角度转换为A表示法的实部虚部
   * @param rms RMS值
   * @param angle 相角 (弧度)
   */
  setFromPolarA(rms: number, angle: number): void {
    this._realA = rms * SQRT_2 * Math.cos(angle)
    this._imagA = rms * SQRT_2 * Math.sin(angle)
  }
}

/**
 * 获取格式化的当前值
 * @param value 值数据
 * @returns 格式化后的数值对象 {a: 格式化后的字符串值, b: 格式化后的实际值}
 */
export function GetCurrentValue(value: ValueData): { a: string; b: number } {
  // const decimalPattern = /([0-9]+\.[0-9]{2})[0-9]*/

  // const formattedStr = value.valueStr.replace(decimalPattern, '$1')
  const formattedNum = parseFloat(value.valueSsz.toFixed(DEFAULT_DECIMAL_PLACES))

  return {
    a: value.valueStr,
    b: formattedNum,
  }
}

/**
 * 值格式化器
 * 用于计算和格式化COMTRADE波形数据的RMS值
 */
export class ValueFormatter {
  private readonly _data: WaveDataType
  private _currentIndex: number = 0

  constructor(data: WaveDataType) {
    this._data = data
  }

  /**
   * 根据索引获取值数据
   * @param index 数据索引
   * @param isPrimaryValue 是否为一次值
   * @param harmonic 谐波次数，默认为1
   * @returns 值数据数组
   */
  getValueDataByIndex(index: number, isPrimaryValue: boolean, harmonic: number = 1): ValueData[] {
    const samplesPerCycle = this._getSamplesPerCycle(index)
    return this._computeRMS(samplesPerCycle, isPrimaryValue, harmonic, index)
  }

  /**
   * 获取每周期采样点数
   * @param index 数据索引
   * @returns 每周期采样点数
   */
  private _getSamplesPerCycle(index: number): number {
    let samplesPerCycle = 0
    const sampleInfoList = [...this._data.sampleInfo]
    this._currentIndex = index

    for (const sampleInfo of sampleInfoList) {
      if (!sampleInfo || sampleInfo.endsamp === undefined) {
        continue
      }

      if (index < sampleInfo.endsamp) {
        // 如果剩余采样点不足一个周期，调整索引
        if (sampleInfo.endsamp - index < samplesPerCycle) {
          this._currentIndex = sampleInfo.endsamp - samplesPerCycle
        }

        // 计算每周期采样点数 (采样率 / 频率)
        samplesPerCycle = sampleInfo.samp / POWER_FREQUENCY
        if (samplesPerCycle <= 0) {
          samplesPerCycle = 1
        }
        break
      }
    }

    return samplesPerCycle
  }

  /**
   * 计算RMS值
   * @param samplesPerCycle 每周期采样点数
   * @param isPrimaryValue 是否为一次值
   * @param harmonic 谐波次数
   * @param index 数据索引
   * @returns 值数据数组
   */
  private _computeRMS(
    samplesPerCycle: number,
    isPrimaryValue: boolean,
    harmonic: number,
    index: number,
  ): ValueData[] {
    const sliceLength = Math.ceil(samplesPerCycle + samplesPerCycle / 2)
    const channels = [...this._data.chns]
    const results: ValueData[] = []

    for (let i = 0; i < channels.length; i++) {
      const channel = channels[i]
      if (!channel || channel.analyse === 0) {
        continue
      }

      // 提取数据切片
      const dataSlice = this._extractDataSlice(channel, sliceLength)

      // 进行傅里叶变换
      const fourierResult = this._performFourierTransform(dataSlice, samplesPerCycle, harmonic)

      // 创建相量对象
      const phasor = new MeasPhasor()
      phasor.realA = fourierResult.real
      phasor.imagA = fourierResult.imag

      // 计算并调整RMS和角度
      let rmsValue = phasor.getRMSA()
      let angleValue = phasor.getAngleA()

      // 应用时间偏移校正
      angleValue -= (channel.skew * 2.0 * Math.PI) / SKEW_TIME_CONVERSION

      // 应用单位转换
      const unitInfo = this._getUnitInfo(channel)
      rmsValue = this._applyUnitConversion(rmsValue, channel, isPrimaryValue, unitInfo)

      // 处理NaN值
      if (isNaN(rmsValue)) rmsValue = 0
      if (isNaN(angleValue)) angleValue = 0

      phasor.setFromPolarA(rmsValue, angleValue)

      // 创建结果对象
      const result = this._createValueData(channel, phasor, i, index, isPrimaryValue)
      results.push(result)
    }

    return results
  }

  /**
   * 提取数据切片并应用线性转换
   * @param channel 通道数据
   * @param length 切片长度
   * @returns 转换后的数据数组
   */
  private _extractDataSlice(channel: ChannelData, length: number): number[] {
    const dataSlice: number[] = []

    for (let j = 0; j < length; j++) {
      const index = Math.min(channel.y.length - 1, this._currentIndex + j)
      dataSlice[j] = channel.y[index]! * channel.a + channel.b
    }

    return dataSlice
  }

  /**
   * 执行离散傅里叶变换
   * @param dataSlice 数据切片
   * @param samplesPerCycle 每周期采样点数
   * @param harmonic 谐波次数
   * @returns 傅里叶变换结果 {real: 实部, imag: 虚部}
   */
  private _performFourierTransform(
    dataSlice: number[],
    samplesPerCycle: number,
    harmonic: number,
  ): { real: number; imag: number } {
    let real = 0
    let imag = 0

    if (samplesPerCycle === 1) {
      // 特殊情况：只有一个采样点
      if (harmonic === 1) {
        real = dataSlice[0]!
        imag = dataSlice[1]!
      }
    } else {
      // 标准DFT计算
      const halfSamples = samplesPerCycle >> 1

      for (let k = 0; k < samplesPerCycle; k++) {
        const angle = (k * harmonic * Math.PI) / halfSamples
        real += dataSlice[k]! * Math.sin(angle)
        imag += dataSlice[k]! * Math.cos(angle)
      }

      real /= halfSamples
      imag /= halfSamples
    }

    return { real, imag }
  }

  /**
   * 获取通道单位信息
   * @param channel 通道数据
   * @returns 单位信息
   */
  private _getUnitInfo(channel: ChannelData): { hasKiloUnit: boolean; unitStr: string } {
    const unitStr = channel.uu.toLowerCase()
    const hasKiloUnit = unitStr.indexOf('k') !== -1 && channel.a < KILO_UNIT_THRESHOLD

    return { hasKiloUnit, unitStr }
  }

  /**
   * 应用单位转换
   * @param rmsValue RMS值
   * @param channel 通道数据
   * @param isPrimaryValue 是否为一次值
   * @param unitInfo 单位信息
   * @returns 转换后的RMS值
   */
  private _applyUnitConversion(
    rmsValue: number,
    channel: ChannelData,
    isPrimaryValue: boolean,
    unitInfo: { hasKiloUnit: boolean },
  ): number {
    const isAlreadyPrimaryValue = channel.ps.toLowerCase().indexOf('p') !== -1

    if (isPrimaryValue) {
      // 如果当前不是一次值，需要乘以变比
      if (!isAlreadyPrimaryValue) {
        rmsValue *= channel.ptct
      }
      // 如果没有k单位，需要转换为k单位
      if (!unitInfo.hasKiloUnit) {
        rmsValue /= KILO_UNIT_DIVISOR
      }
    } else {
      // 如果当前是一次值，需要除以变比
      if (isAlreadyPrimaryValue) {
        rmsValue /= channel.ptct
      }
    }

    return rmsValue
  }

  /**
   * 创建值数据对象
   * @param channel 通道数据
   * @param phasor 相量对象
   * @param channelIndex 通道索引
   * @param dataIndex 数据索引
   * @param isPrimaryValue 是否为一次值
   * @returns 值数据对象
   */
  private _createValueData(
    channel: ChannelData,
    phasor: MeasPhasor,
    channelIndex: number,
    dataIndex: number,
    isPrimaryValue: boolean,
  ): ValueData {
    // 计算RMS值
    const rmsValue = Math.sqrt((Math.pow(phasor.imagA, 2) + Math.pow(phasor.realA, 2)) / 2)

    // 格式化值字符串
    const unitPrefix = isPrimaryValue ? 'k' : ''
    const valueStr = `${rmsValue.toFixed(RMS_DECIMAL_PLACES)}${unitPrefix}${channel.uu}`

    // 获取当前索引的实际值和原始值
    const actualValue = channel.y[dataIndex] ? channel.y[dataIndex] * channel.a + channel.b : 0
    const rawValue = channel.y[dataIndex] ?? 0

    return {
      name: channel.name,
      index: channelIndex,
      valueStr,
      valueSsz: actualValue,
      valueCyz: rawValue,
    }
  }
}
