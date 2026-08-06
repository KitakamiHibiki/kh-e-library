package repository

import (
	"github.com/kitakami-hibiki/e-library/internal/model"
	"gorm.io/gorm"
)

// SettingRepository provides data access for runtime settings.
type SettingRepository struct {
	db *gorm.DB
}

// NewSettingRepository creates a new SettingRepository.
func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

// GetAll returns all settings from the database.
func (r *SettingRepository) GetAll() ([]model.Setting, error) {
	var settings []model.Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// GetMap returns all settings merged with defaults as a flat map.
// Database values override defaults; all predefined keys are guaranteed to have a value.
func (r *SettingRepository) GetMap() (map[string]string, error) {
	settings, err := r.GetAll()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(model.DefaultSettings))
	for k, v := range model.DefaultSettings {
		m[k] = v
	}
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	return m, nil
}

// GetWithDefault returns the value for a key, falling back to the default if not found.
func (r *SettingRepository) GetWithDefault(key string) string {
	var s model.Setting
	if err := r.db.Where("key = ?", key).First(&s).Error; err != nil {
		if val, ok := model.DefaultSettings[key]; ok {
			return val
		}
		return ""
	}
	return s.Value
}

// GetByKey returns a single setting by key.
func (r *SettingRepository) GetByKey(key string) (*model.Setting, error) {
	var s model.Setting
	if err := r.db.Where("key = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// BatchUpsert saves multiple settings using SQLite UPSERT.
// Uses raw SQL INSERT ... ON CONFLICT for atomicity.
func (r *SettingRepository) BatchUpsert(settings map[string]string) error {
	tx := r.db.Begin()
	for k, v := range settings {
		now := model.NowUnix()
		if err := tx.Exec(`
			INSERT INTO settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET
				value = EXCLUDED.value,
				updated_at = EXCLUDED.updated_at
		`, k, v, now).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// BatchRollback restores specific keys to their old values.
// Used when the onSettings callback fails after a successful DB write.
func (r *SettingRepository) BatchRollback(oldValues map[string]string) error {
	tx := r.db.Begin()
	for k, v := range oldValues {
		now := model.NowUnix()
		if err := tx.Model(&model.Setting{}).Where("key = ?", k).
			Updates(map[string]interface{}{"value": v, "updated_at": now}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
