<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getBooks, uploadBook, deleteBook, reprocessBook, getSettings, updateBook } from '@/api'
import type { Book } from '@/types/book'
import { useI18n } from 'vue-i18n'
import BookCard from '@/components/BookCard.vue'

const { t } = useI18n()
const router = useRouter()

const books = ref<Book[]>([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const loading = ref(false)
const pageSize = ref(20)
const sortField = ref('updated_at')
const sortOrder = ref('desc')

let searchTimer: ReturnType<typeof setTimeout> | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

const fetchBooks = async () => {
  loading.value = true
  try {
    const res = await getBooks(page.value, pageSize.value, keyword.value, '', '', sortField.value, sortOrder.value)
    books.value = res.data.data.list || []
    total.value = res.data.data.total
  } finally {
    loading.value = false
  }
}

const fetchSettings = async () => {
  try {
    const res = await getSettings()
    const s = res.data.data
    if (s['ui.page_size']) pageSize.value = parseInt(s['ui.page_size'])
    if (s['ui.sort_field']) sortField.value = s['ui.sort_field']
    if (s['ui.sort_order']) sortOrder.value = s['ui.sort_order']
  } catch { /* ignore */ }
}

const onSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  page.value = 1
  searchTimer = setTimeout(fetchBooks, 300)
}

const handleUpload = async (options: any) => {
  try {
    await uploadBook(options.file)
    ElMessage.success(t('bookShelf.uploadSuccess'))
    fetchBooks()
  } catch {
    ElMessage.error(t('bookShelf.uploadFailed'))
  }
}

const handleDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm(t('bookShelf.deleteConfirm'))
    await deleteBook(id)
    ElMessage.success(t('bookShelf.deleteSuccess'))
    fetchBooks()
  } catch { /* cancel */ }
}

const handleReprocess = async (id: number) => {
  try {
    await reprocessBook(id)
    ElMessage.success(t('bookShelf.reprocessStarted'))
    fetchBooks()
  } catch {
    ElMessage.error(t('bookShelf.reprocessFailed'))
  }
}

const handleRead = (id: number) => {
  router.push({ path: '/read', query: { id: String(id) } })
}

const handleUpdateReadStatus = async (id: number, status: string) => {
  try {
    await updateBook(id, { read_status: status })
    fetchBooks()
  } catch {
    ElMessage.error(t('bookShelf.updateFailed') || '更新失败')
  }
}

// Poll for processing books
const pollProcessing = () => {
  const processingBooks = books.value.filter(b => b.book_status === 'processing')
  if (processingBooks.length > 0) {
    fetchBooks()
  }
}

onMounted(async () => {
  await fetchSettings()
  await fetchBooks()
  pollTimer = setInterval(pollProcessing, 2000)
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <el-container class="shelf-container">
    <el-header class="shelf-header">
      <h1 class="shelf-title">{{ t('bookShelf.title') }}</h1>
      <el-input
        v-model="keyword"
        :placeholder="t('bookShelf.searchPlaceholder')"
        clearable
        @clear="fetchBooks"
        @input="onSearchInput"
        class="search-input"
      />
      <div class="spacer"></div>
      <el-upload :show-file-list="false" :http-request="handleUpload" accept=".epub,.pdf,.EPUB,.PDF">
        <el-button type="primary">{{ t('bookShelf.uploadButton') }}</el-button>
      </el-upload>
      <el-button @click="$router.push('/settings')">{{ t('bookShelf.settingsButton') }}</el-button>
    </el-header>
    <el-main class="shelf-main">
      <div v-if="!loading && books.length === 0" class="empty-state">
        <el-empty :description="t('bookShelf.emptyDescription')" />
      </div>
      <el-row :gutter="16" v-loading="loading">
        <el-col v-for="book in books" :key="book.id" :xs="12" :sm="8" :md="6" :lg="4" class="book-col">
          <BookCard :book="book" @read="handleRead" @delete="handleDelete" @reprocess="handleReprocess" @update-read-status="handleUpdateReadStatus" />
        </el-col>
      </el-row>
      <div class="pagination-wrapper" v-if="total > pageSize">
        <el-pagination
          background
          layout="prev, pager, next"
          :total="total"
          :page-size="pageSize"
          v-model:current-page="page"
          @current-change="fetchBooks"
        />
      </div>
    </el-main>
  </el-container>
</template>

<style scoped>
.shelf-container {
  min-height: 100vh;
}
.shelf-header {
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
}
.shelf-title {
  margin: 0;
  font-size: 1.3rem;
  white-space: nowrap;
}
.search-input {
  flex: 1;
  max-width: 400px;
}
.spacer {
  flex: 1;
}
.shelf-main {
  padding: 16px;
}
.empty-state {
  margin-top: 80px;
}
.book-col {
  margin-bottom: 16px;
}
.pagination-wrapper {
  margin-top: 16px;
  text-align: center;
}
</style>
