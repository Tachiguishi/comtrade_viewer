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
    <button @click="handleUpload" :disabled="!canUpload || datasetStore.loading">
      {{ datasetStore.loading ? 'Importing...' : 'Import Dataset' }}
    </button>
    <div v-if="error" class="error">{{ error }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDatasetStore } from '../stores/dataset'

const datasetStore = useDatasetStore()
const cfgFile = ref<File | null>(null)
const datFile = ref<File | null>(null)
const error = ref('')

const canUpload = computed(() => cfgFile.value && datFile.value)

async function handleUpload() {
  if (!cfgFile.value || !datFile.value) return
  error.value = ''
  try {
    await datasetStore.upload(cfgFile.value, datFile.value)
    // Clear inputs after success
    cfgFile.value = null
    datFile.value = null
    // Reset file inputs visually if needed, simplistic here
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Upload failed'
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
</style>
