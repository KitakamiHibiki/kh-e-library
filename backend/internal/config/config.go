package config

import (
	"os"
	"path/filepath"
	"strconv"
)

const dataDir = "~/.kh/e-library"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Path string
}

type StorageConfig struct {
	Driver string // "local" | "baidu"
	Local  LocalStorageConfig
	Baidu  BaiduStorageConfig
}

type LocalStorageConfig struct {
	BooksDir string
}

type BaiduStorageConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 14325),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", expandHome(filepath.Join(dataDir, "library.db"))),
		},
		Storage: StorageConfig{
			Driver: getEnv("STORAGE_DRIVER", "local"),
			Local: LocalStorageConfig{
				BooksDir: getEnv("STORAGE_LOCAL_BOOKS_DIR", expandHome(filepath.Join(dataDir, "books"))),
			},
			Baidu: BaiduStorageConfig{
				ClientID:     getEnv("BAIDU_CLIENT_ID", ""),
				ClientSecret: getEnv("BAIDU_CLIENT_SECRET", ""),
				RefreshToken: getEnv("BAIDU_REFRESH_TOKEN", ""),
			},
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// expandHome 将路径开头的 "~" 替换为用户主目录。
// 在 Windows 上等价于 %USERPROFILE%。
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
	return filepath.Join(home, path[2:]) // 跳过 "~" 和分隔符
}

