<script setup lang="ts">
import type { Book } from '@/types/book'
import { useI18n } from 'vue-i18n'

// Shared menu body for the book card "more" button and the right-click context
// menu. Items dispatch their `command` to the enclosing el-dropdown via inject,
// so no emit forwarding is needed here.
defineProps<{
  book: Book
}>()

const { t } = useI18n()
</script>

<template>
  <el-dropdown-menu>
    <el-dropdown-item command="bookInfo">{{ t('bookCard.menuBookInfo') }}</el-dropdown-item>
    <el-dropdown-item command="read">{{ t('bookCard.menuRead') }}</el-dropdown-item>
    <el-dropdown-item :command="book.read_status === 'finished' ? 'markUnread' : 'markFinished'">
      {{ book.read_status === 'finished' ? t('bookCard.menuMarkUnread') : t('bookCard.menuMarkFinished') }}
    </el-dropdown-item>
    <el-dropdown-item command="addToShelf">{{ book.tags?.length ? t('bookCard.menuModifyShelf') : t('bookCard.menuAddToShelf') }}</el-dropdown-item>
    <el-dropdown-item command="export">{{ t('bookCard.menuExport') }}</el-dropdown-item>
    <el-dropdown-item command="delete" divided>{{ t('bookCard.menuDelete') }}</el-dropdown-item>
  </el-dropdown-menu>
</template>
