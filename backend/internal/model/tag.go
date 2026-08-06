package model

// Tag represents a label that can be associated with books.
type Tag struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"uniqueIndex;not null;size:64"`
}

// TagWithCount is a Tag with the number of associated books.
type TagWithCount struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}
