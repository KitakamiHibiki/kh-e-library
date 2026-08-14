<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Book } from '@/types/book'
import { getCoverUrl } from '@/api'
import { useI18n } from 'vue-i18n'
import { CaretBottom, MoreFilled } from '@element-plus/icons-vue'
import BookCardMenu from './BookCardMenu.vue'
import { useCardDropdownClose } from '@/composables/useCardDropdownClose'

const props = defineProps<{
  book: Book
}>()

const emit = defineEmits<{
  read: [id: number]
  delete: [id: number]
  reprocess: [id: number]
  updateReadStatus: [id: number, status: string]
  bookInfo: [id: number]
  addToShelf: [id: number]
  exportBook: [id: number]
  contextmenu: [payload: { id: number; x: number; y: number }]
}>()

const { t } = useI18n()

// Right-click on the card opens the book action menu at the cursor position.
// The menu itself lives in BookShelf as one shared instance, so only a single
// context menu can ever be open.
const openContextMenu = (e: MouseEvent) => {
  emit('contextmenu', { id: props.book.id, x: e.clientX, y: e.clientY })
}

// Close the card's dropdowns when the user right-clicks elsewhere (left-clicks
// already close them natively; right-clicks do not fire a "click" event).
const statusDropdown = ref()
const moreDropdown = ref()
let unregisterStatus: (() => void) | null = null
let unregisterMore: (() => void) | null = null

onMounted(() => {
  const { register } = useCardDropdownClose()
  unregisterStatus = register(() => statusDropdown.value?.handleClose?.())
  unregisterMore = register(() => moreDropdown.value?.handleClose?.())
})

onBeforeUnmount(() => {
  unregisterStatus?.()
  unregisterMore?.()
})

const statusLabel = computed(() => {
  switch (props.book.read_status) {
    case 'reading':
      return t('bookShelf.reading')
    case 'finished':
      return t('bookShelf.finished')
    default:
      return t('bookShelf.unread')
  }
})

const coverError = (e: Event) => {
  ;(e.target as HTMLImageElement).style.display = 'none'
}

const readStatusOptions = [
  { label: () => t('bookShelf.unread'), value: 'unread' },
  { label: () => t('bookShelf.reading'), value: 'reading' },
  { label: () => t('bookShelf.finished'), value: 'finished' },
]

const handleStatusChange = (status: string) => {
  emit('updateReadStatus', props.book.id, status)
}

const handleDropdownCommand = (command: string) => {
  switch (command) {
    case 'read':
      emit('read', props.book.id)
      break
    case 'bookInfo':
      emit('bookInfo', props.book.id)
      break
    case 'markFinished':
      emit('updateReadStatus', props.book.id, 'finished')
      break
    case 'markUnread':
      emit('updateReadStatus', props.book.id, 'unread')
      break
    case 'delete':
      emit('delete', props.book.id)
      break
    case 'addToShelf':
      emit('addToShelf', props.book.id)
      break
    case 'export':
      emit('exportBook', props.book.id)
      break
  }
}
</script>

<template>
  <el-card :body-style="{ padding: '12px' }" shadow="hover" class="book-card" @click="emit('read', book.id)" @contextmenu.prevent="openContextMenu">
    <div class="book-cover">
      <img v-if="book.cover" :src="getCoverUrl(book.id)" @error="coverError" class="cover-img" />
      <span v-else-if="book.file_type === 'pdf'" class="cover-badge pdf-badge">PDF</span>
      <span v-else class="cover-badge epub-badge">+</span>
      <!-- File type tag on cover -->
      <span class="file-type-tag">{{ book.file_type?.toUpperCase() }}</span>
      <!-- Processing overlay -->
      <div v-if="book.book_status === 'processing'" class="status-overlay">
        <el-icon class="is-loading"><i class="el-icon-loading" /></el-icon>
        <span>{{ t('bookShelf.processing') }}</span>
      </div>
      <!-- Failed overlay -->
      <div v-if="book.book_status === 'failed'" class="status-overlay failed">
        <span>{{ t('bookShelf.failed') }}</span>
      </div>
    </div>
    <h4 class="book-title">{{ book.title }}</h4>
    <p class="book-author">{{ book.author || t('bookShelf.unknownAuthor') }}</p>
    <div class="book-actions" @click.stop>
      <el-dropdown ref="statusDropdown" trigger="click" @command="handleStatusChange">
        <span class="read-status" :class="`read-status--${book.read_status}`">
          <span class="read-status-dot"></span>
          <span class="read-status-text">{{ statusLabel }}</span>
          <el-icon class="read-status-caret"><CaretBottom /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-for="opt in readStatusOptions" :key="opt.value" :command="opt.value" :disabled="opt.value === book.read_status">
              {{ opt.label() }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button
        v-if="book.book_status === 'failed'"
        size="small"
        type="warning"
        plain
        @click.stop="emit('reprocess', book.id)"
      >{{ t('bookShelf.reprocess') }}</el-button>
      <span class="spacer"></span>
      <el-dropdown ref="moreDropdown" trigger="click" @command="handleDropdownCommand">
        <el-button size="small" circle class="more-btn">
          <el-icon><MoreFilled /></el-icon>
        </el-button>
        <template #dropdown>
          <BookCardMenu :book="book" />
        </template>
      </el-dropdown>
    </div>
  </el-card>
</template>

<style scoped lang="less">
@import '../styles/variables.less';

.book-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  height: 100%;
  box-sizing: border-box;
}
.book-card:hover {
  transform: translateY(-2 * @h);
}
:deep(.el-card__body) {
  display: flex;
  flex-direction: column;
}
.book-cover {
  aspect-ratio: 3/4;
  background: var(--bg-secondary, #f5f5f5);
  border-radius: @cover-radius;
  margin-bottom: @cover-margin-bottom;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  flex-shrink: 0;
}
.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-badge {
  font-size: 2rem;
  color: var(--text-secondary, #999);
}
.pdf-badge {
  color: #e74c3c;
  font-size: @font-md;
  font-weight: bold;
}
.file-type-tag {
  position: absolute;
  left: 0;
  bottom: 0;
  background: #e74c3c;
  color: #fff;
  font-size: @font-xs;
  font-weight: 600;
  padding: 1 * @h 6 * @w;
  border-radius: 0 @radius-sm 0 @radius-sm;
  line-height: 1.4;
  letter-spacing: 0.5px;
}
.status-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: @font-md;
  gap: @gap-xs;
}
.status-overlay.failed {
  background: rgba(231, 76, 60, 0.6);
}
.book-title {
  margin: 0 0 @gap-xs;
  font-size: @font-md;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.book-author {
  margin: 0;
  color: var(--text-secondary, #999);
  font-size: @font-sm;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.book-actions {
  margin-top: auto;
  padding-top: @actions-margin-top;
  display: flex;
  align-items: center;
  gap: @gap-xs;
  flex-wrap: wrap;
}
.read-status {
  display: inline-flex;
  align-items: center;
  gap: calc(5 * @w);
  cursor: pointer;
  font-size: @font-sm;
  line-height: 1;
  padding: calc(5 * @h) calc(8 * @w);
  border-radius: @radius-sm;
  transition: background 0.15s;
  user-select: none;

  &.read-status--unread {
    color: var(--text-secondary, #909399);
    .read-status-dot { background: #909399; }
  }

  &.read-status--reading {
    color: #409eff;
    .read-status-dot { background: #409eff; }
    .read-status-text { font-weight: 500; }
  }

  &.read-status--finished {
    color: #67c23a;
    .read-status-dot { background: #67c23a; }
  }

  &:hover {
    background: var(--bg-secondary, #f5f5f5);
  }
}

.read-status-dot {
  width: calc(8 * @w);
  height: calc(8 * @w);
  border-radius: 50%;
  flex-shrink: 0;
}

.read-status-caret {
  font-size: 0.6rem;
  color: currentColor;
  opacity: 0.6;
}
.more-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary, #999);
  padding: 0;
  width: @more-btn-size;
  height: @more-btn-size;
}
.more-btn:hover {
  color: var(--text-primary, #303133);
  background: var(--bg-secondary, #f5f5f5);
}
.spacer {
  flex: 1;
}
</style>
