package db

import (
	"os"
	"path/filepath"

	"gx-update-server/internal/config"
	"gx-update-server/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
		return nil, err
	}

	database, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	if err := database.AutoMigrate(
		&models.Project{},
		&models.Release{},
		&models.Package{},
		&models.VersionPolicy{},
		&models.BlockedVersion{},
		&models.RequestNonce{},
	); err != nil {
		return nil, err
	}
	return database, nil
}
