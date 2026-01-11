import { defineStore } from 'pinia'

export const useViewStore = defineStore('view', {
  state: () => ({
    selectedAnalogChannels: [] as string[],
    selectedDigitalChannels: [] as string[],
    startMs: 0,
    endMs: 500,
  }),
  actions: {
    setWindow(start: number, end: number) {
      this.startMs = start
      this.endMs = end
    },
    toggleAnalogChannel(id: number) {
      const analogId = 'A' + id.toString()
      const idx = this.selectedAnalogChannels.indexOf(analogId)
      if (idx >= 0) this.selectedAnalogChannels.splice(idx, 1)
      else this.selectedAnalogChannels.push(analogId)
    },
    setAnalogChannels(ids: number[]) {
      this.selectedAnalogChannels = ids.map((id) => 'A' + id.toString())
    },
    toggleDigitalChannel(id: number) {
      const digitalId = 'D' + id.toString()
      const idx = this.selectedDigitalChannels.indexOf(digitalId)
      if (idx >= 0) this.selectedDigitalChannels.splice(idx, 1)
      else this.selectedDigitalChannels.push(digitalId)
    },
    setDigitalChannels(ids: number[]) {
      this.selectedDigitalChannels = ids.map((id) => 'D' + id.toString())
    },
  },
})
