package repository

import (
	"github.com/kitakami-hibiki/e-library/internal/model"
	"gorm.io/gorm"
)

// TagRepository provides data access for tags.
type TagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository.
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

// List returns all tags with their associated book count.
// If keyword is provided, filters by tag name (case-insensitive LIKE).
func (r *TagRepository) List(keyword string) ([]model.TagWithCount, error) {
	var tags []model.TagWithCount
	query := r.db.Table("tags").
		Select("tags.id, tags.name, COUNT(book_tags.id) as count").
		Joins("LEFT JOIN book_tags ON book_tags.tag_id = tags.id").
		Group("tags.id, tags.name")

	if keyword != "" {
		query = query.Where("tags.name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Order("tags.name ASC").Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GetByID returns a tag by ID.
func (r *TagRepository) GetByID(id uint) (*model.Tag, error) {
	var tag model.Tag
	if err := r.db.First(&tag, id).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetByName returns a tag by name.
func (r *TagRepository) GetByName(name string) (*model.Tag, error) {
	var tag model.Tag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// Create creates a new tag.
func (r *TagRepository) Create(name string) (*model.Tag, error) {
	tag := &model.Tag{Name: name}
	if err := r.db.Create(tag).Error; err != nil {
		return nil, err
	}
	return tag, nil
}

// Update renames a tag.
func (r *TagRepository) Update(id uint, name string) error {
	return r.db.Model(&model.Tag{}).Where("id = ?", id).Update("name", name).Error
}

// Delete removes a tag and all its book associations (cascade).
func (r *TagRepository) Delete(id uint) error {
	// Delete book-tag associations first
	r.db.Where("tag_id = ?", id).Delete(&model.BookTag{})
	// Then delete the tag itself
	return r.db.Delete(&model.Tag{}, id).Error
}
