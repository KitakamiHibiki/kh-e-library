<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Reading, Collection, Plus } from '@element-plus/icons-vue'
import { getBooks, deleteBook, reprocessBook, getSettings, updateBook, getCoverUrl, getDownloadUrl, getTags, createTag, addBookTag, removeBookTag, getSystemStatus, checkUpdate, getUpdateStatus, startUpdate } from '@/api'
import type { Book, Tag, UpdateCheckResult } from '@/types/book'
import { useI18n } from 'vue-i18n'
import BookCard from '@/components/BookCard.vue'
import BookCardMenu from '@/components/BookCardMenu.vue'
import UploadBookDialog from '@/components/UploadBookDialog.vue'
import { useCardDropdownClose } from '@/composables/useCardDropdownClose'

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

// Sidebar navigation: '全部图书' shows every book; the '书架列表' sub-menu
// expands to the shelves (tags) and selecting one filters the grid by that tag.
// NoTagFilter is the special tag value meaning "books in no shelf" — must match
// repository.NoTagFilter in the backend.
const NoTagFilter = '__none__'
const activeTag = ref('')
const shelves = ref<Tag[]>([])
const shelvesLoading = ref(false)
// Count of books that belong to no shelf, shown on the "未放入书架" entry.
const noTagCount = ref(0)

// Menu highlight: the selected shelf, the special "no shelf" entry, or 'all'.
const activeMenu = computed(() => {
  if (activeTag.value === NoTagFilter) return 'shelf-null'
  if (activeTag.value) return `shelf:${activeTag.value}`
  return 'all'
})

// "创建书架" dialog state
const createShelfVisible = ref(false)
const newShelfName = ref('')
const createShelfLoading = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

// Shelf dialog state
const shelfDialogVisible = ref(false)
const shelfBookId = ref(0)
// Shelf dialog title: "修改书架" when the book already has shelves, else "添加到书架".
const shelfDialogTitle = computed(() => {
  const book = books.value.find(b => b.id === shelfBookId.value)
  const hasShelf = !!book?.tags?.length
  return t(hasShelf ? 'bookCard.menuModifyShelf' : 'bookCard.menuAddToShelf')
})
const shelfTags = ref<Tag[]>([])
const shelfTagSearch = ref('')
const shelfLoading = ref(false)
// Multi-select state for the shelf dialog; changes apply only on Save.
const selectedTagIds = ref<number[]>([])
const initialTagIds = ref<number[]>([])
const shelfSaving = ref(false)

// App version badge shown next to the title; hasUpdate drives the red dot.
// Clicking the badge opens a dialog with full update info.
const appVersion = ref('')
const hasUpdate = ref(false)
const updateInfo = ref<UpdateCheckResult | null>(null)
const versionDialogVisible = ref(false)
const versionChecking = ref(false)
// True while the async download+install task is running.
const updateRunning = ref(false)
let updatePollTimer: ReturnType<typeof setInterval> | null = null

const stopUpdatePoll = () => {
  if (updatePollTimer) {
    clearInterval(updatePollTimer)
    updatePollTimer = null
  }
}

// Upload dialog
const uploadDialogVisible = ref(false)

const displayVersion = computed(() => {
  const v = appVersion.value
  if (!v) return ''
  return /^\d/.test(v) ? `v${v}` : v
})

const formatFileSize = (bytes: number) => {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

const formatDate = (iso: string) => {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toString() === 'Invalid Date' ? '-' : d.toLocaleDateString()
}

const openVersionDialog = async () => {
  versionDialogVisible.value = true
  // Re-check if we don't already have a result (e.g. the silent check failed).
  if (!updateInfo.value) {
    versionChecking.value = true
    try {
      const res = await checkUpdate('')
      updateInfo.value = res.data.data
      hasUpdate.value = !!res.data.data.has_update
    } catch {
      updateInfo.value = null
    } finally {
      versionChecking.value = false
    }
  }
}

const fetchBooks = async () => {
  loading.value = true
  try {
    const res = await getBooks(page.value, pageSize.value, keyword.value, activeTag.value, '', sortField.value, sortOrder.value)
    books.value = res.data.data.list || []
    total.value = res.data.data.total
  } finally {
    loading.value = false
  }
}

const fetchShelves = async () => {
  shelvesLoading.value = true
  try {
    const [tagRes, noTagRes] = await Promise.all([
      getTags(),
      getBooks(1, 1, '', NoTagFilter, '', '', ''),
    ])
    shelves.value = tagRes.data.data
    noTagCount.value = noTagRes.data.data.total
  } catch {
    shelves.value = []
    noTagCount.value = 0
  } finally {
    shelvesLoading.value = false
  }
}

// 'all' clears any shelf filter; 'create-shelf' opens the create dialog;
// 'shelf-null' selects books in no shelf; 'shelf:<name>' selects a shelf.
const handleMenuSelect = (index: string) => {
  if (index === 'all') {
    if (activeTag.value) {
      activeTag.value = ''
      page.value = 1
      fetchBooks()
    }
  } else if (index === 'create-shelf') {
    newShelfName.value = ''
    createShelfVisible.value = true
  } else if (index === 'shelf-null') {
    selectShelf(NoTagFilter)
  } else if (index.startsWith('shelf:')) {
    selectShelf(index.slice('shelf:'.length))
  }
}

const handleCreateShelf = async () => {
  const name = newShelfName.value.trim()
  if (!name) {
    ElMessage.warning(t('bookShelf.createShelfNameRequired'))
    return
  }
  createShelfLoading.value = true
  try {
    await createTag(name)
    ElMessage.success(t('bookShelf.createShelfSuccess'))
    createShelfVisible.value = false
    newShelfName.value = ''
    await fetchShelves()
  } catch {
    ElMessage.error(t('bookShelf.createShelfFailed'))
  } finally {
    createShelfLoading.value = false
  }
}

const selectShelf = (name: string) => {
  activeTag.value = name
  page.value = 1
  fetchBooks()
}

const clearTagFilter = () => {
  activeTag.value = ''
  page.value = 1
  fetchBooks()
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
// offline, rate-limited) just leaves the red dot hidden. The full result is
// kept so the version dialog can show details without re-querying.
const checkForUpdate = async () => {
  try {
    const res = await checkUpdate('')
    updateInfo.value = res.data.data
    hasUpdate.value = !!res.data.data.has_update
  } catch {
    updateInfo.value = null
    hasUpdate.value = false
  }
}

// "开始更新" from the version dialog: confirm, then start the async
// download+install task on the backend and poll its status. The app
// auto-restarts after a successful install, so the dialog disappears on its own.
const handleStartUpdate = async () => {
  const info = updateInfo.value
  if (!info?.download_url) {
    ElMessage.warning(t('settings.update.noDownloadUrl'))
    return
  }
  try {
    await ElMessageBox.confirm(t('settings.update.installConfirm'), t('settings.update.startUpdate'), {
      confirmButtonText: t('settings.update.startUpdate'),
      cancelButtonText: t('settings.update.cancel'),
      type: 'warning',
    })
  } catch {
    return // user cancelled
  }
  updateRunning.value = true
  try {
    await startUpdate(info.download_url)
    // Poll the async task; the server restarts once it reaches "completed".
    updatePollTimer = setInterval(async () => {
      try {
        const res = await getUpdateStatus()
        if (res.data.data.state === 'completed') {
          stopUpdatePoll()
          updateRunning.value = false
          ElMessage.success(t('settings.update.updateCompleted'))
        } else if (res.data.data.state === 'failed') {
          stopUpdatePoll()
          updateRunning.value = false
          ElMessage.error(res.data.data.message || t('bookShelf.updateFailed'))
        }
      } catch {
        // Transient (server restarting / draining) — keep polling or give up.
      }
    }, 1000)
  } catch (e: unknown) {
    updateRunning.value = false
    const msg = (e as { response?: { data?: { msg?: string } } })?.response?.data?.msg
    ElMessage.error(msg || t('bookShelf.updateFailed'))
  }
}

const onSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  page.value = 1
  searchTimer = setTimeout(fetchBooks, 300)
}

const onUploaded = () => {
  ElMessage.success(t('bookShelf.uploadSuccess'))
  fetchBooks()
  fetchShelves()
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

// "图书信息" dialog: shows the clicked book's metadata straight from the list
// item (the list returns full Book objects, no extra request needed).
const bookInfoDialogVisible = ref(false)
const bookInfoBookId = ref(0)
const bookInfoBook = computed(() => books.value.find(b => b.id === bookInfoBookId.value) ?? null)

const handleBookInfo = (id: number) => {
  bookInfoBookId.value = id
  bookInfoDialogVisible.value = true
}

const statusLabelFor = (status: string) => {
  switch (status) {
    case 'reading':
      return t('bookShelf.reading')
    case 'finished':
      return t('bookShelf.finished')
    default:
      return t('bookShelf.unread')
  }
}

const formatDateTime = (ts: number) => {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toString() === 'Invalid Date' ? '-' : d.toLocaleString()
}

const handleExportBook = (id: number) => {
  window.open(getDownloadUrl(id), '_blank')
}

// Right-click context menu for book cards. A single shared, virtual-triggering
// dropdown anchored at the cursor — one instance means only one context menu can
// ever be open, and the dropdown's outside-click handling closes it on any other
// mouse interaction.
const ctxMenu = ref<{ id: number; x: number; y: number } | null>(null)
const ctxDropdown = ref()
// Close the shared context menu on right-clicks elsewhere (left-clicks already
// close it natively; right-clicks do not fire a "click" event).
const { register: registerDropdownClose } = useCardDropdownClose()
let unregisterCtx: (() => void) | null = null
const ctxVirtualRef = ref<{
  getBoundingClientRect: () => DOMRect
  contextElement?: Element
} | null>(null)
const ctxBook = computed(() => {
  const id = ctxMenu.value?.id
  return id ? books.value.find(b => b.id === id) ?? null : null
})

const handleContextMenu = async (payload: { id: number; x: number; y: number }) => {
  const { x, y } = payload
  ctxMenu.value = payload
  ctxVirtualRef.value = {
    getBoundingClientRect: () =>
      ({
        width: 0,
        height: 0,
        left: x,
        top: y,
        right: x,
        bottom: y,
        x,
        y,
        toJSON: () => ({}),
      } as DOMRect),
    contextElement: document.body,
  }
  // Let the dropdown's virtual reference update before opening.
  await nextTick()
  ctxDropdown.value?.handleOpen?.()
}

const handleCtxCommand = (command: string) => {
  const id = ctxMenu.value?.id
  if (!id) return
  switch (command) {
    case 'read':
      handleRead(id)
      break
    case 'bookInfo':
      handleBookInfo(id)
      break
    case 'markFinished':
      handleUpdateReadStatus(id, 'finished')
      break
    case 'markUnread':
      handleUpdateReadStatus(id, 'unread')
      break
    case 'addToShelf':
      openShelfDialog(id)
      break
    case 'export':
      handleExportBook(id)
      break
    case 'delete':
      handleDelete(id)
      break
  }
}

const handleCtxVisibleChange = (visible: boolean) => {
  if (!visible) ctxMenu.value = null
}

const openShelfDialog = async (id: number) => {
  shelfBookId.value = id
  shelfTagSearch.value = ''
  await fetchShelfTags()
  // Pre-select the book's current shelves (matched by tag name).
  const book = books.value.find(b => b.id === id)
  const names = new Set(book?.tags ?? [])
  const selected = shelfTags.value.filter(t => names.has(t.name)).map(t => t.id)
  selectedTagIds.value = selected
  initialTagIds.value = [...selected]
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

// Apply the multi-select changes: add newly checked shelves, remove unchecked ones.
const handleSaveShelf = async () => {
  const bookId = shelfBookId.value
  const initial = new Set(initialTagIds.value)
  const selected = new Set(selectedTagIds.value)
  const toAdd = selectedTagIds.value.filter(id => !initial.has(id))
  const toRemove = [...initial].filter(id => !selected.has(id))
  if (toAdd.length === 0 && toRemove.length === 0) {
    shelfDialogVisible.value = false
    return
  }
  shelfSaving.value = true
  try {
    for (const id of toAdd) await addBookTag(bookId, id)
    for (const id of toRemove) await removeBookTag(bookId, id)
    ElMessage.success(t('bookCard.addToShelfSuccess'))
    shelfDialogVisible.value = false
    fetchBooks()
    fetchShelves()
  } catch {
    ElMessage.error(t('bookCard.addToShelfFailed'))
  } finally {
    shelfSaving.value = false
  }
}

// Prompt for a new tag name, create the tag, and add it to the multi-select
// list. The book-tag association is only applied when the user presses Save.
const openCreateTag = async () => {
  let name = ''
  try {
    const res = await ElMessageBox.prompt(t('bookCard.newTagPlaceholder'), t('bookShelf.create'), {
      confirmButtonText: t('bookShelf.create'),
      cancelButtonText: t('bookShelf.cancel'),
      inputPlaceholder: t('bookCard.newTagPlaceholder'),
      inputValidator: (v: string) => (v && v.trim() ? true : t('bookShelf.createShelfNameRequired')),
    })
    name = res.value.trim()
  } catch {
    return // user cancelled
  }
  try {
    const tagRes = await createTag(name)
    shelfTags.value = [...shelfTags.value, tagRes.data.data]
    selectedTagIds.value = [...selectedTagIds.value, tagRes.data.data.id]
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
  fetchShelves()
  pollTimer = setInterval(pollProcessing, 2000)
  unregisterCtx = registerDropdownClose(() => ctxDropdown.value?.handleClose?.())
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
  stopUpdatePoll()
  unregisterCtx?.()
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
          :title="hasUpdate ? t('bookShelf.newVersionAvailable') : t('bookShelf.versionInfo')"
          @click="openVersionDialog"
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
      <el-button type="primary" @click="uploadDialogVisible = true">{{ t('bookShelf.uploadButton') }}</el-button>
      <el-button @click="$router.push('/settings')">{{ t('bookShelf.settingsButton') }}</el-button>
    </el-header>
    <el-container class="shelf-body">
      <el-aside class="shelf-sidebar">
        <el-menu
          class="shelf-menu"
          :default-active="activeMenu"
          :default-openeds="['shelves']"
          @select="handleMenuSelect"
        >
          <el-menu-item index="all">
            <el-icon><Reading /></el-icon>
            <span>{{ t('bookShelf.sidebarAll') }}</span>
          </el-menu-item>
          <el-sub-menu index="shelves">
            <template #title>
              <el-icon><Collection /></el-icon>
              <span>{{ t('bookShelf.sidebarShelfList') }}</span>
            </template>
            <el-menu-item index="create-shelf" class="shelf-create-item">
              <el-icon><Plus /></el-icon>
              <span>{{ t('bookShelf.createShelf') }}</span>
            </el-menu-item>
            <el-menu-item v-if="!shelvesLoading && shelves.length === 0" index="shelves-empty" disabled>
              {{ t('bookShelf.shelfEmpty') }}
            </el-menu-item>
            <el-menu-item
              v-for="shelf in shelves"
              :key="shelf.id"
              :index="`shelf:${shelf.name}`"
              class="shelf-item"
            >
              <span class="shelf-item-name" :title="shelf.name">{{ shelf.name }}</span>
              <span class="shelf-item-count">{{ shelf.count }}</span>
            </el-menu-item>
            <el-menu-item index="shelf-null" class="shelf-item shelf-null-item">
              <span class="shelf-item-name">{{ t('bookShelf.notInShelf') }}</span>
              <span class="shelf-item-count">{{ noTagCount }}</span>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-aside>
      <el-main class="shelf-main">
        <div v-if="activeTag" class="tag-filter">
          <el-tag closable type="primary" @close="clearTagFilter">
            <template v-if="activeTag === NoTagFilter">
              {{ t('bookShelf.notInShelf') }}
            </template>
            <template v-else>
              {{ t('bookShelf.filteredByTag', { name: activeTag }) }}
            </template>
          </el-tag>
        </div>
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
              @contextmenu="handleContextMenu"
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
    </el-container>

    <!-- Add to Shelf Dialog -->
    <el-dialog v-model="shelfDialogVisible" :title="shelfDialogTitle" width="480px">
      <el-input
        v-model="shelfTagSearch"
        :placeholder="t('bookCard.searchTags')"
        clearable
        @input="fetchShelfTags"
        style="margin-bottom: 12px"
      />
      <div v-loading="shelfLoading">
        <div v-if="shelfTags.length > 0" class="shelf-tag-list">
          <el-checkbox-group v-model="selectedTagIds">
            <div v-for="tag in shelfTags" :key="tag.id" class="shelf-tag-option">
              <el-checkbox :value="tag.id" class="shelf-tag-checkbox">{{ tag.name }} ({{ tag.count }})</el-checkbox>
            </div>
          </el-checkbox-group>
        </div>
        <el-empty v-else-if="!shelfLoading" :description="t('bookCard.noTags')" />
      </div>
      <template #footer>
        <div class="shelf-dialog-footer">
          <el-button plain @click="openCreateTag">{{ t('bookShelf.create') }}</el-button>
          <div class="shelf-dialog-footer-right">
            <el-button @click="shelfDialogVisible = false">{{ t('bookShelf.cancel') }}</el-button>
            <el-button type="primary" :loading="shelfSaving" @click="handleSaveShelf">{{ t('bookShelf.save') }}</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- Create Shelf Dialog -->
    <el-dialog v-model="createShelfVisible" :title="t('bookShelf.createShelf')" width="360px" @closed="newShelfName = ''">
      <el-input
        v-model="newShelfName"
        :placeholder="t('bookShelf.createShelfPlaceholder')"
        maxlength="64"
        clearable
        @keyup.enter="handleCreateShelf"
      />
      <template #footer>
        <el-button @click="createShelfVisible = false">{{ t('bookShelf.cancel') }}</el-button>
        <el-button type="primary" :loading="createShelfLoading" :disabled="!newShelfName.trim()" @click="handleCreateShelf">
          {{ t('bookShelf.create') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Version Info Dialog -->
    <el-dialog v-model="versionDialogVisible" :title="t('bookShelf.versionInfo')" width="460px">
      <div v-loading="versionChecking" class="version-dialog">
        <template v-if="updateInfo">
          <el-alert
            v-if="updateInfo.has_update"
            class="version-alert"
            type="success"
            :closable="false"
            show-icon
            :title="t('bookShelf.newVersionAvailable')"
          />
          <div class="version-row">
            <span class="version-label">{{ t('bookShelf.currentVersion') }}</span>
            <span class="version-value">{{ displayVersion || t('bookShelf.unknown') }}</span>
          </div>
          <template v-if="updateInfo.has_update">
            <div class="version-row">
              <span class="version-label">{{ t('bookShelf.latestVersion') }}</span>
              <span class="version-value">{{ updateInfo.latest_version }}</span>
            </div>
            <div class="version-row" v-if="updateInfo.published_at">
              <span class="version-label">{{ t('bookShelf.publishedAt') }}</span>
              <span class="version-value">{{ formatDate(updateInfo.published_at) }}</span>
            </div>
            <div class="version-row" v-if="updateInfo.file_size">
              <span class="version-label">{{ t('bookShelf.fileSize') }}</span>
              <span class="version-value">{{ formatFileSize(updateInfo.file_size) }}</span>
            </div>
            <div class="version-row" v-if="updateInfo.release_url || updateInfo.download_url">
              <span class="version-label">{{ t('bookShelf.downloadUrl') }}</span>
              <a class="version-value version-link" :href="updateInfo.release_url || updateInfo.download_url" target="_blank" rel="noopener">
                {{ updateInfo.release_url || updateInfo.download_url }}
              </a>
            </div>
            <div v-if="updateInfo.release_notes" class="version-notes">
              <div class="version-notes-title">{{ t('bookShelf.releaseNotes') }}</div>
              <pre>{{ updateInfo.release_notes }}</pre>
            </div>
          </template>
          <el-alert
            v-else
            class="version-alert"
            type="info"
            :closable="false"
            show-icon
            :title="t('bookShelf.upToDate')"
          />
        </template>
        <div v-else-if="!versionChecking" class="version-error">
          <el-alert type="warning" :closable="false" show-icon :title="t('bookShelf.checkUpdateFailed')" />
        </div>
      </div>
      <template #footer>
        <el-button
          v-if="updateInfo && updateInfo.has_update"
          type="primary"
          :loading="updateRunning"
          :disabled="!updateInfo.download_url"
          @click="handleStartUpdate"
        >
          {{ t('bookShelf.startUpdate') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Book Info Dialog -->
    <el-dialog v-model="bookInfoDialogVisible" :title="t('bookInfo.title')" width="520px" :append-to-body="true">
      <div v-if="bookInfoBook" class="book-info">
        <div class="book-info-header">
          <div class="book-info-cover">
            <img v-if="bookInfoBook.cover" :src="getCoverUrl(bookInfoBook.id)" class="book-info-cover-img" />
            <span v-else-if="bookInfoBook.file_type === 'pdf'" class="book-info-cover-badge pdf-badge">PDF</span>
            <span v-else class="book-info-cover-badge">+</span>
          </div>
          <div class="book-info-main">
            <h3 class="book-info-title">{{ bookInfoBook.title }}</h3>
            <p class="book-info-author">{{ bookInfoBook.author || t('bookShelf.unknownAuthor') }}</p>
            <div v-if="bookInfoBook.tags?.length" class="book-info-tags">
              <el-tag v-for="tag in bookInfoBook.tags" :key="tag" size="small">{{ tag }}</el-tag>
            </div>
          </div>
        </div>
        <div class="book-info-meta">
          <div class="book-info-row">
            <span class="book-info-label">{{ t('bookInfo.status') }}</span>
            <span class="book-info-value">
              <span class="book-info-status-dot" :class="`book-info-status-dot--${bookInfoBook.read_status}`"></span>
              {{ statusLabelFor(bookInfoBook.read_status) }}
            </span>
          </div>
          <div class="book-info-row">
            <span class="book-info-label">{{ t('bookInfo.fileType') }}</span>
            <span class="book-info-value">{{ bookInfoBook.file_type?.toUpperCase() }}</span>
          </div>
          <div class="book-info-row">
            <span class="book-info-label">{{ t('bookInfo.fileSize') }}</span>
            <span class="book-info-value">{{ formatFileSize(bookInfoBook.file_size) }}</span>
          </div>
          <div class="book-info-row">
            <span class="book-info-label">{{ bookInfoBook.file_type === 'pdf' ? t('bookInfo.pages') : t('bookInfo.chapterCount') }}</span>
            <span class="book-info-value">{{ bookInfoBook.pages || '-' }}</span>
          </div>
          <div class="book-info-row" v-if="bookInfoBook.isbn">
            <span class="book-info-label">{{ t('bookInfo.isbn') }}</span>
            <span class="book-info-value">{{ bookInfoBook.isbn }}</span>
          </div>
          <div class="book-info-row" v-if="bookInfoBook.publisher">
            <span class="book-info-label">{{ t('bookInfo.publisher') }}</span>
            <span class="book-info-value">{{ bookInfoBook.publisher }}</span>
          </div>
          <div class="book-info-row" v-if="bookInfoBook.language">
            <span class="book-info-label">{{ t('bookInfo.language') }}</span>
            <span class="book-info-value">{{ bookInfoBook.language }}</span>
          </div>
          <div class="book-info-row">
            <span class="book-info-label">{{ t('bookInfo.created') }}</span>
            <span class="book-info-value">{{ formatDateTime(bookInfoBook.created_at) }}</span>
          </div>
          <div class="book-info-row">
            <span class="book-info-label">{{ t('bookInfo.updated') }}</span>
            <span class="book-info-value">{{ formatDateTime(bookInfoBook.updated_at) }}</span>
          </div>
        </div>
        <div v-if="bookInfoBook.description" class="book-info-desc">
          <div class="book-info-label">{{ t('bookInfo.description') }}</div>
          <p class="book-info-desc-text">{{ bookInfoBook.description }}</p>
        </div>
      </div>
      <template #footer>
        <div class="book-info-footer">
          <el-button @click="bookInfoDialogVisible = false">{{ t('bookInfo.close') }}</el-button>
          <el-button @click="handleExportBook(bookInfoBookId)">{{ t('bookInfo.download') }}</el-button>
          <el-button type="primary" @click="handleRead(bookInfoBookId)">{{ t('bookInfo.read') }}</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Upload Book Dialog -->
    <UploadBookDialog v-model="uploadDialogVisible" @uploaded="onUploaded" />

    <!-- Shared right-click context menu for book cards, anchored at the cursor -->
    <el-dropdown
      v-if="ctxBook"
      ref="ctxDropdown"
      :virtual-triggering="true"
      :virtual-ref="ctxVirtualRef"
      placement="bottom-start"
      trigger="click"
      @command="handleCtxCommand"
      @visible-change="handleCtxVisibleChange"
    >
      <template #dropdown>
        <BookCardMenu :book="ctxBook" />
      </template>
    </el-dropdown>
  </el-container>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.shelf-container {
  min-height: 100vh;
}
.shelf-body {
  flex: 1;
  min-height: 0;
}
.shelf-sidebar {
  --el-aside-width: @sidebar-width;
  border-right: 1px solid var(--border-color, #e0e0e0);
}
.shelf-menu {
  border-right: none;
}
.shelf-create-item {
  color: var(--text-secondary, #909399);
}
.shelf-create-item .el-icon {
  margin-right: calc(6 * @w);
}
.shelf-null-item {
  margin-top: calc(4 * @h);
  border-top: 1px solid var(--border-color, #e0e0e0);
  color: var(--text-secondary, #909399);
}
.shelf-tag-list {
  max-height: 300px;
  overflow-y: auto;
}
.shelf-tag-option {
  margin-bottom: calc(4 * @h);
}
:deep(.shelf-tag-checkbox) {
  width: 100%;
  box-sizing: border-box;
  margin-right: 0;
  padding: calc(6 * @h) calc(10 * @w);
  border-radius: @radius-sm;
  cursor: pointer;
  transition: background 0.15s;
}
:deep(.shelf-tag-checkbox:hover) {
  background: var(--bg-secondary, #f5f5f5);
}
.shelf-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.shelf-dialog-footer-right {
  display: flex;
  gap: calc(8 * @w);
}
.shelf-item {
  display: flex;
  align-items: center;
  gap: calc(8 * @w);
  min-width: 0;
}
.shelf-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shelf-item-count {
  flex-shrink: 0;
  font-size: @font-xs;
  color: var(--text-secondary, #909399);
  background: var(--bg-secondary, #f5f5f5);
  border-radius: @radius-sm;
  padding: calc(1 * @h) calc(6 * @w);
  line-height: 1.4;
}
.shelf-main {
  padding: @gap-lg;
}
.tag-filter {
  margin-bottom: @gap-lg;
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
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.version-badge:hover {
  color: var(--text-primary, #303133);
  background: var(--border-color, #e0e0e0);
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
.empty-state {
  margin-top: @empty-margin-top;
}
.book-col {
  margin-bottom: @gap-lg;
  display: flex;
}
.pagination-wrapper {
  margin-top: @gap-lg;
  text-align: center;
}
.version-dialog {
  min-height: calc(60 * @h);
}
.version-row {
  display: flex;
  align-items: baseline;
  gap: @gap-md;
  padding: calc(4 * @h) 0;
  font-size: @font-sm;
}
.version-label {
  flex-shrink: 0;
  width: calc(80 * @w);
  color: var(--text-secondary, #909399);
}
.version-value {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  color: var(--text-primary, #303133);
}
.version-link {
  color: #409eff;
  text-decoration: none;
}
.version-link:hover {
  text-decoration: underline;
}
.version-notes {
  margin-top: @gap-sm;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: @radius-sm;
  padding: @gap-sm @gap-md;
  background: var(--bg-secondary, #f5f5f5);
}
.version-notes-title {
  font-weight: 600;
  margin-bottom: @gap-xs;
  font-size: @font-sm;
}
.version-notes pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: calc(180 * @h);
  overflow-y: auto;
  font-family: inherit;
  font-size: @font-sm;
  color: var(--text-primary, #303133);
}
.version-alert {
  margin-top: @gap-lg;
}
.version-error {
  margin-top: @gap-sm;
}

/* Book Info Dialog */
.book-info-header {
  display: flex;
  gap: @gap-lg;
  align-items: flex-start;
}
.book-info-cover {
  flex-shrink: 0;
  width: calc(96 * @w);
  aspect-ratio: 3/4;
  background: var(--bg-secondary, #f5f5f5);
  border-radius: @cover-radius;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}
.book-info-cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.book-info-cover-badge {
  font-size: 2rem;
  color: var(--text-secondary, #999);
}
.book-info-cover-badge.pdf-badge {
  color: #e74c3c;
  font-size: @font-md;
  font-weight: bold;
}
.book-info-main {
  flex: 1;
  min-width: 0;
}
.book-info-title {
  margin: 0 0 @gap-xs;
  font-size: @font-lg;
  word-break: break-word;
}
.book-info-author {
  margin: 0;
  color: var(--text-secondary, #909399);
  font-size: @font-sm;
}
.book-info-tags {
  display: flex;
  flex-wrap: wrap;
  gap: @tag-gap;
  margin-top: @gap-sm;
}
.book-info-meta {
  margin-top: @gap-lg;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: @gap-sm @gap-lg;
}
.book-info-row {
  display: flex;
  align-items: baseline;
  gap: @gap-xs;
  font-size: @font-sm;
  min-width: 0;
}
.book-info-label {
  flex-shrink: 0;
  width: calc(56 * @w);
  color: var(--text-secondary, #909399);
}
.book-info-value {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  display: flex;
  align-items: center;
  gap: calc(4 * @w);
  color: var(--text-primary, #303133);
}
.book-info-status-dot {
  width: calc(8 * @w);
  height: calc(8 * @w);
  border-radius: 50%;
  flex-shrink: 0;
}
.book-info-status-dot--unread {
  background: #909399;
}
.book-info-status-dot--reading {
  background: #409eff;
}
.book-info-status-dot--finished {
  background: #67c23a;
}
.book-info-desc {
  margin-top: @gap-lg;
  padding: @gap-sm @gap-md;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: @radius-sm;
  background: var(--bg-secondary, #f5f5f5);
}
.book-info-desc-text {
  margin: @gap-xs 0 0;
  font-size: @font-sm;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: calc(180 * @h);
  overflow-y: auto;
  color: var(--text-primary, #303133);
}
.book-info-footer {
  display: flex;
  justify-content: flex-end;
  gap: @gap-xs;
}
</style>
