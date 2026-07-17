<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ElMessage } from "element-plus"
import { getBook, getReadUrl, getProgress, saveProgress, getBookmarks, createBookmark, deleteBookmark } from "@/api"

const route = useRoute()
const router = useRouter()
const bookId = Number(route.params.id)

const loading = ref(true)
const isPdf = ref(false)
const progressVal = ref(0)
const bookmarks = ref<any[]>([])
const showBookmarks = ref(false)
const bookTitle = ref("")

const currentPage = ref(1)
const totalPages = ref(0)
const pdfCanvas = ref<HTMLCanvasElement | null>(null)

let rendition: any = null
let book: any = null
let pdfDoc: any = null
let saveTimer: any = null

const initReader = async () => {
  try {
    const bookRes = await getBook(bookId)
    bookTitle.value = bookRes.data.data.title || ""
    isPdf.value = (bookRes.data.data.file_path || "").toLowerCase().endsWith(".pdf")

    const progRes = await getProgress(bookId)
    const restoredCfi = progRes.data.data?.cfi || null
    if (progRes.data.data) progressVal.value = progRes.data.data.progress || 0

    const bmRes = await getBookmarks(bookId)
    bookmarks.value = bmRes.data.data || []

    if (isPdf.value) {
      await initPDF(restoredCfi)
    } else {
      await initEPUB(restoredCfi)
    }
  } finally {
    loading.value = false
  }
}

// ===== PDF =====
const initPDF = async (restoredPage: string | null) => {
  await nextTick()
  const pdfjsLib = await import("pdfjs-dist")
  const ver = (pdfjsLib as any).version
  pdfjsLib.GlobalWorkerOptions.workerSrc = "https://cdnjs.cloudflare.com/ajax/libs/pdf.js/" + ver + "/pdf.worker.min.mjs"

  const doc = await (pdfjsLib as any).getDocument(getReadUrl(bookId)).promise
  pdfDoc = doc
  totalPages.value = doc.numPages

  const startPage = restoredPage ? Math.min(parseInt(restoredPage), doc.numPages) : 1
  currentPage.value = Math.max(startPage, 1)
  await renderPage(currentPage.value)
}

const renderPage = async (pageNum: number) => {
  if (!pdfDoc) return
  const page = await pdfDoc.getPage(pageNum)
  const canvas = pdfCanvas.value
  if (!canvas) return
  const ctx = canvas.getContext("2d")
  if (!ctx) return

  const containerWidth = canvas.parentElement?.clientWidth || 800
  const vp = page.getViewport({ scale: 1 })
  const scale = containerWidth / vp.width
  const svp = page.getViewport({ scale })

  canvas.width = svp.width
  canvas.height = svp.height
  await page.render({ canvasContext: ctx, viewport: svp }).promise

  const prog = pageNum / totalPages.value
  progressVal.value = prog
  scheduleSave(String(pageNum), prog)
}

const prevPage = () => {
  if (!pdfDoc || currentPage.value <= 1) return
  currentPage.value--
  renderPage(currentPage.value)
}

const nextPage = () => {
  if (!pdfDoc || currentPage.value >= totalPages.value) return
  currentPage.value++
  renderPage(currentPage.value)
}

const goToPage = () => {
  if (!pdfDoc) return
  let p = currentPage.value
  if (p < 1) p = 1
  if (p > totalPages.value) p = totalPages.value
  currentPage.value = p
  renderPage(p)
}

// ===== EPUB =====
const initEPUB = async (restoredCfi: string | null) => {
  await nextTick()
  const ePub = (await import("epubjs")).default
  book = ePub(getReadUrl(bookId))
  rendition = book.renderTo("reader-area", {
    width: "100%",
    height: window.innerHeight - 50,
    spread: "none",
  })
  await rendition.display(restoredCfi || undefined)
  rendition.on("relocated", (loc: any) => {
    if (loc && loc.percentage != null) {
      progressVal.value = loc.percentage
      scheduleSave(loc.start.cfi, loc.percentage, loc.start.href)
    }
  })
}

// ===== Shared =====
const scheduleSave = (cfi: string, progress: number, href?: string) => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await saveProgress({ book_id: bookId, progress, cfi, chapter_href: href })
    } catch { /* ignore */ }
  }, 2000)
}

const addBookmark = async () => {
  try {
    if (isPdf.value) {
      const p = currentPage.value
      await createBookmark({ book_id: bookId, cfi: String(p), chapter_name: "Page " + p, progress: p / totalPages.value })
    } else if (rendition) {
      const loc = rendition.currentLocation()
      if (!loc) return
      await createBookmark({ book_id: bookId, cfi: loc.start.cfi, chapter_href: loc.start.href, progress: loc.percentage })
    }
    ElMessage.success("bookmark added")
    const res = await getBookmarks(bookId)
    bookmarks.value = res.data.data || []
  } catch { ElMessage.error("failed") }
}

const removeBookmark = async (id: number) => {
  await deleteBookmark(id)
  bookmarks.value = bookmarks.value.filter((b: any) => b.id !== id)
}

const goToBookmark = async (bm: any) => {
  showBookmarks.value = false
  if (isPdf.value && pdfDoc) {
    const p = Math.max(1, Math.min(parseInt(bm.cfi) || 1, totalPages.value))
    currentPage.value = p
    await renderPage(p)
  } else if (rendition) {
    await rendition.display(bm.cfi)
  }
}

const handleKey = (e: KeyboardEvent) => {
  if (!isPdf.value) return
  if (e.key === "ArrowLeft") { prevPage(); e.preventDefault() }
  if (e.key === "ArrowRight") { nextPage(); e.preventDefault() }
}

onMounted(() => {
  document.addEventListener("keydown", handleKey)
  initReader()
})

onBeforeUnmount(() => {
  document.removeEventListener("keydown", handleKey)
  if (saveTimer) clearTimeout(saveTimer)
  if (rendition) rendition.destroy()
})
</script>

<template>
  <div style="height: 100vh; display: flex; flex-direction: column; overflow: hidden">
    <div style="display: flex; align-items: center; padding: 6px 12px; border-bottom: 1px solid #e0e0e0; background: #fff; z-index: 10; gap: 8px; flex-shrink: 0">
      <el-button size="small" @click="router.push('/')">back</el-button>
      <span style="font-size: 0.9rem; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ bookTitle }}</span>
      <span v-if="isPdf" style="font-size: 0.8rem; color: #666">Page {{ currentPage }}/{{ totalPages }} {{ Math.round(progressVal * 100) }}%</span>
      <span v-else style="font-size: 0.8rem; color: #666">{{ Math.round(progressVal * 100) }}%</span>
      <el-button size="small" @click="addBookmark">+ bookmark</el-button>
      <el-button size="small" :type="showBookmarks ? 'primary' : 'default'" @click="showBookmarks = !showBookmarks">bookmarks ({{ bookmarks.length }})</el-button>
    </div>
    <div style="flex: 1; position: relative; overflow: auto">
      <div v-if="loading" style="display: flex; justify-content: center; align-items: center; height: 100%"><span>loading book...</span></div>
      <div v-if="!isPdf && !loading" id="reader-area" style="height: 100%"></div>
      <div v-if="isPdf && !loading" style="display: flex; flex-direction: column; align-items: center; padding: 16px">
        <canvas ref="pdfCanvas" style="max-width: 100%; box-shadow: 0 2px 8px rgba(0,0,0,0.15)"></canvas>
        <div style="display: flex; align-items: center; gap: 8px; margin-top: 12px">
          <el-button size="small" :disabled="currentPage <= 1" @click="prevPage">Prev</el-button>
          <el-input-number v-model="currentPage" :min="1" :max="totalPages" size="small" controls-position="right" style="width: 130px" @change="goToPage" />
          <span style="font-size: 0.85rem">/ {{ totalPages }}</span>
          <el-button size="small" :disabled="currentPage >= totalPages" @click="nextPage">Next</el-button>
        </div>
      </div>
    </div>
    <div v-if="showBookmarks" style="max-height: 30vh; overflow-y: auto; border-top: 1px solid #e0e0e0; background: #fafafa; padding: 8px 16px; flex-shrink: 0">
      <div v-for="bm in bookmarks" :key="bm.id" style="display: flex; align-items: center; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid #eee">
        <a style="cursor: pointer; color: #409eff; font-size: 0.85rem" @click="goToBookmark(bm)">{{ bm.chapter_name || "page " + Math.round(bm.progress * 100) + "%" }}</a>
        <el-button size="small" type="danger" link @click="removeBookmark(bm.id)">del</el-button>
      </div>
      <div v-if="bookmarks.length === 0" style="color: #999; font-size: 0.85rem; padding: 8px 0">no bookmarks</div>
    </div>
  </div>
</template>
