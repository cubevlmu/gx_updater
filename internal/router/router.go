package router

import (
	"log"
	"net/http"

	"gx-update-server/internal/appconfig"
	"gx-update-server/internal/config"
	"gx-update-server/internal/handlers"
	"gx-update-server/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(database *gorm.DB, cfg config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/-/reload", func(c *gin.Context) {
		apps, err := appconfig.Load(cfg.AppsConfigPath)
		if err != nil {
			log.Printf("[reload] load config failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := appconfig.SyncToDB(database, apps); err != nil {
			log.Printf("[reload] sync config failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[reload] config reloaded successfully")
		c.JSON(http.StatusOK, gin.H{"status": "reloaded"})
	})

	publicHandler := handlers.NewPublicHandler(database, cfg)

	api := r.Group("/api/v1")
	api.POST("/check", middleware.ClientAuth(database, cfg), publicHandler.CheckUpdate)
	api.GET("/packages/:package_id/download", publicHandler.DownloadPackage)

	return r
}
