<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getBooks, uploadBook, deleteBook, reprocessBook, getSettings, updateBook, getDownloadUrl, getTags, createTag, addBookTag, getSystemStatus, checkUpdate } from '@/api'
import type { Book, Tag } from '@/types/book'
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

// Shelf dialog state
const shelfDialogVisible = ref(false)
const shelfBookId = ref(0)
const shelfTags = ref<Tag[]>([])
const shelfTagSearch = ref('')
const shelfNewTagName = ref('')
const shelfLoading = ref(false)

// App version badge shown next to the title; hasUpdate drives the red dot.
const appVersion = ref('')
const hasUpdate = ref(false)

const displayVersion = computed(() => {
  const v = appVersion.value
  if (!v) return ''
  return /^\d/.test(v) ? `v${v}` : v
})

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

// Silently check for a newer release. The empty repo param makes the backend
// fall back to the configured update.github_repo; any failure (not configured,
// offline, rate-limited) just leaves the red dot hidden.
const checkForUpdate = async () => {
  try {
    const res = await checkUpdate('')
    hasUpdate.value = !!res.data.data.has_update
  } catch {
    hasUpdate.value = false
  }
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

const handleBookInfo = (id: number) => {
  ElMessage.info(`Book info for ID: ${id}`)
}

const handleExportBook = (id: number) => {
  window.open(getDownloadUrl(id), '_blank')
}

const openShelfDialog = async (id: number) => {
  shelfBookId.value = id
  shelfTagSearch.value = ''
  shelfNewTagName.value = ''
  await fetchShelfTags()
  shelfDialogVisible.value = true
}

const fetchShelfTags = async () => {
  shelfLoading.value = true
  try {
    const res = await getTags(shelfTagSearch.value)
    shelfTags.value = res.data.data
  } catch {
    shelfTags.value = []
  } finally {
    shelfLoading.value = false
  }
}

const handleAddTag = async (tagId: number) => {
  try {
    await addBookTag(shelfBookId.value, tagId)
    ElMessage.success(t('bookCard.addToShelfSuccess'))
    shelfDialogVisible.value = false
    fetchBooks()
  } catch {
    ElMessage.error(t('bookCard.addToShelfFailed'))
  }
}

const handleCreateAndAddTag = async () => {
  if (!shelfNewTagName.value.trim()) return
  try {
    const res = await createTag(shelfNewTagName.value.trim())
    await addBookTag(shelfBookId.value, res.data.data.id)
    ElMessage.success(t('bookCard.addToShelfSuccess'))
    shelfDialogVisible.value = false
    fetchBooks()
  } catch {
    ElMessage.error(t('bookCard.addToShelfFailed'))
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
  // Fire-and-forget: load the version badge and update indicator without
  // blocking the bookshelf render.
  getSystemStatus()
    .then(res => { appVersion.value = res.data.data.version })
    .catch(() => {})
  checkForUpdate()
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <el-container class="shelf-container">
    <el-header class="shelf-header">
      <div class="title-wrap">
        <h1 class="shelf-title">{{ t('bookShelf.title') }}</h1>
        <span
          v-if="displayVersion"
          class="version-badge"
          :class="{ 'has-update': hasUpdate }"
          :title="hasUpdate ? t('bookShelf.newVersionAvailable') : displayVersion"
        >
          {{ displayVersion }}
          <i v-if="hasUpdate" class="update-dot"></i>
        </span>
      </div>
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
          <BookCard
            :book="book"
            @read="handleRead"
            @delete="handleDelete"
            @reprocess="handleReprocess"
            @update-read-status="handleUpdateReadStatus"
            @book-info="handleBookInfo"
            @add-to-shelf="openShelfDialog"
            @export-book="handleExportBook"
          />
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

    <!-- Add to Shelf Dialog -->
    <el-dialog v-model="shelfDialogVisible" :title="t('bookCard.menuAddToShelf')" width="400px">
      <el-input
        v-model="shelfTagSearch"
        :placeholder="t('bookCard.searchTags')"
        clearable
        @input="fetchShelfTags"
        style="margin-bottom: 12px"
      />
      <div v-loading="shelfLoading">
        <div v-if="shelfTags.length > 0" style="max-height: 200px; overflow-y: auto">
          <div v-for="tag in shelfTags" :key="tag.id" style="margin-bottom: 6px">
            <el-button size="small" @click="handleAddTag(tag.id)" plain style="width: 100%; text-align: left">
              {{ tag.name }} ({{ tag.count }})
            </el-button>
          </div>
        </div>
        <el-empty v-else-if="!shelfLoading" :description="t('bookCard.noTags')" />
      </div>
      <el-divider />
      <div style="display: flex; gap: 8px">
        <el-input v-model="shelfNewTagName" :placeholder="t('bookCard.newTagPlaceholder')" @keyup.enter="handleCreateAndAddTag" />
        <el-button type="primary" @click="handleCreateAndAddTag" :disabled="!shelfNewTagName.trim()">
          {{ t('bookCard.createAndAdd') }}
        </el-button>
      </div>
    </el-dialog>
  </el-container>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.shelf-container {
  min-height: 100vh;
}
.shelf-header {
  display: flex;
  align-items: center;
  gap: @gap-md;
  border-bottom: 1px solid var(--border-color, #e0e0e0);
}
.shelf-title {
  margin: 0;
  font-size: @font-lg;
  white-space: nowrap;
}
.title-wrap {
  display: inline-flex;
  align-items: flex-end;
  gap: calc(4 * @w);
  flex-shrink: 0;
}
.version-badge {
  position: relative;
  font-size: @font-xs;
  line-height: 1;
  color: var(--text-secondary, #909399);
  background: var(--bg-secondary, #f5f5f5);
  border-radius: @radius-sm;
  padding: calc(2 * @h) calc(5 * @w);
  margin-bottom: calc(2 * @h);
  white-space: nowrap;
}
.update-dot {
  position: absolute;
  top: calc(-3 * @w);
  right: calc(-3 * @w);
  width: calc(7 * @w);
  height: calc(7 * @w);
  border-radius: 50%;
  background: #f56c6c;
  border: 1px solid var(--bg-primary, #ffffff);
}
.search-input {
  flex: 1;
  max-width: calc(400 * @w);
}
.spacer {
  flex: 1;
}
.shelf-main {
  padding: @gap-lg;
}
.empty-state {
  margin-top: @empty-margin-top;
}
.book-col {
  margin-bottom: @gap-lg;
}
.pagination-wrapper {
  margin-top: @gap-lg;
  text-align: center;
}
</style>
