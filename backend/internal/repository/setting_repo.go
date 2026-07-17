package repository

import "github.com/kitakami-hibiki/e-library/internal/model"

type SettingRepository struct{}

func NewSettingRepository() *SettingRepository {
	return &SettingRepository{}
}

func (r *SettingRepository) GetAll() ([]model.Setting, error) {
	var settings []model.Setting
	if err := DB.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// GetMap returns all settings merged with defaults as a flat map.
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

func (r *SettingRepository) GetByKey(key string) (*model.Setting, error) {
	var s model.Setting
	if err := DB.Where("key = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingRepository) Upsert(key, value string) error {
	return DB.Where("key = ?", key).Assign(model.Setting{Value: value}).FirstOrCreate(&model.Setting{}).Error
}

func (r *SettingRepository) BatchUpsert(settings map[string]string) error {
	tx := DB.Begin()
	for k, v := range settings {
		if err := tx.Where("key = ?", k).Assign(model.Setting{Value: v}).FirstOrCreate(&model.Setting{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}