export interface Book {
  id: number
  title: string
  author: string
  publisher?: string
  isbn?: string
  cover_path?: string
  file_path: string
  file_size: number
  description?: string
  language?: string
  tags?: string
  pages?: number
  storage_key: string
  created_at: string
  updated_at: string
}

export interface ReadingProgress {
  id: number
  book_id: number
  progress: number
  cfi?: string
  chapter_href?: string
  created_at: string
  updated_at: string
}

export interface Bookmark {
  id: number
  book_id: number
  cfi: string
  chapter_href?: string
  chapter_name?: string
  text?: string
  note?: string
  progress: number
  created_at: string
  updated_at: string
}