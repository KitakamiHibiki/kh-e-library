export interface Book {
  id: number
  title: string
  author: string
  publisher?: string
  isbn?: string
  cover: string          // filename, e.g. "cover.jpg"; empty string = no cover
  file: string           // filename, e.g. "book.epub"
  file_type: string      // "epub" | "pdf"
  file_size: number
  file_hash: string
  description?: string
  language?: string
  tags: string[]         // array of tag name strings
  pages?: number
  storage_key: string
  read_status: string    // "unread" | "reading" | "finished"
  book_status: string    // "processing" | "failed" | "ready" | "deleted"
  created_at: number     // Unix timestamp (seconds)
  updated_at: number     // Unix timestamp (seconds)
}

export interface ReadingProgress {
  id: number
  book_id: number
  progress: number       // 0.0 ~ 1.0
  cfi?: string           // EPUB CFI / PDF page number string
  chapter_href?: string
  created_at: number
  updated_at: number
}

export interface Tag {
  id: number
  name: string
  count: number          // number of associated books
}

export interface StatsOverview {
  total_books: number
  unread_count: number
  reading_count: number
  finished_count: number
  recent_reading: Array<{
    id: number
    title: string
    updated_at: number
  }>
}

export interface ApiResponse<T> {
  code: number
  data: T
  msg: string
}

export interface UpdateCheckResult {
  current_version: string
  latest_version: string
  has_update: boolean
  release_notes: string
  download_url: string
  file_name: string
  file_size: number
  published_at: string
  release_url: string
}

export interface SystemStatus {
  version: string
  platform: string
  executable: string
}

export interface DownloadUpdateResult {
  status: string
  file_path: string
  file_size: number
}
