import axios, { AxiosError } from 'axios'

export const api = axios.create({ baseURL: '/api' })
export const waveCanvasApi = axios.create({ baseURL: '/fault' })

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
  ratesNum: number
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

export async function getWaveforms(
  id: string,
  analogChannels: number[],
  digitalChannels: number[],
) {
  const params = new URLSearchParams({
    A: analogChannels.join(','),
    D: digitalChannels.join(','),
  })
  const { data } = await api.get(`/datasets/${id}/waveforms`, { params })
  return data as {
    series: {
      channel: number
      type: string
      name: string
      unit: string
      times: number[]
      y: number[]
    }[]
    times: number[]
    window: { start: number; end: number }
    downsample: { method: string; targetPoints: number; originalPoints: number }
  }
}

export async function getWaveCanvas(fileDirectory: string, fileName: string) {
  const params = new URLSearchParams({
    fileDirectory: fileDirectory,
    fileName: fileName,
  })
  const data = await waveCanvasApi.get(`/previewRCDTest`, { params })
  return data
}

// --- Error utilities & upload with progress ---
export type ApiError = { error?: { code: string; message: string; details?: unknown } } | string

export function extractApiError(e: unknown): { message: string; details?: unknown } {
  // Axios error shape handling
  const ax = e as AxiosError
  const data = ax?.response?.data as ApiError | undefined
  if (data && typeof data === 'object' && 'error' in data && data.error) {
    return { message: data.error.message, details: data.error.details }
  }
  if (typeof data === 'string') {
    return { message: data }
  }
  const msg = ax?.message || (e instanceof Error ? e.message : '请求失败')
  return { message: msg }
}

export async function importDatasetWithProgress(
  form: FormData,
  onProgress?: (pct: number) => void,
) {
  const { data } = await api.post('/datasets/import', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (evt) => {
      if (!onProgress) return
      const total = evt.total || 0
      const loaded = evt.loaded || 0
      const pct = total > 0 ? Math.round((loaded / total) * 100) : 0
      onProgress(pct)
    },
  })
  return data as { datasetId: string; name: string }
}
