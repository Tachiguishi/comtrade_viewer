import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'

export const api = axios.create({ baseURL: '/api' })
export const waveCanvasApi = axios.create({ baseURL: '/fault' })

let authInterceptorsInstalled = false

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
export type WaveData = {
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
  timeRange: { start: number; end: number }
  downsample: { method: string; targetPoints: number; originalPoints: number }
}

type LoginRequest = { username: string; password: string }
export type LoginResponse = { token: string; expiresAt: number }

export function setupApiAuth(getToken: () => string | null, onUnauthorized?: () => void) {
  if (authInterceptorsInstalled) return
  authInterceptorsInstalled = true

  const injectAuth = (config: InternalAxiosRequestConfig) => {
    const token = getToken()
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  }

  const handleUnauthorized = (error: AxiosError) => {
    if (error.response?.status === 401 && onUnauthorized) {
      onUnauthorized()
    }
    return Promise.reject(error)
  }

  api.interceptors.request.use(injectAuth)
  api.interceptors.response.use((resp) => resp, handleUnauthorized)

  waveCanvasApi.interceptors.request.use(injectAuth)
  waveCanvasApi.interceptors.response.use((resp) => resp, handleUnauthorized)
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
  startTime?: number,
  endTime?: number,
) {
  const params = new URLSearchParams({
    A: analogChannels.join(','),
    D: digitalChannels.join(','),
  })
  if (startTime !== undefined) {
    params.set('startTime', String(startTime))
  }
  if (endTime !== undefined) {
    params.set('endTime', String(endTime))
  }
  const { data } = await api.get(`/datasets/${id}/waveforms`, { params })
  return data as WaveData
}

export async function login(payload: LoginRequest) {
  const { data } = await api.post('/auth/login', payload)
  return data as LoginResponse
}

export async function getWaveCanvas(id: string) {
  const { data } = await api.get(`/datasets/${id}/wavecanvas`)
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
