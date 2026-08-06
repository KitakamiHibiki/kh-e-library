package model

// StatsOverview contains reading statistics.
type StatsOverview struct {
	TotalBooks    int64              `json:"total_books"`
	UnreadCount   int64              `json:"unread_count"`
	ReadingCount  int64              `json:"reading_count"`
	FinishedCount int64              `json:"finished_count"`
	RecentReading []RecentReadingItem `json:"recent_reading"`
}

// RecentReadingItem is a lightweight struct for the recent reading list.
type RecentReadingItem struct {
	ID        uint  `json:"id"`
	Title     string `json:"title"`
	UpdatedAt int64  `json:"updated_at"`
}

// PDFMetadata holds extracted PDF metadata.
type PDFMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Pages       int    `json:"pages"`
}
