<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getBook, updateBook, getCoverUrl } from '@/api'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const bookId = ref(Number(route.query.id) || 0)
const bookTitle = ref('')
const hasCover = ref(false)
const isFinished = ref(false)
const marking = ref(false)

const coverUrl = computed(() => getCoverUrl(bookId.value))

const coverError = (e: Event) => {
  ;(e.target as HTMLImageElement).style.display = 'none'
}

const loadBook = async () => {
  if (!bookId.value) return
  try {
    const res = await getBook(bookId.value)
    const book = res.data.data
    bookTitle.value = book.title || ''
    hasCover.value = !!book.cover
    isFinished.value = book.read_status === 'finished'
  } catch { /* ignore */ }
}

const markFinished = async () => {
  if (!bookId.value || isFinished.value) return
  marking.value = true
  try {
    await updateBook(bookId.value, { read_status: 'finished' })
    isFinished.value = true
    ElMessage.success(t('readComplete.markSuccess'))
  } catch {
    ElMessage.error(t('readComplete.markFailed'))
  } finally {
    marking.value = false
  }
}

const readAgain = () => {
  // restart value is a timestamp so each "read again" gets a fresh KeepAlive
  // slot (unique fullPath) instead of reactivating the previous end-of-book one.
  // Replace: the completion page is a terminal state; the new session swaps it
  // out so the back button reaches the shelf, not back to "finished".
  router.replace({ path: '/read', query: { id: String(bookId.value), restart: String(Date.now()) } })
}

const goBack = () => {
  router.replace('/')
}

// Return to the reader at its last saved position. Triggered by the "previous"
// gesture (scroll up / swipe up) on the completion page — the mirror of how the
// reader reached this page by going forward past the last page.
const backToReader = () => {
  if (!bookId.value) { router.replace('/'); return }
  // Replace: returning to the reader swaps the completion page out so the back
  // button leads to the shelf, not back to the "finished" state.
  router.replace({ path: '/read', query: { id: String(bookId.value) } })
}

const handleWheel = (e: WheelEvent) => {
  if (e.deltaY < 0) {
    e.preventDefault()
    backToReader()
  }
}

const handleKey = (e: KeyboardEvent) => {
  if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
    e.preventDefault()
    backToReader()
  }
}

onMounted(() => {
  loadBook()
  window.addEventListener('wheel', handleWheel, { passive: false })
  document.addEventListener('keydown', handleKey)
})

onBeforeUnmount(() => {
  window.removeEventListener('wheel', handleWheel)
  document.removeEventListener('keydown', handleKey)
})
</script>

<template>
  <div class="complete-container">
    <div class="complete-card">
      <div class="complete-cover">
        <img v-if="hasCover" :src="coverUrl" @error="coverError" class="cover-img" />
        <span v-else class="cover-badge">+</span>
      </div>
      <h2 class="complete-title">{{ bookTitle }}</h2>
      <p class="complete-subtitle">{{ t('readComplete.title') }}</p>

      <div class="complete-actions">
        <el-button
          type="primary"
          plain
          class="mark-btn"
          :loading="marking"
          :disabled="isFinished"
          @click="markFinished"
        >
          {{ isFinished ? t('readComplete.marked') : t('readComplete.markFinished') }}
        </el-button>
        <div class="btn-row">
          <el-button size="large" @click="readAgain">{{ t('readComplete.readAgain') }}</el-button>
          <el-button size="large" type="primary" @click="goBack">{{ t('readComplete.back') }}</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.complete-container {
  height: 100vh;
  height: 100dvh;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: var(--bg-primary, #fff);
  color: var(--text-primary, #303133);
  padding: @gap-xl;
}
.complete-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: calc(320 * @w);
  width: 100%;
}
.complete-cover {
  width: calc(180 * @w);
  height: calc(240 * @h);
  background: var(--bg-secondary, #f5f5f5);
  border-radius: @radius-sm;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: @shadow-card;
  margin-bottom: @gap-lg;
}
.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-badge {
  font-size: 2.5rem;
  color: var(--text-secondary, #999);
}
.complete-title {
  margin: 0 0 @gap-sm;
  font-size: @font-lg;
}
.complete-subtitle {
  margin: 0 0 @gap-xl;
  color: var(--text-secondary, #909399);
  font-size: @font-md;
}
.complete-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: @gap-md;
  width: 100%;
}
.mark-btn {
  width: 100%;
}
.btn-row {
  display: flex;
  gap: @gap-md;
}
.btn-row .el-button {
  min-width: calc(100 * @w);
}
</style>
