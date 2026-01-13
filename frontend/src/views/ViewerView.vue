<template>
  <div class="viewer-view">
    <n-layout has-sider>
      <n-layout-sider
        bordered
        :width="380"
        :collapsed-width="0"
        :native-scrollbar="false"
        show-trigger
        collapse-mode="width"
      >
        <div class="channel-panel">
          <ChannelList />
        </div>
      </n-layout-sider>
      <n-layout-content>
        <WaveformViewer />
      </n-layout-content>
    </n-layout>
  </div>
</template>

<script setup lang="ts">
import { NLayout, NLayoutSider, NLayoutContent } from 'naive-ui'
import WaveformViewer from '../components/WaveformViewer.vue'
import ChannelList from '../components/ChannelList.vue'
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
