// ==================== 类型定义 ====================
export interface ChannelData {
  name: string
  uu: string // 单位
  y: number[]
  a: number
  b: number
  skew: number
  ps: string
  ptct: number
  analyse: number
  cursor?: number
  cursor1?: number
}

export interface SampleInfo {
  samp: number
  endsamp: number
}

export interface AllSelector {
  AD: string // 'A' 或 'V'
  phase: string // 'A', 'B', 'C', 'N'
}

export interface WaveDataType {
  chns: ChannelData[]
  ts: number[]
  beginTime: string
  sampleInfo: SampleInfo[]
  allSelector: AllSelector[]
}

export interface ValueData {
  name: string
  index: number
  valueStr: string
  valueSsz: number
  valueCyz: number
}

export interface MeasPhasor {
  m_dVra: number
  m_dVia: number
  m_dVrp: number
  m_dVip: number
  m_nZeroValue: number
  RMSP(): number
  RMSA(): number
  AngleP(): number
  AngleA(): number
  ToXYP(rms: number, angle: number): void
  ToXYA(rms: number, angle: number): void
}

// ==================== ValueFormatter 类 ====================
export class ValueFormatter {
  private _data: WaveDataType
  private _index: number = 0

  constructor(data: WaveDataType) {
    this._data = data
  }

  getValueDataByIndex(index: number, bOneValue: boolean, harmonic: number = 1): ValueData[] {
    const TN = this._getTN(index)
    return this._computeRms(TN, bOneValue, harmonic, index)
  }

  private _getTN(index: number): number {
    let TN = 0
    const sampList = [...this._data.sampleInfo]
    this._index = index

    for (let k = 0; k < sampList.length; k++) {
      const currSamp = sampList[k]
      if (!currSamp || currSamp.endsamp === undefined) {
        continue
      }
      if (index < currSamp.endsamp) {
        if (currSamp.endsamp - index < TN) {
          this._index = currSamp.endsamp - TN
        }
        TN = currSamp.samp / 50
        if (TN <= 0) {
          TN = 1
        }
        break
      }
    }

    return TN
  }

  private _computeRms(
    TN: number,
    bOneValue: boolean,
    harmonic: number,
    index: number,
  ): ValueData[] {
    const sliceNum = Math.ceil(TN + TN / 2)
    const channelArr = [...this._data.chns]
    const rel: ValueData[] = []

    for (let i = 0; i < channelArr.length; i++) {
      if (!channelArr[i]) {
        channelArr.splice(i, 1)
        continue
      }

      const channel = channelArr[i]
      if (!channel || channel.analyse === 0) {
        continue
      }

      const dataArr: number[] = []
      for (let j = 0; j < sliceNum; j++) {
        const endIndex = Math.min(channel.y.length - 1, this._index + j)
        dataArr[j] = channel.y[endIndex] * channel.a + channel.b
      }

      const ftabc = { fir: 0, sec: 0 }

      if (TN === 1) {
        if (harmonic === 1) {
          ftabc.fir = dataArr[0]
          ftabc.sec = dataArr[1]
        }
      } else {
        const m = TN >> 1
        for (let k = 0; k < TN; k++) {
          ftabc.fir += dataArr[k] * Math.sin((k * harmonic * Math.PI) / m)
          ftabc.sec += dataArr[k] * Math.cos((k * harmonic * Math.PI) / m)
        }
        ftabc.fir /= m
        ftabc.sec /= m
      }

      const phasor = this._createMeasPhasor()
      phasor.m_dVra = ftabc.fir
      phasor.m_dVia = ftabc.sec

      let bUnit = false
      const uuStr = channel.uu.toLowerCase()
      if (uuStr.indexOf('k') !== -1 && channel.a < 1) {
        bUnit = true
      }

      let dTempM = phasor.RMSA()
      let dTempA = phasor.AngleA()
      dTempA -= (channel.skew * 2.0 * Math.PI) / 20000

      const bAlreadOne = channel.ps.toLowerCase().indexOf('p') !== -1
      if (bOneValue) {
        if (!bAlreadOne) {
          dTempM *= channel.ptct
        }
        if (!bUnit) dTempM /= 1000.0
      } else {
        if (bAlreadOne) {
          dTempM /= channel.ptct
        }
      }

      if (isNaN(dTempM)) {
        dTempM = 0
      }

      if (isNaN(dTempA)) {
        dTempA = 0
      }

      phasor.ToXYA(dTempM, dTempA)

      const reObj: ValueData = {
        name: channel.name,
        index: i,
        valueStr:
          Math.sqrt((Math.pow(phasor.m_dVia, 2) + Math.pow(phasor.m_dVra, 2)) / 2).toFixed(3) +
          (bOneValue ? 'k' : '') +
          channel.uu,
        valueSsz: channel.y[index] ? channel.y[index] * channel.a + channel.b : 0,
        valueCyz: channel.y[index],
      }

      rel.push(reObj)
    }

    return rel
  }

  private _createMeasPhasor(): MeasPhasor {
    return {
      m_dVra: 0,
      m_dVia: 0,
      m_dVrp: 0,
      m_dVip: 0,
      m_nZeroValue: 0,
      RMSP(): number {
        return Math.sqrt((this.m_dVrp * this.m_dVrp + this.m_dVip * this.m_dVip) / 2)
      },
      RMSA(): number {
        return Math.sqrt((this.m_dVra * this.m_dVra + this.m_dVia * this.m_dVia) / 2)
      },
      AngleP(): number {
        if (this.m_dVrp === 0.0 && this.m_dVip === 0.0) {
          return 0.0
        }
        return Math.atan2(this.m_dVip, this.m_dVrp)
      },
      AngleA(): number {
        if (this.m_dVra === 0.0 && this.m_dVia === 0.0) {
          return 0.0
        }
        return Math.atan2(this.m_dVia, this.m_dVra)
      },
      ToXYP(rms: number, angle: number): void {
        this.m_dVrp = rms * Math.sqrt(2.0) * Math.cos(angle)
        this.m_dVip = rms * Math.sqrt(2.0) * Math.sin(angle)
      },
      ToXYA(rms: number, angle: number): void {
        this.m_dVra = rms * Math.sqrt(2.0) * Math.cos(angle)
        this.m_dVia = rms * Math.sqrt(2.0) * Math.sin(angle)
      },
    }
  }
}
