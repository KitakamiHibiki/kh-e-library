<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getBook, getProgress, getSettings, updateBook } from '@/api'
import { useI18n } from 'vue-i18n'
import PdfReader from '@/components/PdfReader.vue'
import EpubReader from '@/components/EpubReader.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const getBookIdFromRoute = (): number => {
  const id = route.query.id
  return id ? Number(id) : 0
}

const bookId = ref(getBookIdFromRoute())
const loading = ref(true)
const loadError = ref(false)
const isPdf = ref(false)
const progressVal = ref(0)
const bookTitle = ref('')
const bookStatus = ref('')
const fontSize = ref(16)
let restoredCfi: string | null = null

// Refs to child components (for method calls only)
const pdfReaderRef = ref<InstanceType<typeof PdfReader> | null>(null)
const epubReaderRef = ref<InstanceType<typeof EpubReader> | null>(null)

// Synced state from child components (reactive in parent)
const pdfState = ref({
  currentPage: 1,
  totalPages: 0,
  zoomLevel: 100,
  pdfViewMode: 'single',
  doublePageDisplay: '1',
})
const epubHasToc = ref(false)
const epubState = ref({ epubViewMode: 'single' })

const initReader = async () => {
  if (!bookId.value || isNaN(bookId.value)) {
    loading.value = false
    ElMessage.error(t('reader.loadFailed'))
    return
  }
  try {
    const bookRes = await getBook(bookId.value)
    const bookData = bookRes.data.data
    bookTitle.value = bookData.title || ''
    bookStatus.value = bookData.book_status || ''
    isPdf.value = bookData.file_type === 'pdf'

    if (bookStatus.value !== 'ready') {
      loading.value = false
      return
    }

    // Mark as reading if currently unread
    if (bookData.read_status === 'unread') {
      updateBook(bookId.value, { read_status: 'reading' }).catch(() => {})
    }

    // Load reader settings
    try {
      const settingsRes = await getSettings()
      const data = settingsRes.data.data
      const fs = data?.['reader.font_size']
      if (fs) fontSize.value = parseInt(fs)
    } catch { /* ignore */ }

    const progRes = await getProgress(bookId.value)
    restoredCfi = progRes.data.data?.cfi || null
    if (progRes.data.data) progressVal.value = progRes.data.data.progress || 0

    loading.value = false
  } catch {
    loading.value = false
    loadError.value = true
    ElMessage.error(t('reader.loadFailed'))
  }
}

// Watch route query changes
watch(() => route.query.id, (newId) => {
  const newBookId = newId ? Number(newId) : 0
  if (newBookId && newBookId !== bookId.value) {
    loading.value = true
    loadError.value = false
    bookTitle.value = ''
    bookStatus.value = ''
    isPdf.value = false
    progressVal.value = 0
    restoredCfi = null
    bookId.value = newBookId
    initReader()
  }
})

const onProgressUpdate = (progress: number) => {
  progressVal.value = progress
}

const onPdfStateChange = (state: { currentPage: number; totalPages: number; zoomLevel: number; pdfViewMode: string; doublePageDisplay: string }) => {
  pdfState.value = state
}

const onEpubTocLoaded = (items: any[]) => {
  epubHasToc.value = items.length > 0
}

const onEpubStateChange = (state: { epubViewMode: string }) => {
  epubState.value = state
}

onMounted(() => {
  initReader()
})

const goBack = async () => {
  if (!isPdf.value) await epubReaderRef.value?.saveProgressNow()
  router.push('/')
}
</script>

<template>
  <div class="reader-container">
    <!-- Single unified toolbar -->
    <div class="reader-toolbar">
      <el-button size="small" @click="goBack">{{ t('reader.back') }}</el-button>
      <span class="book-title">{{ bookTitle }}</span>

      <!-- PDF controls in toolbar -->
      <template v-if="isPdf && pdfState.totalPages > 0">
        <el-select
          :model-value="pdfState.pdfViewMode"
          size="small"
          class="view-mode-select"
          @change="(val: any) => pdfReaderRef?.onViewModeChange(val)"
        >
          <el-option value="single" :label="t('reader.viewModeSingle')" />
          <el-option value="double" :label="t('reader.viewModeDouble')" />
          <el-option value="scroll" :label="t('reader.viewModeScroll')" />
        </el-select>
        <el-button size="small" @click="pdfReaderRef?.zoomOut()" :disabled="pdfState.zoomLevel <= 25">{{ t('reader.zoomOut') }}</el-button>
        <span class="zoom-label">{{ pdfState.zoomLevel }}%</span>
        <el-button size="small" @click="pdfReaderRef?.zoomIn()" :disabled="pdfState.zoomLevel >= 300">{{ t('reader.zoomIn') }}</el-button>
        <el-button v-if="pdfState.zoomLevel !== 100" size="small" @click="pdfReaderRef?.resetZoom()">{{ t('reader.fit') }}</el-button>
        <span class="toolbar-divider"></span>
        <span v-if="pdfState.pdfViewMode === 'double' && pdfState.currentPage !== 1" class="progress-text">{{ pdfState.doublePageDisplay }} / {{ pdfState.totalPages }} {{ Math.round(((pdfState.currentPage - 1) / pdfState.totalPages) * 100) }}%</span>
        <span v-else class="progress-text">{{ t('reader.page', { current: pdfState.currentPage, total: pdfState.totalPages, percent: Math.round(((pdfState.currentPage - 1) / pdfState.totalPages) * 100) }) }}</span>
      </template>

      <!-- EPUB controls in toolbar -->
      <template v-if="!isPdf">
        <el-select
          :model-value="epubState.epubViewMode"
          size="small"
          class="view-mode-select"
          @change="(val: any) => epubReaderRef?.onViewModeChange(val)"
        >
          <el-option value="single" :label="t('reader.viewModeSingle')" />
          <el-option value="double" :label="t('reader.viewModeDouble')" />
        </el-select>
        <el-button v-if="epubHasToc" size="small" @click="epubReaderRef?.toggleToc()">{{ t('reader.toc') }}</el-button>
        <span class="progress-text">{{ t('reader.progressPercent', { percent: Math.round(progressVal * 100) }) }}</span>
      </template>
    </div>

    <div class="reader-content">
      <!-- Not ready state -->
      <div v-if="bookStatus && bookStatus !== 'ready'" class="not-ready">
        <el-result v-if="bookStatus === 'processing'" :title="t('reader.processing')" icon="info" />
        <el-result v-else-if="bookStatus === 'failed'" :title="t('reader.failed')" icon="error" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="loading-state">
        <span>{{ t('reader.loading') }}</span>
      </div>

      <!-- Load error -->
      <div v-if="loadError" class="not-ready">
        <el-result :title="t('reader.loadFailed')" icon="error" />
      </div>

      <!-- EPUB reader -->
      <EpubReader
        v-if="!isPdf && !loading && bookStatus === 'ready'"
        ref="epubReaderRef"
        :book-id="bookId"
        :restored-cfi="restoredCfi"
        :font-size="fontSize"
        :initial-progress="progressVal"
        @progress-update="onProgressUpdate"
        @toc-loaded="onEpubTocLoaded"
        @state-change="onEpubStateChange"
      />

      <!-- PDF reader -->
      <PdfReader
        v-if="isPdf && !loading && bookStatus === 'ready'"
        ref="pdfReaderRef"
        :book-id="bookId"
        :restored-cfi="restoredCfi"
        @progress-update="onProgressUpdate"
        @state-change="onPdfStateChange"
      />
    </div>
  </div>
</template>

<style scoped>
.reader-container {
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-primary, #fff);
  color: var(--text-primary, #303133);
}
.reader-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 12px;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  z-index: 10;
  gap: 8px;
  flex-shrink: 0;
}
.book-title {
  font-size: 0.9rem;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.view-mode-select {
  width: 120px;
}
.zoom-label {
  font-size: 0.85rem;
  min-width: 40px;
  text-align: center;
}
.toolbar-divider {
  width: 1px;
  height: 16px;
  background: var(--border-color, #e0e0e0);
  margin: 0 4px;
}
.progress-text {
  font-size: 0.8rem;
  color: var(--text-secondary, #666);
}
.reader-content {
  flex: 1;
  min-height: 0;
  position: relative;
  overflow: hidden;
}
.loading-state, .not-ready {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}
</style>
