import { defineStore } from 'pinia'

export const useViewStore = defineStore('view', {
  state: () => ({
    selectedChannels: [] as string[],
    startMs: 0,
    endMs: 500,
  }),
  actions: {
    setWindow(start: number, end: number) {
      this.startMs = start
      this.endMs = end
    },
    toggleChannel(id: string) {
      const idx = this.selectedChannels.indexOf(id)
      if (idx >= 0) this.selectedChannels.splice(idx, 1)
      else this.selectedChannels.push(id)
    },
    setChannels(ids: string[]) {
      this.selectedChannels = ids
    },
  },
})
