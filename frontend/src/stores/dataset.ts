import { defineStore } from 'pinia'
import {
  listDatasets,
  importDatasetWithProgress,
  getMetadata,
  extractApiError,
  type DatasetInfo,
  type Metadata,
} from '../api'

export const useDatasetStore = defineStore('dataset', {
  state: () => ({
    datasets: [] as DatasetInfo[],
    currentId: '' as string,
    metadata: null as Metadata | null,
    loading: false as boolean,
    error: '' as string,
    uploadProgress: 0 as number,
    errorDetails: null as unknown,
  }),
  actions: {
    async refreshList() {
      this.loading = true
      try {
        this.datasets = await listDatasets()
      } catch (e: unknown) {
        this.error = e instanceof Error ? e.message : String(e)
      } finally {
        this.loading = false
      }
    },
    async upload(cfg: File, dat: File) {
      const form = new FormData()
      form.append('cfg', cfg)
      form.append('dat', dat)
      this.error = ''
      this.errorDetails = null
      this.uploadProgress = 0
      try {
        const res = await importDatasetWithProgress(form, (pct) => {
          this.uploadProgress = pct
        })
        await this.refreshList()
        this.currentId = res.datasetId
        await this.loadMetadata(res.datasetId)
      } catch (e: unknown) {
        const { message, details } = extractApiError(e)
        this.error = message
        this.errorDetails = details ?? null
        throw new Error(message)
      } finally {
        // small delay to show 100% if reached
        if (this.uploadProgress < 100) this.uploadProgress = 100
      }
    },
    async loadMetadata(id: string) {
      this.loading = true
      try {
        this.metadata = await getMetadata(id)
        this.currentId = id
      } catch (e: unknown) {
        const { message } = extractApiError(e)
        this.error = message
      } finally {
        this.loading = false
      }
    },
  },
})
