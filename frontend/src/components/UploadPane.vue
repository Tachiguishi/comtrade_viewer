<template>
  <div class="upload-pane">
    <n-space vertical :size="12">
      <n-form-item label="配置文件 (.cfg)">
        <n-upload
          :max="1"
          accept=".cfg"
          :file-list="cfgFileList"
          @update:file-list="handleCfgChange"
        >
          <n-button>选择 .cfg 文件</n-button>
        </n-upload>
      </n-form-item>

      <n-form-item label="数据文件 (.dat)">
        <n-upload
          :max="1"
          accept=".dat"
          :file-list="datFileList"
          @update:file-list="handleDatChange"
        >
          <n-button>选择 .dat 文件</n-button>
        </n-upload>
      </n-form-item>

      <n-button
        type="primary"
        block
        @click="handleUpload"
        :disabled="!canUpload || datasetStore.loading"
        :loading="datasetStore.loading"
      >
        {{ datasetStore.loading ? '导入中...' : '导入数据集' }}
      </n-button>

      <n-progress
        v-if="datasetStore.uploadProgress > 0 && datasetStore.uploadProgress < 100"
        type="line"
        :percentage="datasetStore.uploadProgress"
        :indicator-placement="'inside'"
      />

      <n-alert v-if="validationError" type="warning" :title="validationError" />
      <n-alert v-else-if="error" type="error" :title="error">
        <n-code
          v-if="datasetStore.errorDetails"
          :code="formatDetails(datasetStore.errorDetails)"
          language="json"
        />
      </n-alert>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  // NSpace,
  // NFormItem,
  // NUpload,
  // NButton,
  // NProgress,
  // NAlert,
  // NCode,
  type UploadFileInfo,
} from 'naive-ui'
import { useDatasetStore } from '../stores/dataset'

const datasetStore = useDatasetStore()
const cfgFileList = ref<UploadFileInfo[]>([])
const datFileList = ref<UploadFileInfo[]>([])
const error = ref('')

const cfgFile = computed(() => cfgFileList.value[0]?.file || null)
const datFile = computed(() => datFileList.value[0]?.file || null)

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

function handleCfgChange(fileList: UploadFileInfo[]) {
  cfgFileList.value = fileList
}

function handleDatChange(fileList: UploadFileInfo[]) {
  datFileList.value = fileList
}

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
    cfgFileList.value = []
    datFileList.value = []
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
  width: 100%;
}

.n-form-item {
  display: flex;
  gap: 25px;
  align-items: center;
}
.n-form-item .n-form-item-label {
  margin-right: 12px;
}
</style>
