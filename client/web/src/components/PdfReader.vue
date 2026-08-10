<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick, computed } from 'vue'
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
  (e: 'completed'): void
}>()

const { t } = useI18n()

const currentPage = ref(1)
const totalPages = ref(0)
const pdfCanvas = ref<HTMLCanvasElement | null>(null)
const pdfCanvasLeft = ref<HTMLCanvasElement | null>(null)
const pdfCanvasRight = ref<HTMLCanvasElement | null>(null)

const zoomLevel = ref(100)
const pdfViewMode = ref<'single' | 'double' | 'scroll'>('single')
const loading = ref(true)

let pdfDoc: any = null
let saveTimer: ReturnType<typeof setTimeout> | null = null
let isRendering = false
let pendingWheelDir: 0 | 1 | -1 = 0
const scrollCanvasRefs = ref<Record<number, HTMLCanvasElement | null>>({})
const pdfScrollRef = ref<HTMLElement | null>(null)
let scrollListener: (() => void) | null = null
let completedFired = false
let suppressNextComplete = false
// Current scroll offset, captured from the live scroll container. Kept separate
// from reading el.scrollTop on deactivation because KeepAlive has already
// detached the element by then, and a detached element's scrollTop reads 0.
let lastScrollTop = 0
// Scroll position to restore when the component is reactivated (returning from
// the completion page) — the scroll container resets to 0 when KeepAlive
// detaches/reattaches its DOM.
let savedScrollTop = 0

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

// Flush an in-flight progress save so re-entering the book resumes here.
// Called both on real unmount and on KeepAlive deactivation.
const flushSave = () => {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
    if (totalPages.value > 0) {
      const prog = (currentPage.value - 1) / totalPages.value
      saveProgress({ book_id: props.bookId, progress: prog, cfi: String(currentPage.value) }).catch(() => {})
    }
  }
}

const cleanup = () => {
  detachScrollHandlers()
  flushSave()
  if (pdfDoc) { pdfDoc.destroy(); pdfDoc = null }
  pdfDoc = null
  totalPages.value = 0
  currentPage.value = 1
  zoomLevel.value = 100
  scrollCanvasRefs.value = {}
  isRendering = false
}

// Persist the current position immediately — called when jumping to the
// completion page so the final page is saved regardless of the debounce timer.
const saveProgressNow = async () => {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
  if (totalPages.value > 0) {
    const prog = (currentPage.value - 1) / totalPages.value
    try {
      await saveProgress({ book_id: props.bookId, progress: prog, cfi: String(currentPage.value) })
    } catch { /* ignore */ }
  }
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
    if (pdfViewMode.value === 'scroll') {
      setupScrollHandlers()
      restoreScrollPosition(currentPage.value)
    }
    // Only reveal the reader once the initial page is rendered and, in scroll
    // mode, positioned at the resume page — no flash of page 1 followed by a jump.
    loading.value = false
  } catch {
    loading.value = false
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
    // Process any wheel event that arrived while rendering
    if (pendingWheelDir !== 0) {
      const dir = pendingWheelDir
      pendingWheelDir = 0
      if (dir > 0) nextPage()
      else prevPage()
    }
  }
}

const renderScrollPages = async (from = 1, to = totalPages.value) => {
  if (!pdfDoc) return
  // Render pages with a small concurrency limit so large documents load
  // noticeably faster while the initial loading overlay is showing.
  let next = from
  const worker = async () => {
    while (next <= to) {
      const pageNum = next++
      const canvas = scrollCanvasRefs.value[pageNum]
      if (!canvas) continue
      try { await renderPageToCanvas(pageNum, canvas) } catch { /* keep going */ }
    }
  }
  const workers: Promise<void>[] = []
  for (let w = 0; w < 3; w++) workers.push(worker())
  await Promise.all(workers)
}

// ---- Scroll-mode navigation & completion ----

// Compute the current page from the scroll position of the scroll container.
// The current page is the last page whose top has scrolled up to (or past) the
// top edge of the visible scroll area.
const updateScrollPage = () => {
  const el = pdfScrollRef.value
  if (!el) return
  const elTop = el.getBoundingClientRect().top
  const containers = el.querySelectorAll('.scroll-page-container')
  let page = 1
  for (const node of containers) {
    const pageNum = parseInt((node as HTMLElement).dataset.page || '0')
    if (!pageNum) continue
    if (node.getBoundingClientRect().top - elTop <= 0) page = pageNum
    else break
  }
  if (page !== currentPage.value) {
    currentPage.value = page
    const prog = (page - 1) / totalPages.value
    emit('progressUpdate', prog)
    emitState()
    scheduleSave(String(page), prog)
  }
}

// Auto-complete once the scroll reaches the very bottom of the document.
const checkScrollBottom = () => {
  const el = pdfScrollRef.value
  if (!el || completedFired) return
  if (suppressNextComplete) { suppressNextComplete = false; return }
  const room = el.scrollHeight - el.clientHeight
  if (room <= 1) return // no real scrolling room — handled by key/wheel paths
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 1) {
    completedFired = true
    emit('completed')
  }
}

// Scroll the scroll container by one viewport height. Reaching the end while
// moving forward triggers the reading-completed event.
const scrollByStep = (dir: 1 | -1) => {
  const el = pdfScrollRef.value
  if (!el) return
  const room = el.scrollHeight - el.clientHeight
  const atBottom = room <= 1 || el.scrollTop + el.clientHeight >= el.scrollHeight - 1
  const atTop = el.scrollTop <= 1
  if (dir > 0 && atBottom) {
    if (!completedFired) { completedFired = true; emit('completed') }
    return
  }
  if (dir < 0 && atTop) return
  el.scrollBy({ top: dir * el.clientHeight, behavior: 'smooth' })
}

const handleScroll = () => {
  const el = pdfScrollRef.value
  if (el) lastScrollTop = el.scrollTop
  updateScrollPage()
  checkScrollBottom()
}

const setupScrollHandlers = () => {
  if (scrollListener) detachScrollHandlers()
  completedFired = false
  suppressNextComplete = false
  nextTick(() => {
    const el = pdfScrollRef.value
    if (!el) return
    scrollListener = handleScroll
    el.addEventListener('scroll', scrollListener)
  })
}

const detachScrollHandlers = () => {
  if (scrollListener && pdfScrollRef.value) {
    pdfScrollRef.value.removeEventListener('scroll', scrollListener)
  }
  scrollListener = null
}

// Align the scroll container so `page` sits at the top. When the resume
// position is the last page, suppress the auto-complete for this restore so the
// user isn't immediately bounced to the completion page.
const restoreScrollPosition = (page: number) => {
  const el = pdfScrollRef.value
  if (!el) return
  const target = el.querySelector(`.scroll-page-container[data-page="${page}"]`) as HTMLElement | null
  if (!target) return
  const elRect = el.getBoundingClientRect()
  const targetRect = target.getBoundingClientRect()
  el.scrollTop += targetRect.top - elRect.top
  lastScrollTop = el.scrollTop
  const room = el.scrollHeight - el.clientHeight
  if (room > 1 && el.scrollTop + el.clientHeight >= el.scrollHeight - 1) {
    suppressNextComplete = true
  }
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
  if (!pdfDoc) return
  // Already on the last page — treat "next" as reaching the end of the book.
  if (currentPage.value >= totalPages.value) {
    emit('completed')
    return
  }
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
  if (pdfViewMode.value === 'scroll') {
    // Native wheel scrolling drives the container; detect scrolling forward
    // past the bottom (incl. short documents with no scroll room) to complete.
    const el = pdfScrollRef.value
    if (el && e.deltaY > 0) {
      const room = el.scrollHeight - el.clientHeight
      const atBottom = room <= 1 || el.scrollTop + el.clientHeight >= el.scrollHeight - 1
      if (atBottom) {
        e.preventDefault()
        if (!completedFired) { completedFired = true; emit('completed') }
      }
    }
    return
  }
  e.preventDefault()
  if (isRendering) {
    // Store direction of the most recent wheel event while rendering
    if (e.deltaY > 0) pendingWheelDir = 1
    else if (e.deltaY < 0) pendingWheelDir = -1
    return
  }
  if (e.deltaY > 0) nextPage()
  else if (e.deltaY < 0) prevPage()
}

const onViewModeChange = (val: 'single' | 'double' | 'scroll') => {
  pdfViewMode.value = val
  updateSettings({ 'reader.pdf_view_mode': val }).catch(() => {})
  if (val !== 'scroll') detachScrollHandlers()
  nextTick(async () => {
    await renderCurrentView()
    if (pdfViewMode.value === 'scroll') {
      setupScrollHandlers()
      restoreScrollPosition(currentPage.value)
    }
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
  if (e.ctrlKey && (e.key === '=' || e.key === '+')) { zoomIn(); e.preventDefault() }
  if (e.ctrlKey && e.key === '-') { zoomOut(); e.preventDefault() }
  if (e.ctrlKey && e.key === '0') { resetZoom(); e.preventDefault() }
  // In scroll mode the arrow / page keys drive the scroll container directly.
  if (pdfViewMode.value === 'scroll') {
    if (e.key === 'ArrowDown' || e.key === 'ArrowRight' || e.key === 'PageDown' || e.key === ' ') {
      scrollByStep(1)
      e.preventDefault()
    } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft' || e.key === 'PageUp') {
      scrollByStep(-1)
      e.preventDefault()
    }
    return
  }
  if (e.key === 'ArrowLeft') { prevPage(); e.preventDefault() }
  if (e.key === 'ArrowRight') { nextPage(); e.preventDefault() }
}

onMounted(() => {
  document.addEventListener('keydown', handleKey)
  init()
})

// KeepAlive lifecycle: the reader stays mounted across the completion page, so
// the parsed document and rendered canvases survive — returning to /read is
// instant instead of re-processing the book. The scroll container's DOM is
// detached/re-attached, which resets scrollTop, so we save and restore it.
onActivated(() => {
  // Re-add idempotently (avoid double listeners after deactivation).
  document.removeEventListener('keydown', handleKey)
  document.addEventListener('keydown', handleKey)
  completedFired = false
  nextTick(() => {
    const el = pdfScrollRef.value
    if (pdfViewMode.value !== 'scroll' || !el) return
    // Always align the preserved current page (the last one read) into view so
    // returning lands on the last page even if the exact offset was lost.
    restoreScrollPosition(currentPage.value)
    // Fine-tune to the exact captured offset (e.g. bottom of the last page).
    if (savedScrollTop > 0) {
      el.scrollTop = savedScrollTop
    }
    lastScrollTop = el.scrollTop
    savedScrollTop = 0
    // Recompute the page from the restored position so the toolbar matches.
    updateScrollPage()
    const room = el.scrollHeight - el.clientHeight
    if (room > 1 && el.scrollTop + el.clientHeight >= el.scrollHeight - 1) {
      suppressNextComplete = true
    }
  })
})

onDeactivated(() => {
  document.removeEventListener('keydown', handleKey)
  // NOTE: KeepAlive has already detached the DOM here, so el.scrollTop reads 0.
  // Use the offset captured by handleScroll/restoreScrollPosition instead.
  if (pdfViewMode.value === 'scroll') savedScrollTop = lastScrollTop
  flushSave()
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
  saveProgressNow,
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
      <div v-else-if="pdfViewMode === 'scroll'" ref="pdfScrollRef" class="pdf-container pdf-scroll">
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
        <!-- Shown as the reader crosses into the bottom buffer: scrolling past
             this point ends the book. -->
        <div class="scroll-end-hint">{{ t('reader.scrollEndHint') }}</div>
      </div>
    </div>

    <!-- Bottom navigation -->
    <div v-if="pdfViewMode !== 'scroll' && totalPages > 0" class="pdf-controls">
      <el-button size="small" :disabled="currentPage <= 1" @click="prevPage">{{ t('reader.prev') }}</el-button>
      <el-input-number v-model="currentPage" :min="1" :max="totalPages" size="small" controls-position="right" class="page-input" @change="goToPage" />
      <span class="page-total">/ {{ totalPages }}</span>
      <el-button size="small" @click="nextPage">{{ t('reader.next') }}</el-button>
    </div>

    <!-- Initial render overlay: hides the raw canvas until the resume page is
         rendered and positioned, so there is no page-1 flash / mid-load jump. -->
    <div v-if="loading" class="pdf-loading">
      <div class="pdf-loading-spinner"></div>
      <span>{{ t('reader.loading') }}</span>
    </div>
  </div>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.pdf-reader {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
  position: relative;
}
.pdf-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
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
  // Bottom buffer after the last page: reaching the end of the content only
  // means "just finished reading the last page" — the user must scroll this
  // extra space before the reading-complete page triggers. Half a viewport so a
  // normal wheel flick lands inside the buffer rather than instantly completing.
  padding: @gap-md 0 50vh;
  gap: @gap-sm;
}
.pdf-canvas {
  box-shadow: @shadow-card;
}
.pdf-double .double-pages {
  display: flex;
  gap: @gap-xs;
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
.scroll-end-hint {
  text-align: center;
  color: var(--text-secondary, #909399);
  font-size: @font-sm;
  padding: @gap-sm 0;
}
.pdf-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: @control-gap;
  padding: @toolbar-padding;
  border-top: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  flex-shrink: 0;
}
.page-input {
  width: @page-input-width;
}
.page-total {
  font-size: @font-md;
}
.pdf-loading {
  position: absolute;
  inset: 0;
  z-index: 20;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: @gap-sm;
  background: var(--bg-primary, #fff);
  color: var(--text-secondary, #909399);
  font-size: @font-md;
}
.pdf-loading-spinner {
  width: calc(28 * @w);
  height: calc(28 * @w);
  border: calc(3 * @w) solid var(--border-color, #e0e0e0);
  border-top-color: #409eff;
  border-radius: 50%;
  animation: pdf-loading-rotate 0.8s linear infinite;
}
@keyframes pdf-loading-rotate {
  to { transform: rotate(360deg); }
}
</style>
