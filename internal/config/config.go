package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Addr                string `yaml:"addr"`
	DBPath              string `yaml:"db_path"`
	StorageDir          string `yaml:"storage_dir"`
	PublicBaseURL       string `yaml:"public_base_url"`
	AppsConfigPath      string `yaml:"apps_config_path"`
	DownloadTokenSecret string `yaml:"download_token_secret"`
	ClientClockSkewSec  int    `yaml:"client_clock_skew_seconds"`
	DownloadTokenTTLSec int    `yaml:"download_token_ttl_seconds"`
}

type Config struct {
	Addr                string
	DBPath              string
	StorageDir          string
	PublicBaseURL       string
	AppsConfigPath      string
	DownloadTokenSecret string
	ClientClockSkew     time.Duration
	DownloadTokenTTL    time.Duration
}

func Load() Config {
	path := getenv("UPDATE_SYSTEM_CONFIG", "data/config/system.yaml")
	fc := loadFileDefaults(path)

	return Config{
		Addr:                getenv("UPDATE_SERVER_ADDR", fc.Addr),
		DBPath:              getenv("UPDATE_DB_PATH", fc.DBPath),
		StorageDir:          getenv("UPDATE_STORAGE_DIR", fc.StorageDir),
		PublicBaseURL:       strings.TrimRight(getenv("UPDATE_PUBLIC_BASE_URL", fc.PublicBaseURL), "/"),
		AppsConfigPath:      getenv("UPDATE_APPS_CONFIG", fc.AppsConfigPath),
		DownloadTokenSecret: getenv("UPDATE_DOWNLOAD_TOKEN_SECRET", fc.DownloadTokenSecret),
		ClientClockSkew:     time.Duration(getenvInt("UPDATE_CLIENT_CLOCK_SKEW_SECONDS", fc.ClientClockSkewSec)) * time.Second,
		DownloadTokenTTL:    time.Duration(getenvInt("UPDATE_DOWNLOAD_TOKEN_TTL_SECONDS", fc.DownloadTokenTTLSec)) * time.Second,
	}
}

func loadFileDefaults(path string) fileConfig {
	fc := fileConfig{
		Addr:                ":8080",
		DBPath:              "data/update.db",
		StorageDir:          "storage",
		PublicBaseURL:       "http://127.0.0.1:8080",
		AppsConfigPath:      "data/config/apps.yaml",
		DownloadTokenSecret: "change-me-download-token-secret",
		ClientClockSkewSec:  300,
		DownloadTokenTTLSec: 900,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fc
	}
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fc
	}
	return fc
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
