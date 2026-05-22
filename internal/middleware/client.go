package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"gx-update-server/internal/config"
	"gx-update-server/internal/models"
	"gx-update-server/internal/security"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ProjectContextKey = "project"

func ClientAuth(database *gorm.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		appID := c.GetHeader("X-App-Id")
		timestamp := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")
		if appID == "" || timestamp == "" || nonce == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing client auth headers"})
			return
		}

		var project models.Project
		if err := database.Where("app_id = ? AND enabled = ?", appID, true).First(&project).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unknown app"})
			return
		}

		if err := validateTimestamp(timestamp, cfg.ClientClockSkew); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if !security.VerifyRequestHMAC(project.ClientSecret, c.Request.Method, c.Request.URL.Path, timestamp, nonce, body, signature) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bad request signature"})
			return
		}

		expires := time.Now().Add(cfg.ClientClockSkew)
		_ = database.Where("expires_at < ?", time.Now()).Delete(&models.RequestNonce{}).Error
		if err := database.Create(&models.RequestNonce{ProjectID: project.ID, Nonce: nonce, ExpiresAt: expires}).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "replayed nonce"})
			return
		}

		c.Set(ProjectContextKey, &project)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	}
}

func validateTimestamp(value string, skew time.Duration) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return errors.New("bad timestamp")
	}
	now := time.Now()
	t := time.Unix(n, 0)
	if t.Before(now.Add(-skew)) || t.After(now.Add(skew)) {
		return errors.New("timestamp outside allowed window")
	}
	return nil
}

func CurrentProject(c *gin.Context) (*models.Project, bool) {
	v, ok := c.Get(ProjectContextKey)
	if !ok {
		return nil, false
	}
	p, ok := v.(*models.Project)
	return p, ok
}
