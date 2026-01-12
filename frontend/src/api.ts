import axios from 'axios'

export const api = axios.create({ baseURL: '/api' })

export type DatasetInfo = { datasetId: string; name: string; createdAt: number; sizeBytes: number }
export type AnalogChannelMeta = {
  id: number
  name: string
  phase: string
  ccbm: string
  unit: string
  multiplier: number
  offset: number
  skew: number
  minValue: number
  maxValue: number
  primary: number
  secondary: number
  ps: string
}
export type DigitalChannelMeta = {
  id: number
  name: string
  phase: string
  ccbm: string
  y: number
}
export type Metadata = {
  station: string
  relay: string
  version: string
  totalChannelNum: number
  analogChannelNum: number
  digitalChannelNum: number
  analogChannels: AnalogChannelMeta[]
  digitalChannels: DigitalChannelMeta[]
  frequency: number
  ratesName: number
  sampleRates: { sampRate: number; lastSampleNum: number }[]
  startTime: string
  endTime: string
  dataFileType: string
  timeMultiplier: number
}

export async function listDatasets() {
  const { data } = await api.get<DatasetInfo[]>('/datasets')
  return data
}

export async function importDataset(form: FormData) {
  const { data } = await api.post('/datasets/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data as { datasetId: string; name: string }
}

export async function getMetadata(id: string) {
  const { data } = await api.get<Metadata>(`/datasets/${id}/metadata`)
  return data
}

export async function getWaveforms(id: string, channels: string[]) {
  const params = new URLSearchParams({
    channels: channels.join(','),
  })
  const { data } = await api.get(`/datasets/${id}/waveforms`, { params })
  return data as {
    series: { channel: string; name: string; unit: string; y: number[] }[]
    times: number[]
    window: { start: number; end: number }
  }
}
