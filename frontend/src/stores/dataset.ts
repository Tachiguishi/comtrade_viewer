import { defineStore } from 'pinia'
import { listDatasets, importDataset, getMetadata, type DatasetInfo, type Metadata } from '../api'

export const useDatasetStore = defineStore('dataset', {
  state: () => ({
    datasets: [] as DatasetInfo[],
    currentId: '' as string,
    metadata: null as Metadata | null,
    loading: false as boolean,
    error: '' as string,
  }),
  actions: {
    async refreshList() {
      this.loading = true
      try {
        this.datasets = await listDatasets()
      } catch (e: any) {
        this.error = e?.message ?? String(e)
      } finally {
        this.loading = false
      }
    },
    async upload(cfg: File, dat: File) {
      const form = new FormData()
      form.append('cfg', cfg)
      form.append('dat', dat)
      const res = await importDataset(form)
      await this.refreshList()
      this.currentId = res.datasetId
      await this.loadMetadata(res.datasetId)
    },
    async loadMetadata(id: string) {
      this.loading = true
      try {
        this.metadata = await getMetadata(id)
        this.currentId = id
      } catch (e: any) {
        this.error = e?.message ?? String(e)
      } finally {
        this.loading = false
      }
    },
  },
})
