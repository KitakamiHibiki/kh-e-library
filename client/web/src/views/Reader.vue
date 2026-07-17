<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ElMessage } from "element-plus"
import {
  getBook, getReadUrl, getProgress, saveProgress,
  getBookmarks, createBookmark, deleteBookmark, getReadUrl
} from "@/api"

const route = useRoute()
const router = useRouter()
const bookId = Number(route.params.id)
const isPdf = ref(false)

const loading = ref(true)
const progressVal = ref(0)
const bookmarks = ref<any[]>([])
const showBookmarks = ref(false)
const bookTitle = ref("")

let rendition: any = null
let book: any = null
let saveTimer: any = null

const initReader = async () => {
  try {
    const bookRes = await getBook(bookId)
    bookTitle.value = bookRes.data.data.title || ""
    isPdf.value = (bookRes.data.data.file_path || "").toLowerCase().endsWith(".pdf")
    if (isPdf.value) { loading.value = false; return }

    // load progress
    const progRes = await getProgress(bookId)
    const restoredCfi = progRes.data.data?.cfi || null
    if (progRes.data.data) {
      progressVal.value = progRes.data.data.progress || 0
    }

    // load bookmarks
    const bmRes = await getBookmarks(bookId)
    bookmarks.value = bmRes.data.data || []

    // init epubjs
    const ePub = (await import("epubjs")).default
    book = ePub(getReadUrl(bookId))
    rendition = book.renderTo("reader-area", {
      width: "100%",
      height: window.innerHeight - 50,
      spread: "none",
    })

    const startCfi = restoredCfi || undefined
    await rendition.display(startCfi)

    rendition.on("relocated", (loc: any) => {
      if (loc && loc.percentage != null) {
        progressVal.value = loc.percentage
        scheduleSave(loc)
      }
    })
  } finally {
    loading.value = false
  }
}

const scheduleSave = (loc: any) => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await saveProgress({
        book_id: bookId,
        progress: loc.percentage,
        cfi: loc.start.cfi,
        chapter_href: loc.start.href,
      })
    } catch { /* ignore */ }
  }, 2000)
}

const addBookmark = async () => {
  if (!rendition) return
  const loc = rendition.currentLocation()
  if (!loc) return
  try {
    await createBookmark({
      book_id: bookId,
      cfi: loc.start.cfi,
      chapter_href: loc.start.href,
      chapter_name: "",
      progress: loc.percentage,
    })
    ElMessage.success("bookmark added")
    const res = await getBookmarks(bookId)
    bookmarks.value = res.data.data || []
  } catch {
    ElMessage.error("failed to add bookmark")
  }
}

const removeBookmark = async (id: number) => {
  await deleteBookmark(id)
  bookmarks.value = bookmarks.value.filter((b: any) => b.id !== id)
}

const goToBookmark = async (bm: any) => {
  if (!rendition) return
  showBookmarks.value = false
  await rendition.display(bm.cfi)
}

onMounted(initReader)
onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer)
  if (rendition) rendition.destroy()
})
</script>

<template>
  <div style="height: 100vh; display: flex; flex-direction: column; overflow: hidden">
    <!-- toolbar -->
    <div style="display: flex; align-items: center; padding: 6px 12px; border-bottom: 1px solid #e0e0e0; background: #fff; z-index: 10; gap: 8px; flex-shrink: 0">
      <el-button size="small" @click="router.push('/')">back</el-button>
      <span style="font-size: 0.9rem; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ bookTitle }}</span>
      <span v-if="!isPdf" style="font-size: 0.8rem; color: #666">{{ Math.round(progressVal * 100) }}%</span>
      <el-button v-if="!isPdf" size="small" @click="addBookmark">+ bookmark</el-button>
      <el-button v-if="!isPdf" size="small" :type="showBookmarks ? 'primary' : 'default'" @click="showBookmarks = !showBookmarks">
        bookmarks ({{ bookmarks.length }})
      </el-button>
    </div>

    <!-- reader area -->
    <div style="flex: 1; position: relative; overflow: hidden">
      <div v-if="loading" style="display: flex; justify-content: center; align-items: center; height: 100%">
        <el-icon class="is-loading" :size="32"><i class="el-icon-loading"></i></el-icon>
        <span style="margin-left: 8px">loading book...</span>
      </div>
      <iframe v-if="isPdf" :src="getReadUrl(bookId)" style="width: 100%; height: 100%; border: none"></iframe>
      <div v-else id="reader-area" style="height: 100%"></div>
    </div>

    <!-- bookmarks panel -->
    <div v-if="showBookmarks" style="max-height: 30vh; overflow-y: auto; border-top: 1px solid #e0e0e0; background: #fafafa; padding: 8px 16px; flex-shrink: 0">
      <div v-for="bm in bookmarks" :key="bm.id" style="display: flex; align-items: center; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid #eee">
        <a style="cursor: pointer; color: #409eff; font-size: 0.85rem" @click="goToBookmark(bm)">
          {{ bm.chapter_name || "page" + " " + Math.round(bm.progress * 100) + "%" }}
        </a>
        <el-button size="small" type="danger" link @click="removeBookmark(bm.id)">del</el-button>
      </div>
      <div v-if="bookmarks.length === 0" style="color: #999; font-size: 0.85rem; padding: 8px 0">no bookmarks</div>
    </div>
  </div>
</template>

<style>
#reader-area iframe { border: none; }
</style>
