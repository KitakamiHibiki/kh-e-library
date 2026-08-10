<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, onActivated, onDeactivated } from 'vue'
import { getReadUrl, saveProgress, getSettings, updateSettings } from '@/api'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  bookId: number
  restoredCfi: string | null
  fontSize: number
  initialProgress: number
}>()

const emit = defineEmits<{
  (e: 'progressUpdate', progress: number): void
  (e: 'tocLoaded', items: any[]): void
  (e: 'stateChange', state: { epubViewMode: string }): void
  (e: 'completed'): void
}>()

const { t } = useI18n()

const tocVisible = ref(false)
const tocItems = ref<any[]>([])
const epubArea = ref<HTMLElement | null>(null)
const loading = ref(true)
const currentLocation = ref<any>(null)
const epubViewMode = ref<'single' | 'double'>('single')
const progressPercent = ref(Math.round((props.initialProgress || 0) * 100))

let rendition: any = null
let book: any = null
let saveTimer: ReturnType<typeof setTimeout> | null = null
// Last CFI, captured on deactivation so returning from the completion page can
// re-display the exact page (the iframe's internal scroll may reset while the
// KeepAlive DOM is detached).
let lastCfi: string | null = null
// True between a KeepAlive re-activation and the completed re-display. While
// set, a late epubjs resize (it re-renders from its own last location, which
// the hidden-container resize may have corrupted) re-applies the real target
// instead of leaving the reader on the section's first page.
let restoring = false

const emitState = () => {
  emit('stateChange', { epubViewMode: epubViewMode.value })
}

const onViewModeChange = (val: 'single' | 'double') => {
  epubViewMode.value = val
  if (rendition) {
    rendition.spread(val === 'double' ? 'auto' : 'none')
  }
  updateSettings({ 'reader.epub_view_mode': val }).catch(() => {})
  emitState()
}

/**
 * Save progress immediately (called on exit/unmount).
 * Uses the latest progressPercent value which is always up-to-date.
 */
const saveProgressNow = async () => {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
  if (!currentLocation.value) return
  const cfi = currentLocation.value.start?.cfi
  const href = currentLocation.value.start?.href
  const pct = progressPercent.value / 100
  if (pct > 0 || cfi) {
    try {
      await saveProgress({ book_id: props.bookId, progress: pct, cfi, chapter_href: href })
    } catch { /* ignore */ }
  }
}

/**
 * Re-display the reader at a given CFI and resolve with whether the reader
 * actually reached that position (the page within the section, not just the
 * section). Used when returning to the reader (KeepAlive re-activation): the
 * rendition's own location drifts while the iframe sits in the hidden
 * container, and the manager measures the target CFI's coordinates before the
 * re-attached content is laid out — which can land on the section's first page.
 * The caller retries until this reports success.
 */
const restoreTo = (cfi: string | null): Promise<boolean> => {
  if (!rendition || !cfi) return Promise.resolve(false)
  return rendition.display(cfi)
    .then(() => new Promise<boolean>((resolve) => {
      // Let the relocation settle, then verify the within-section page matches.
      setTimeout(() => {
        const cur = currentLocation.value?.start?.cfi || ''
        const ok = cur.split('!')[1] === cfi.split('!')[1]
        resolve(ok)
      }, 80)
    }))
    .catch(() => false)
}

const cleanup = () => {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
  try { if (rendition) { rendition.destroy(); rendition = null } } catch { rendition = null }
  try { if (book) { book.destroy?.(); book = null } } catch { book = null }
  tocVisible.value = false
  tocItems.value = []
  currentLocation.value = null
  progressPercent.value = 0
}

/**
 * Get the epubjs container element (.epub-container).
 * This is the scrollable element that epubjs uses for paginated flow.
 */
const getContainer = (): HTMLElement | null => {
  const area = epubArea.value
  if (!area) return null
  return area.querySelector('.epub-container')
}

/**
 * Custom page navigation: directly manipulate scrollLeft on the .epub-container
 * instead of relying on epubjs's rendition.next()/prev() which uses a stale
 * layout.delta that can be larger than the actual clientWidth, causing it to
 * prematurely think the chapter has ended and skip remaining pages.
 */
const prevPage = () => {
  if (!rendition) return
  const container = getContainer()
  if (!container) { rendition.prev(); return }

  const step = container.clientWidth
  if (container.scrollLeft <= 1) {
    // At the very beginning of this chapter — go to previous chapter
    rendition.prev()
  } else {
    container.scrollLeft = Math.max(0, container.scrollLeft - step)
    rendition.reportLocation()
  }
}

const nextPage = () => {
  if (!rendition) return
  // Already on the last page — treat "next" as reaching the end of the book.
  if (currentLocation.value?.atEnd) {
    emit('completed')
    return
  }
  const container = getContainer()
  if (!container) { rendition.next(); return }

  const step = container.clientWidth
  const maxScroll = container.scrollWidth - container.clientWidth
  if (maxScroll <= 0 || container.scrollLeft >= maxScroll - 1) {
    // At the very end of this chapter — go to next chapter
    rendition.next()
  } else {
    container.scrollLeft = Math.min(maxScroll, container.scrollLeft + step)
    rendition.reportLocation()
  }
}

const init = async () => {
  const ePub = (await import('epubjs')).default
  const response = await fetch(getReadUrl(props.bookId))
  const data = await response.arrayBuffer()
  book = ePub(data)
  const area = epubArea.value
  if (!area) return

  // Load epub view mode setting
  try {
    const settingsRes = await getSettings()
    const vm = settingsRes.data.data?.['reader.epub_view_mode']
    if (vm === 'single' || vm === 'double') epubViewMode.value = vm
  } catch { /* ignore */ }

  rendition = book.renderTo(area, {
    width: '100%',
    height: '100%',
    spread: epubViewMode.value === 'double' ? 'auto' : 'none',
    flow: 'paginated',
    allowScriptedContent: true,
  })

  rendition.themes.register('custom', { body: { 'font-size': `${props.fontSize}px` } })
  rendition.themes.select('custom')

  const isDark = document.documentElement.classList.contains('dark')
  if (isDark) {
    rendition.themes.override('background', '#1a1a1a')
    rendition.themes.override('color', '#e0e0e0')
    rendition.themes.override('a', 'color', '#409eff')
  }

  // Register listeners BEFORE the first display: epubjs reports the initial
  // location on a rAF after display() resolves, so if the handler is attached
  // only after the locations await below, that first "relocated" is missed and
  // currentLocation stays null — losing the restore target when the reader is
  // deactivated right after entering.
  rendition.on('relocated', (loc: any) => {
    currentLocation.value = loc

    // Try percentageFromCfi first (accurate after locations generated),
    // then loc.percentage (from epubjs internal), skip if both unavailable
    let pct: number | null = null
    if (loc?.start?.cfi && book?.locations) {
      try {
        pct = book.locations.percentageFromCfi(loc.start.cfi)
      } catch { /* ignore */ }
    }
    if (pct == null || pct < 0) {
      pct = loc?.percentage ?? null
    }
    // Only update if we got a valid value; never overwrite with 0 on inner-chapter pages
    // (epubjs can return incorrect 0% when using custom scrollLeft navigation)
    if (pct != null && pct > 0) {
      progressPercent.value = Math.round(pct * 100)
      emit('progressUpdate', pct)
      const cfi = loc.start?.cfi
      const href = loc.start?.href
      scheduleSave(cfi, pct, href)
    } else if (pct === 0) {
      // Only accept 0% if we're genuinely at the start of the book
      const isFirstChapter = loc?.start?.index === 0
      const isFirstPage = loc?.start?.displayed?.page === 1
      if (isFirstChapter && isFirstPage) {
        progressPercent.value = 0
        emit('progressUpdate', 0)
        const cfi = loc.start?.cfi
        const href = loc.start?.href
        scheduleSave(cfi, 0, href)
      }
    }

    emitState()
  })

  await rendition.display(props.restoredCfi || undefined)

  // Generate locations — needed for accurate percentage progress. Keep the
  // loading overlay up until the first page renders AND this data processing
  // finishes (capped), so the reader is never revealed with a misleading 0%
  // progress before it is actually usable.
  const locationsReady = book.locations.generate(1600).catch(() => {})
  await Promise.race([
    locationsReady,
    new Promise((resolve) => setTimeout(resolve, 6000)),
  ])

  // Reflect the actual position now that percentageFromCfi is usable.
  const syncProgress = () => {
    if (!currentLocation.value?.start?.cfi) return
    try {
      const pct = book.locations.percentageFromCfi(currentLocation.value.start.cfi)
      if (pct != null && pct >= 0) {
        progressPercent.value = Math.round(pct * 100)
        emit('progressUpdate', pct)
      }
    } catch { /* ignore */ }
  }
  syncProgress()
  // If generation finished only after the cap, still fix the progress once ready.
  locationsReady.then(syncProgress)

  // First page rendered and data processed — reveal the reader.
  loading.value = false

  book.loaded.navigation.then((nav: any) => {
    tocItems.value = nav.toc || []
    emit('tocLoaded', tocItems.value)
  })

  emitState()
}

const navigateToToc = (item: any) => {
  if (rendition && item.href) {
    rendition.display(item.href)
    tocVisible.value = false
  }
}

const toggleToc = () => { tocVisible.value = !tocVisible.value }

const scheduleSave = (cfi: string | undefined, progress: number, href?: string) => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await saveProgress({ book_id: props.bookId, progress, cfi, chapter_href: href })
    } catch { /* ignore */ }
  }, 500)
}

const forwardClick = (e: MouseEvent) => {
  const area = epubArea.value
  if (!area) return
  const iframe = area.querySelector('iframe')
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc) return
    const target = doc.elementFromPoint(e.offsetX, e.offsetY)
    if (target instanceof HTMLElement) target.click()
  } catch { /* ignore */ }
}

const handleWheel = (e: WheelEvent) => {
  if (!rendition) return
  e.preventDefault()
  if (e.deltaY > 0) nextPage()
  else if (e.deltaY < 0) prevPage()
}

const handleKey = (e: KeyboardEvent) => {
  if (e.key === 'ArrowLeft') { prevPage(); e.preventDefault() }
  if (e.key === 'ArrowRight') { nextPage(); e.preventDefault() }
}

onMounted(() => {
  document.addEventListener('keydown', handleKey)
  // Never leave the loading overlay up if init fails.
  init().catch(() => { loading.value = false })
})

// KeepAlive lifecycle: the parsed book + rendition stay alive across the
// completion page, so returning to /read is instant (no re-parse). Only the
// keydown listener and progress save need gating/finalizing here.
onActivated(() => {
  // Re-add idempotently (avoid double listeners after deactivation).
  document.removeEventListener('keydown', handleKey)
  document.addEventListener('keydown', handleKey)
  const target = lastCfi || currentLocation.value?.start?.cfi || null
  if (!rendition || !target) return
  // Show the loading overlay while the re-attached iframe is laid out and we
  // re-align to the captured page, so the section's first page never flashes
  // before the real page is restored. Retry until the within-section position
  // matches (the first attempt can land on page 1 when the content is still
  // being laid out).
  restoring = true
  loading.value = true
  let tries = 0
  const attempt = () => {
    if (!restoring) { loading.value = false; return }
    if (tries++ > 12) { restoring = false; loading.value = false; return }
    restoreTo(target).then((ok) => {
      if (!restoring) { loading.value = false; return }
      if (ok) { restoring = false; loading.value = false }
      else setTimeout(attempt, 150)
    })
  }
  attempt()
})

onDeactivated(() => {
  document.removeEventListener('keydown', handleKey)
  lastCfi = currentLocation.value?.start?.cfi || null
  restoring = false
  saveProgressNow()
})

onBeforeUnmount(async () => {
  document.removeEventListener('keydown', handleKey)
  await saveProgressNow()
  cleanup()
})

defineExpose({
  tocVisible,
  navigateToToc,
  toggleToc,
  prevPage,
  nextPage,
  epubViewMode,
  onViewModeChange,
  saveProgressNow,
})
</script>

<template>
  <div class="epub-reader">
    <div class="epub-layout">
      <div v-if="tocVisible" class="toc-sidebar">
        <div class="toc-header">
          <span class="toc-title">{{ t('reader.toc') }}</span>
          <el-button size="small" text @click="tocVisible = false">✕</el-button>
        </div>
        <ul class="toc-list">
          <li v-for="(item, idx) in tocItems" :key="idx" class="toc-item" @click="navigateToToc(item)">
            {{ item.label }}
          </li>
        </ul>
      </div>
      <div class="epub-area-wrapper">
        <div ref="epubArea" class="epub-area"></div>
        <div class="epub-overlay" @wheel.prevent="handleWheel" @click="forwardClick"></div>
      </div>
    </div>
    <div class="epub-controls">
      <el-button size="small" :disabled="currentLocation?.atStart" @click="prevPage">{{ t('reader.prev') }}</el-button>
      <span class="epub-progress">{{ progressPercent }}%</span>
      <el-button size="small" @click="nextPage">{{ t('reader.next') }}</el-button>
    </div>

    <!-- Initial render overlay: keeps the reader hidden until the first page
         has rendered, so there is no flash of an empty iframe. -->
    <div v-if="loading" class="epub-loading">
      <div class="epub-loading-spinner"></div>
      <span>{{ t('reader.loading') }}</span>
    </div>
  </div>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.epub-reader {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
  position: relative;
}
.epub-layout {
  display: flex;
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}
.epub-area-wrapper {
  flex: 1;
  min-height: 0;
  min-width: 0;
  position: relative;
  overflow: hidden;
}
.epub-area {
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.epub-overlay {
  position: absolute;
  inset: 0;
  z-index: 1;
  cursor: default;
}
.epub-area :deep(iframe) {
  border: none !important;
}
.epub-area :deep(.epub-container) {
  height: 100% !important;
}
.toc-sidebar {
  width: @toc-width;
  min-width: @toc-width;
  border-right: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  overflow-y: auto;
  flex-shrink: 0;
}
.toc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: @gap-sm @gap-md;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
  font-weight: 600;
  font-size: @font-md;
}
.toc-title {
  color: var(--text-primary, #303133);
}
.toc-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.toc-item {
  padding: @toc-item-padding;
  cursor: pointer;
  font-size: @font-sm;
  color: var(--text-primary, #303133);
  border-bottom: 1px solid var(--border-color, #eee);
  transition: background 0.15s;
}
.toc-item:hover {
  background: var(--bg-secondary, #f5f5f5);
}
.epub-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: @gap-md;
  padding: @toolbar-padding;
  border-top: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  flex-shrink: 0;
}
.epub-progress {
  font-size: @font-sm;
  color: var(--text-secondary, #666);
  min-width: calc(60 * @w);
  text-align: center;
}
.epub-loading {
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
.epub-loading-spinner {
  width: calc(28 * @w);
  height: calc(28 * @w);
  border: calc(3 * @w) solid var(--border-color, #e0e0e0);
  border-top-color: #409eff;
  border-radius: 50%;
  animation: epub-loading-rotate 0.8s linear infinite;
}
@keyframes epub-loading-rotate {
  to { transform: rotate(360deg); }
}
</style>
