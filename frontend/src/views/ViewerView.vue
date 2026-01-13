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
          <n-card title="通道选择" :bordered="false">
            <ChannelList />
          </n-card>
        </div>
      </n-layout-sider>
      <n-layout-content>
        <WaveformViewer />
      </n-layout-content>
    </n-layout>
  </div>
</template>

<script setup lang="ts">
import { NLayout, NLayoutSider, NLayoutContent, NCard } from 'naive-ui'
import WaveformViewer from '../components/WaveformViewer.vue'
import ChannelList from '../components/ChannelList.vue'
</script>

<style scoped>
.viewer-view {
  height: 100%;
  overflow: hidden;
}
.channel-panel {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
}
</style>
