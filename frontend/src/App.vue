<template>
  <div class="layout">
    <header class="topbar">
      <h1>ComTrade Viewer</h1>
    </header>
    <main class="content">
      <section class="sidebar">
        <UploadPane />
        <DatasetList />
      </section>
      <section class="viewer">
        <WaveformViewer />
      </section>
    </main>
  </div>
  </template>

<script setup lang="ts">
import UploadPane from './components/UploadPane.vue'
import DatasetList from './components/DatasetList.vue'
import WaveformViewer from './components/WaveformViewer.vue'
</script>

<style scoped>
.layout { display: flex; flex-direction: column; height: 100vh; }
.topbar { padding: 8px 12px; border-bottom: 1px solid #eee; }
.content { display: flex; flex: 1; }
.sidebar { width: 320px; border-right: 1px solid #eee; padding: 10px; overflow: auto; }
.viewer { flex: 1; padding: 10px; }
h1 { font-size: 18px; margin: 0; }
</style>
