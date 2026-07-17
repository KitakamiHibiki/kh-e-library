package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Hardcoded data paths — not user-configurable.
const (
	DataDirRel       = "./data"
	BooksDirRel      = "./data/books"
	DatabaseFileRel  = "./data/library.db"
)

func DatabasePath() string { return resolveExeRelative(DatabaseFileRel) }
func DefaultBooksDir() string { return resolveExeRelative(BooksDirRel) }

func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 14325,
		},
	}

	yamlPath := getEnv("CONFIG_PATH", "application.yml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("warning: failed to parse %s: %v", yamlPath, err)
		}
	}

	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = i
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveExeRelative(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	exe, err := os.Executable()
	if err != nil {
		return path
	}
	return filepath.Join(filepath.Dir(exe), path)
}