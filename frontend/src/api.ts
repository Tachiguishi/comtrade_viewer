import axios from 'axios'

export const api = axios.create({ baseURL: '/api' })

export type DatasetInfo = { datasetId: string; name: string; createdAt: number; sizeBytes: number }
export type ChannelMeta = { id: string; name: string; type: 'analog' | 'digital'; unit?: string | null }
export type Metadata = {
  station: string
  recording: Record<string, any>
  sampling: { rate: number }
  channels: ChannelMeta[]
  timebase: number
  startTime: number
  endTime: number
}

export async function listDatasets() {
  const { data } = await api.get<DatasetInfo[]>('/datasets')
  return data
}

export async function importDataset(form: FormData) {
  const { data } = await api.post('/datasets/import', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return data as { datasetId: string; name: string }
}

export async function getMetadata(id: string) {
  const { data } = await api.get<Metadata>(`/datasets/${id}/metadata`)
  return data
}

export async function getWaveforms(id: string, channels: string[], startMs: number, endMs: number) {
  const params = new URLSearchParams({ channels: channels.join(','), start: String(startMs), end: String(endMs) })
  const { data } = await api.get(`/datasets/${id}/waveforms`, { params })
  return data as { series: { channelId: string; t: number[]; y: number[] }[]; window: { start: number; end: number } }
}
