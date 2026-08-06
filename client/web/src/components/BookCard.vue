<script setup lang="ts">
import type { Book } from '@/types/book'
import { getCoverUrl } from '@/api'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  book: Book
}>()

const emit = defineEmits<{
  read: [id: number]
  delete: [id: number]
  reprocess: [id: number]
  updateReadStatus: [id: number, status: string]
}>()

const { t } = useI18n()

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
</script>

<template>
  <el-card :body-style="{ padding: '12px' }" shadow="hover" class="book-card" @click="emit('read', book.id)">
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
    <div class="book-tags" v-if="book.tags && book.tags.length">
      <el-tag v-for="tag in book.tags.slice(0, 3)" :key="tag" size="small" class="book-tag">{{ tag }}</el-tag>
      <el-tag v-if="book.tags.length > 3" size="small" type="info">+{{ book.tags.length - 3 }}</el-tag>
    </div>
    <div class="book-actions">
      <el-dropdown trigger="click" @command="handleStatusChange" @click.stop>
        <el-tag
          :type="book.read_status === 'reading' ? 'warning' : book.read_status === 'finished' ? 'success' : 'info'"
          size="small"
          class="status-tag"
        >
          {{ book.read_status === 'reading' ? t('bookShelf.reading') : book.read_status === 'finished' ? t('bookShelf.finished') : t('bookShelf.unread') }}
          <el-icon class="el-icon--right"><i class="el-icon-arrow-down" /></el-icon>
        </el-tag>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-for="opt in readStatusOptions" :key="opt.value" :command="opt.value" :disabled="opt.value === book.read_status">
              {{ opt.label() }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <span class="spacer"></span>
      <el-button
        v-if="book.book_status === 'failed'"
        size="small"
        type="warning"
        plain
        @click.stop="emit('reprocess', book.id)"
      >{{ t('bookShelf.reprocess') }}</el-button>
      <el-button
        v-if="book.book_status === 'ready'"
        size="small"
        @click.stop="emit('read', book.id)"
      >{{ t('bookShelf.readButton') }}</el-button>
      <el-button size="small" type="danger" plain @click.stop="emit('delete', book.id)">{{ t('bookShelf.deleteButton') }}</el-button>
    </div>
  </el-card>
</template>

<style scoped>
.book-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}
.book-card:hover {
  transform: translateY(-2px);
}
.book-cover {
  aspect-ratio: 3/4;
  background: var(--bg-secondary, #f5f5f5);
  border-radius: 4px;
  margin-bottom: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
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
  font-size: 0.85rem;
  font-weight: bold;
}
.file-type-tag {
  position: absolute;
  left: 0;
  bottom: 0;
  background: #e74c3c;
  color: #fff;
  font-size: 0.65rem;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 0 4px 0 4px;
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
  font-size: 0.8rem;
  gap: 4px;
}
.status-overlay.failed {
  background: rgba(231, 76, 60, 0.6);
}
.book-title {
  margin: 0 0 4px;
  font-size: 0.85rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.book-author {
  margin: 0;
  color: var(--text-secondary, #999);
  font-size: 0.75rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.book-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}
.book-tag {
  font-size: 0.65rem;
}
.book-actions {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.status-tag {
  cursor: pointer;
}
.spacer {
  flex: 1;
}
</style>
