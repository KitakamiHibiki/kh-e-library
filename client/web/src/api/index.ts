import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
})

export interface PageResult<T> {
  data: T[]
  total: number
  page: number
}

/* Books */
export const getBooks = (page = 1, pageSize = 20, keyword = '') =>
  api.get('/books', { params: { page, page_size: pageSize, keyword } })

export const getBook = (id: number) =>
  api.get('/books/' + id)

export const uploadBook = (file: File) => {
  const fd = new FormData()
  fd.append('file', file)
  return api.post('/books', fd)
}

export const updateBook = (id: number, data: any) =>
  api.put('/books/' + id, data)

export const deleteBook = (id: number) =>
  api.delete('/books/' + id)

export const getReadUrl = (id: number) =>
  '/api/v1/books/' + id + '/read'

export const getCoverUrl = (id: number) =>
  '/api/v1/books/' + id + '/cover'

/* Progress */
export const getProgress = (bookId: number) =>
  api.get('/books/' + bookId + '/progress')

export const saveProgress = (data: { book_id: number; progress: number; cfi?: string; chapter_href?: string }) =>
  api.put('/books/' + data.book_id + '/progress', data)

/* Bookmarks */
export const getBookmarks = (bookId: number) =>
  api.get('/books/' + bookId + '/bookmarks')

export const createBookmark = (data: any) =>
  api.post('/books/' + data.book_id + '/bookmarks', data)

export const deleteBookmark = (id: number) =>
  api.delete('/bookmarks/' + id)

export default api
