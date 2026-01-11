import { defineStore } from 'pinia'

export const useViewStore = defineStore('view', {
  state: () => ({
    selectedAnalogChannels: [] as number[],
    selectedDigitalChannels: [] as number[],
    startMs: 0,
    endMs: 500,
  }),
  actions: {
    setWindow(start: number, end: number) {
      this.startMs = start
      this.endMs = end
    },
    toggleAnalogChannel(id: number) {
      const idx = this.selectedAnalogChannels.indexOf(id)
      if (idx >= 0) this.selectedAnalogChannels.splice(idx, 1)
      else this.selectedAnalogChannels.push(id)
    },
    setAnalogChannels(ids: number[]) {
      this.selectedAnalogChannels = ids
    },
    toggleDigitalChannel(id: number) {
      const idx = this.selectedDigitalChannels.indexOf(id)
      if (idx >= 0) this.selectedDigitalChannels.splice(idx, 1)
      else this.selectedDigitalChannels.push(id)
    },
    setDigitalChannels(ids: number[]) {
      this.selectedDigitalChannels = ids
    },
  },
})
