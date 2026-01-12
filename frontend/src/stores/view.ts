import { defineStore } from 'pinia'

export const useViewStore = defineStore('view', {
  state: () => ({
    selectedAnalogChannels: [] as string[],
    selectedDigitalChannels: [] as string[],
    station: '',
    relay: '',
    version: '',
    startTime: '',
    endTime: '',
  }),
  actions: {
    setMetaData(station: string, relay: string, version: string) {
      this.station = station
      this.relay = relay
      this.version = version
    },
    setTimeRange(start: string, end: string) {
      this.startTime = start
      this.endTime = end
    },
    clearChannelSelection() {
      this.selectedAnalogChannels = []
      this.selectedDigitalChannels = []
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
