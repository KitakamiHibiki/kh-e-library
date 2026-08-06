<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getReadUrl, saveProgress, getSettings, updateSettings } from '@/api'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  bookId: number
  restoredCfi: string | null
}>()

const emit = defineEmits<{
  (e: 'progressUpdate', progress: number): void
  (e: 'stateChange', state: { currentPage: number; totalPages: number; zoomLevel: number; pdfViewMode: string; doublePageDisplay: string }): void
}>()

const { t } = useI18n()

const currentPage = ref(1)
const totalPages = ref(0)
const pdfCanvas = ref<HTMLCanvasElement | null>(null)
const pdfCanvasLeft = ref<HTMLCanvasElement | null>(null)
const pdfCanvasRight = ref<HTMLCanvasElement | null>(null)

const zoomLevel = ref(100)
const pdfViewMode = ref<'single' | 'double' | 'scroll'>('single')

let pdfDoc: any = null
let saveTimer: ReturnType<typeof setTimeout> | null = null
let wheelTimer: ReturnType<typeof setTimeout> | null = null
let isRendering = false
let scrollObserver: IntersectionObserver | null = null
const scrollCanvasRefs = ref<Record<number, HTMLCanvasElement | null>>({})

const doubleLeftPage = computed(() => {
  if (currentPage.value === 1) return 0
  const left = currentPage.value % 2 === 0 ? currentPage.value : currentPage.value - 1
  return left
})
const doubleRightPage = computed(() => {
  if (currentPage.value === 1) return 1
  const left = doubleLeftPage.value
  const right = left + 1
  return right > totalPages.value ? 0 : right
})
const doublePageDisplay = computed(() => {
  if (currentPage.value === 1) return '1'
  const left = doubleLeftPage.value
  const right = doubleRightPage.value
  return right ? `${left}-${right}` : `${left}`
})

const emitState = () => {
  emit('stateChange', {
    currentPage: currentPage.value,
    totalPages: totalPages.value,
    zoomLevel: zoomLevel.value,
    pdfViewMode: pdfViewMode.value,
    doublePageDisplay: doublePageDisplay.value,
  })
}

const cleanup = () => {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
  if (wheelTimer) { clearTimeout(wheelTimer); wheelTimer = null }
  if (scrollObserver) { scrollObserver.disconnect(); scrollObserver = null }
  if (pdfDoc) { pdfDoc.destroy(); pdfDoc = null }
  pdfDoc = null
  totalPages.value = 0
  currentPage.value = 1
  zoomLevel.value = 100
  scrollCanvasRefs.value = {}
  isRendering = false
}

const init = async () => {
  try {
    try {
      const settingsRes = await getSettings()
      const vm = settingsRes.data.data?.['reader.pdf_view_mode']
      if (vm === 'single' || vm === 'double' || vm === 'scroll') {
        pdfViewMode.value = vm
      }
    } catch { /* ignore */ }

    const pdfjsLib = await import('pdfjs-dist')
    pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
      'pdfjs-dist/build/pdf.worker.min.mjs',
      import.meta.url
    ).href

    const doc = await (pdfjsLib as any).getDocument(getReadUrl(props.bookId)).promise
    pdfDoc = doc
    totalPages.value = doc.numPages

    const startPage = props.restoredCfi ? Math.min(parseInt(props.restoredCfi), doc.numPages) : 1
    currentPage.value = Math.max(startPage, 1)

    emitState()
    await nextTick()
    await renderCurrentView()
  } catch {
    ElMessage.error(t('reader.loadFailed'))
  }
}

const renderPageToCanvas = async (pageNum: number, canvas: HTMLCanvasElement, isDouble = false) => {
  if (!pdfDoc || pageNum < 1 || pageNum > totalPages.value) return
  const page = await pdfDoc.getPage(pageNum)
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const vp = page.getViewport({ scale: 1 })
  let scale: number
  if (isDouble) {
    const availWidth = (window.innerWidth - 40) / 2
    const availHeight = window.innerHeight - 100
    const widthScale = availWidth / vp.width
    const heightScale = availHeight / vp.height
    scale = Math.min(widthScale, heightScale) * zoomLevel.value / 100
  } else {
    const availHeight = window.innerHeight - 100
    const fitScale = availHeight / vp.height
    scale = fitScale * zoomLevel.value / 100
  }

  const svp = page.getViewport({ scale })
  canvas.width = svp.width
  canvas.height = svp.height
  await page.render({ canvasContext: ctx, viewport: svp }).promise
}

const renderCurrentView = async () => {
  if (!pdfDoc) return
  if (isRendering) return
  isRendering = true
  try {
    if (pdfViewMode.value === 'single') {
      const canvas = pdfCanvas.value
      if (canvas) await renderPageToCanvas(currentPage.value, canvas)
    } else if (pdfViewMode.value === 'double') {
      if (currentPage.value === 1) {
        const rightCanvas = pdfCanvasRight.value
        if (rightCanvas) await renderPageToCanvas(1, rightCanvas, false)
      } else {
        const leftCanvas = pdfCanvasLeft.value
        const rightCanvas = pdfCanvasRight.value
        if (leftCanvas && doubleLeftPage.value) await renderPageToCanvas(doubleLeftPage.value, leftCanvas, true)
        if (rightCanvas && doubleRightPage.value) await renderPageToCanvas(doubleRightPage.value, rightCanvas, true)
        else if (rightCanvas) { rightCanvas.width = 0; rightCanvas.height = 0 }
      }
    } else if (pdfViewMode.value === 'scroll') {
      await renderScrollPages()
    }

    const prog = (currentPage.value - 1) / totalPages.value
    emit('progressUpdate', prog)
    emitState()
    scheduleSave(String(currentPage.value), prog)
  } finally {
    isRendering = false
  }
}

const renderScrollPages = async () => {
  if (!pdfDoc) return
  for (let i = 1; i <= totalPages.value; i++) {
    const canvas = scrollCanvasRefs.value[i]
    if (canvas) await renderPageToCanvas(i, canvas)
  }
}

const setupScrollObserver = () => {
  if (scrollObserver) { scrollObserver.disconnect(); scrollObserver = null }
  scrollObserver = new IntersectionObserver((entries) => {
    let topPage = currentPage.value
    let topY = Infinity
    for (const entry of entries) {
      if (entry.isIntersecting) {
        const pageNum = parseInt((entry.target as HTMLElement).dataset.page || '0')
        if (pageNum && entry.boundingClientRect.top < topY) {
          topY = entry.boundingClientRect.top
          topPage = pageNum
        }
      }
    }
    if (topPage !== currentPage.value) {
      currentPage.value = topPage
      const prog = (topPage - 1) / totalPages.value
      emit('progressUpdate', prog)
      emitState()
      scheduleSave(String(topPage), prog)
    }
  }, { root: null, threshold: 0.5 })

  nextTick(() => {
    const containers = document.querySelectorAll('.scroll-page-container')
    containers.forEach(el => scrollObserver?.observe(el))
  })
}

const prevPage = () => {
  if (!pdfDoc || currentPage.value <= 1) return
  if (pdfViewMode.value === 'double') {
    if (currentPage.value === 1) return
    const prevEven = currentPage.value - 2
    currentPage.value = prevEven < 2 ? 1 : prevEven
  } else {
    currentPage.value--
  }
  nextTick(() => renderCurrentView())
}

const nextPage = () => {
  if (!pdfDoc || currentPage.value >= totalPages.value) return
  if (pdfViewMode.value === 'double') {
    if (currentPage.value === 1) {
      currentPage.value = 2
    } else {
      const nextEven = currentPage.value + 2
      currentPage.value = nextEven > totalPages.value ? totalPages.value : nextEven
    }
  } else {
    currentPage.value++
  }
  nextTick(() => renderCurrentView())
}

const zoomIn = () => { zoomLevel.value = Math.min(300, zoomLevel.value + 10); renderCurrentView() }
const zoomOut = () => { zoomLevel.value = Math.max(25, zoomLevel.value - 10); renderCurrentView() }
const resetZoom = () => { zoomLevel.value = 100; renderCurrentView() }

const goToPage = () => {
  if (!pdfDoc) return
  let p = currentPage.value
  if (p < 1) p = 1
  if (p > totalPages.value) p = totalPages.value
  currentPage.value = p
  renderCurrentView()
}

const handleWheel = (e: WheelEvent) => {
  if (!pdfDoc) return
  if (e.ctrlKey) {
    e.preventDefault()
    if (e.deltaY < 0) zoomIn()
    else if (e.deltaY > 0) zoomOut()
    return
  }
  if (pdfViewMode.value === 'scroll') return
  e.preventDefault()
  if (wheelTimer) return
  if (e.deltaY > 0) nextPage()
  else if (e.deltaY < 0) prevPage()
  wheelTimer = setTimeout(() => { wheelTimer = null }, 300)
}

const onViewModeChange = (val: 'single' | 'double' | 'scroll') => {
  pdfViewMode.value = val
  updateSettings({ 'reader.pdf_view_mode': val }).catch(() => {})
  nextTick(async () => {
    await renderCurrentView()
    if (pdfViewMode.value === 'scroll') setupScrollObserver()
  })
}

const scheduleSave = (cfi: string, progress: number, href?: string) => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await saveProgress({ book_id: props.bookId, progress, cfi, chapter_href: href })
    } catch { /* ignore */ }
  }, 2000)
}

const handleKey = (e: KeyboardEvent) => {
  if (e.key === 'ArrowLeft') { prevPage(); e.preventDefault() }
  if (e.key === 'ArrowRight') { nextPage(); e.preventDefault() }
  if (e.ctrlKey && (e.key === '=' || e.key === '+')) { zoomIn(); e.preventDefault() }
  if (e.ctrlKey && e.key === '-') { zoomOut(); e.preventDefault() }
  if (e.ctrlKey && e.key === '0') { resetZoom(); e.preventDefault() }
}

onMounted(() => {
  document.addEventListener('keydown', handleKey)
  init()
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKey)
  cleanup()
})

defineExpose({
  prevPage,
  nextPage,
  goToPage,
  zoomIn,
  zoomOut,
  resetZoom,
  onViewModeChange,
})
</script>

<template>
  <div class="pdf-reader" @wheel="handleWheel">
    <div class="pdf-content">
      <!-- Single page mode -->
      <div v-if="pdfViewMode === 'single'" class="pdf-container">
        <canvas ref="pdfCanvas" class="pdf-canvas"></canvas>
      </div>

      <!-- Double page mode -->
      <div v-else-if="pdfViewMode === 'double'" class="pdf-container pdf-double">
        <div v-if="currentPage === 1" class="double-cover">
          <canvas ref="pdfCanvasRight" class="pdf-canvas"></canvas>
        </div>
        <div v-else class="double-pages">
          <canvas ref="pdfCanvasLeft" class="pdf-canvas"></canvas>
          <canvas v-if="doubleRightPage" ref="pdfCanvasRight" class="pdf-canvas"></canvas>
        </div>
      </div>

      <!-- Scroll mode -->
      <div v-else-if="pdfViewMode === 'scroll'" class="pdf-container pdf-scroll">
        <div
          v-for="page in totalPages"
          :key="page"
          :data-page="page"
          class="scroll-page-container"
        >
          <canvas
            :ref="(el: any) => { if (el) scrollCanvasRefs[page] = el as HTMLCanvasElement }"
            class="pdf-canvas"
          />
        </div>
      </div>
    </div>

    <!-- Bottom navigation -->
    <div v-if="pdfViewMode !== 'scroll' && totalPages > 0" class="pdf-controls">
      <el-button size="small" :disabled="currentPage <= 1" @click="prevPage">{{ t('reader.prev') }}</el-button>
      <el-input-number v-model="currentPage" :min="1" :max="totalPages" size="small" controls-position="right" class="page-input" @change="goToPage" />
      <span class="page-total">/ {{ totalPages }}</span>
      <el-button size="small" :disabled="currentPage >= totalPages" @click="nextPage">{{ t('reader.next') }}</el-button>
    </div>
  </div>
</template>

<style scoped>
.pdf-reader {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
}
.pdf-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.pdf-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: hidden;
  flex: 1;
  min-height: 0;
}
.pdf-scroll {
  overflow-y: auto;
  align-items: center;
  padding: 12px 0;
  gap: 8px;
}
.pdf-canvas {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}
.pdf-double .double-pages {
  display: flex;
  gap: 4px;
  justify-content: center;
  align-items: flex-start;
}
.pdf-double .double-cover {
  display: flex;
  justify-content: center;
  align-items: flex-start;
}
.scroll-page-container {
  display: flex;
  justify-content: center;
}
.pdf-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 4px 12px;
  border-top: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  flex-shrink: 0;
}
.page-input {
  width: 130px;
}
.page-total {
  font-size: 0.85rem;
}
</style>
