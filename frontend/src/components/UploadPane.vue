<template>
  <div class="upload-pane">
    <h4>New Import</h4>
    <div class="file-inputs">
      <label>
        .cfg:
        <input
          type="file"
          accept=".cfg"
          @change="(e) => (cfgFile = (e.target as HTMLInputElement).files?.[0] || null)"
        />
      </label>
      <label>
        .dat:
        <input
          type="file"
          accept=".dat"
          @change="(e) => (datFile = (e.target as HTMLInputElement).files?.[0] || null)"
        />
      </label>
    </div>
    <button @click="handleUpload" :disabled="!canUpload || datasetStore.loading || validationError">
      {{ datasetStore.loading ? 'Importing...' : 'Import Dataset' }}
    </button>

    <div
      v-if="datasetStore.uploadProgress > 0 && datasetStore.uploadProgress < 100"
      class="progress"
    >
      <div class="bar" :style="{ width: datasetStore.uploadProgress + '%' }"></div>
      <span>{{ datasetStore.uploadProgress }}%</span>
    </div>

    <div v-if="validationError" class="error">{{ validationError }}</div>
    <div v-else-if="error" class="error">
      <div>{{ error }}</div>
      <pre v-if="datasetStore.errorDetails" class="details">{{
        formatDetails(datasetStore.errorDetails)
      }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDatasetStore } from '../stores/dataset'

const datasetStore = useDatasetStore()
const cfgFile = ref<File | null>(null)
const datFile = ref<File | null>(null)
const error = ref('')

const canUpload = computed(() => !!cfgFile.value && !!datFile.value)
const validationError = computed(() => {
  if (!cfgFile.value || !datFile.value) return ''
  const cfgOk = cfgFile.value.name.toLowerCase().endsWith('.cfg')
  const datOk = datFile.value.name.toLowerCase().endsWith('.dat')
  if (!cfgOk) return '请选择后缀为 .cfg 的配置文件'
  if (!datOk) return '请选择后缀为 .dat 的数据文件'
  if (cfgFile.value.size === 0) return '配置文件为空'
  if (datFile.value.size === 0) return '数据文件为空'
  return ''
})

async function handleUpload() {
  if (!cfgFile.value || !datFile.value) return
  if (validationError.value) {
    error.value = validationError.value
    return
  }
  error.value = ''
  try {
    await datasetStore.upload(cfgFile.value, datFile.value)
    // Clear inputs after success
    cfgFile.value = null
    datFile.value = null
    // Reset file inputs visually if needed, simplistic here
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '上传失败'
  }
}

function formatDetails(d: unknown): string {
  try {
    return typeof d === 'string' ? d : JSON.stringify(d, null, 2)
  } catch {
    return String(d)
  }
}
</script>

<style scoped>
.upload-pane {
  border-bottom: 1px solid #ddd;
  padding-bottom: 10px;
  margin-bottom: 10px;
}
.file-inputs {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 8px;
}
button {
  width: 100%;
  cursor: pointer;
}
.error {
  color: red;
  font-size: 12px;
  margin-top: 4px;
}
.progress {
  position: relative;
  height: 8px;
  background: #eee;
  border-radius: 4px;
  margin-top: 6px;
}
.progress .bar {
  height: 100%;
  background: #4caf50;
  width: 0;
  transition: width 0.2s ease;
  border-radius: 4px;
}
.progress span {
  display: inline-block;
  margin-top: 4px;
  font-size: 12px;
}
.details {
  margin-top: 6px;
  background: #f9f9f9;
  border: 1px solid #eee;
  padding: 6px;
  white-space: pre-wrap;
}
</style>
