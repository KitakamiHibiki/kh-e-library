package config

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

//go:embed application.yml
var defaultConfigYAML embed.FS

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type StorageConfig struct {
	Driver string            `yaml:"driver"`
	Local  LocalStorageConfig `yaml:"local"`
	Baidu  BaiduStorageConfig `yaml:"baidu"`
}

type LocalStorageConfig struct {
	BooksDir string `yaml:"books_dir"`
}

type BaiduStorageConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RefreshToken string `yaml:"refresh_token"`
}

func Load() *Config {
	cfg := loadEmbeddedDefaults()

	yamlPath := getEnv("CONFIG_PATH", "application.yml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("warning: failed to parse %s: %v", yamlPath, err)
		}
	}

	applyEnvOverrides(cfg)

	cfg.Database.Path = expandHome(cfg.Database.Path)
	cfg.Storage.Local.BooksDir = expandHome(cfg.Storage.Local.BooksDir)

	return cfg
}

func loadEmbeddedDefaults() *Config {
	data, err := defaultConfigYAML.ReadFile("application.yml")
	if err != nil {
		log.Fatalf("failed to read embedded config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse embedded config: %v", err)
	}
	return &cfg
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = i
		}
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("STORAGE_DRIVER"); v != "" {
		cfg.Storage.Driver = v
	}
	if v := os.Getenv("STORAGE_LOCAL_BOOKS_DIR"); v != "" {
		cfg.Storage.Local.BooksDir = v
	}
	if v := os.Getenv("BAIDU_CLIENT_ID"); v != "" {
		cfg.Storage.Baidu.ClientID = v
	}
	if v := os.Getenv("BAIDU_CLIENT_SECRET"); v != "" {
		cfg.Storage.Baidu.ClientSecret = v
	}
	if v := os.Getenv("BAIDU_REFRESH_TOKEN"); v != "" {
		cfg.Storage.Baidu.RefreshToken = v
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	return filepath.Join(home, path[2:])
}