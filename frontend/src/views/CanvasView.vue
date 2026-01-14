<template>
  <div class="viewer-view">
    <WaveCanvas fileDirectory="" fileName="" randomTime="" />
  </div>
</template>

<script setup lang="ts">
import WaveCanvas from '@/components/WaveCanvas.vue'
</script>

<style scoped>
.viewer-view {
  height: 100%;
  overflow: hidden;
}
.channel-panel {
  padding: 0 16px;
  height: 100%;
  overflow-y: auto;
}
</style>
