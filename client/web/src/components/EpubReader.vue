<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
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
}>()

const { t } = useI18n()

const tocVisible = ref(false)
const tocItems = ref<any[]>([])
const epubArea = ref<HTMLElement | null>(null)
const currentLocation = ref<any>(null)
const epubViewMode = ref<'single' | 'double'>('single')
const progressPercent = ref(Math.round((props.initialProgress || 0) * 100))

let rendition: any = null
let book: any = null
let saveTimer: ReturnType<typeof setTimeout> | null = null

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

  await rendition.display(props.restoredCfi || undefined)

  // Generate locations in background — needed for percentage progress
  book.locations.generate(1600).then(() => {
    if (currentLocation.value?.start?.cfi) {
      const pct = book.locations.percentageFromCfi(currentLocation.value.start.cfi)
      if (pct != null && pct >= 0) {
        progressPercent.value = Math.round(pct * 100)
        emit('progressUpdate', pct)
      }
    }
  })

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
  init()
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
      <el-button size="small" :disabled="currentLocation?.atEnd" @click="nextPage">{{ t('reader.next') }}</el-button>
    </div>
  </div>
</template>

<style scoped>
.epub-reader {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
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
  width: 240px;
  min-width: 240px;
  border-right: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  overflow-y: auto;
  flex-shrink: 0;
}
.toc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
  font-weight: 600;
  font-size: 0.85rem;
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
  padding: 8px 16px;
  cursor: pointer;
  font-size: 0.8rem;
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
  gap: 12px;
  padding: 4px 12px;
  border-top: 1px solid var(--border-color, #e0e0e0);
  background: var(--bg-primary, #fff);
  flex-shrink: 0;
}
.epub-progress {
  font-size: 0.8rem;
  color: var(--text-secondary, #666);
  min-width: 60px;
  text-align: center;
}
</style>
