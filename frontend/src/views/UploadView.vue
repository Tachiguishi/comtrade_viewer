<template>
  <div class="upload-view">
    <n-card title="导入录波文件" :bordered="false">
      <n-space vertical>
        <UploadPane />
      </n-space>
    </n-card>
    <n-card
      title="录波文件列表"
      :bordered="false"
      style="margin-top: 16px; flex: 1; overflow: hidden"
    >
      <DatasetList />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import UploadPane from '../components/UploadPane.vue'
import DatasetList from '../components/DatasetList.vue'
</script>

<style scoped>
.upload-view {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
