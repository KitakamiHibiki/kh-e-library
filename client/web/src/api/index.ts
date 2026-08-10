import axios from 'axios'
import type { ApiResponse, Book, ReadingProgress, Tag, StatsOverview, UpdateCheckResult, SystemStatus, DownloadUpdateResult } from '@/types/book'

const api = axios.create({
  baseURL: '/api/v1',
})

/* Books */
export const getBooks = (page = 1, pageSize = 20, keyword = '', tag = '', bookStatus = '', sortField = '', sortOrder = '') =>
  api.get<ApiResponse<{ list: Book[]; total: number; page: number }>>('/books/list', {
    params: { page, page_size: pageSize, keyword, tag, book_status: bookStatus, sort_field: sortField, sort_order: sortOrder },
  })

export const getBook = (id: number) =>
  api.get<ApiResponse<Book>>('/books/detail', { params: { id } })

export const uploadBook = (file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return api.post<ApiResponse<Book>>('/books/create', fd)
}

export const updateBook = (id: number, data: { title?: string; author?: string; publisher?: string; read_status?: string }) =>
  api.post<ApiResponse<null>>('/books/update', data, { params: { id } })

export const reprocessBook = (id: number) =>
  api.post<ApiResponse<null>>('/books/reprocess', null, { params: { id } })

export const deleteBook = (id: number) =>
  api.post<ApiResponse<null>>('/books/delete', null, { params: { id } })

export const batchDeleteBooks = (ids: number[]) =>
  api.post<ApiResponse<{ deleted: number }>>('/books/batch_delete', { ids })

export const getReadUrl = (id: number) => `/api/v1/books/read?id=${id}`
export const getDownloadUrl = (id: number) => `/api/v1/books/download?id=${id}`
export const getCoverUrl = (id: number) => `/api/v1/books/cover?id=${id}`

/* Progress */
export const getProgress = (id: number) =>
  api.get<ApiResponse<ReadingProgress>>('/books/progress', { params: { id } })

export const saveProgress = (data: { book_id: number; progress: number; cfi?: string; chapter_href?: string }) =>
  api.post<ApiResponse<ReadingProgress>>('/books/progress/save', data)

/* Tags */
export const getTags = (keyword = '') =>
  api.get<ApiResponse<Tag[]>>('/tags/list', { params: { keyword } })

export const createTag = (name: string) =>
  api.post<ApiResponse<Tag>>('/tags/create', { name })

export const updateTag = (id: number, name: string) =>
  api.post<ApiResponse<null>>('/tags/update', { name }, { params: { id } })

export const deleteTag = (id: number) =>
  api.post<ApiResponse<null>>('/tags/delete', null, { params: { id } })

/* Book-Tag associations */
export const addBookTag = (bookId: number, tagId: number) =>
  api.post<ApiResponse<null>>('/books/tags/add', { book_id: bookId, tag_id: tagId })

export const removeBookTag = (bookId: number, tagId: number) =>
  api.post<ApiResponse<null>>('/books/tags/remove', { book_id: bookId, tag_id: tagId })

/* Settings */
export const getSettings = () =>
  api.get<ApiResponse<Record<string, string>>>('/settings/list')

export const updateSettings = (settings: Record<string, string>) =>
  api.post<ApiResponse<null>>('/settings/update', { settings })

/* Stats */
export const getStatsOverview = () =>
  api.get<ApiResponse<StatsOverview>>('/stats/overview')

/* System / Software update */
export const getSystemStatus = () =>
  api.get<ApiResponse<SystemStatus>>('/system/status')

export const checkUpdate = (githubRepo: string) =>
  api.get<ApiResponse<UpdateCheckResult>>('/system/check-update', { params: { repo: githubRepo } })

export const downloadUpdate = (downloadUrl: string) =>
  api.post<ApiResponse<DownloadUpdateResult>>('/system/download-update', { download_url: downloadUrl })

export const installUpdate = () =>
  api.post<ApiResponse<{ status: string; new_file: string }>>('/system/install-update')

export default api
