package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"gx-update-server/internal/appconfig"
	"gx-update-server/internal/config"
	"gx-update-server/internal/db"
	"gx-update-server/internal/router"

	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open database failed: %v", err)
	}

	if err := reloadConfig(database, cfg.AppsConfigPath); err != nil {
		log.Fatalf("sync apps config failed: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		log.Printf("[server] type 'reload' and Enter to reload config")
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "reload" || line == "r" {
				if err := reloadConfig(database, cfg.AppsConfigPath); err != nil {
					log.Printf("[reload] failed: %v", err)
				} else {
					log.Printf("[reload] config reloaded successfully")
				}
			}
		}
	}()

	r := router.New(database, cfg)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func reloadConfig(database *gorm.DB, path string) error {
	apps, err := appconfig.Load(path)
	if err != nil {
		return err
	}
	return appconfig.SyncToDB(database, apps)
}
