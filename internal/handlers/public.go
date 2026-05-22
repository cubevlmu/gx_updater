package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"gx-update-server/internal/config"
	"gx-update-server/internal/middleware"
	"gx-update-server/internal/models"
	"gx-update-server/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PublicHandler struct {
	DB  *gorm.DB
	Cfg config.Config
}

func NewPublicHandler(database *gorm.DB, cfg config.Config) *PublicHandler {
	return &PublicHandler{DB: database, Cfg: cfg}
}

func (h *PublicHandler) CheckUpdate(c *gin.Context) {
	project, ok := middleware.CurrentProject(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
		return
	}

	var req CheckUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AppID != "" && req.AppID != project.AppID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id does not match X-App-Id"})
		return
	}
	normalizeCheckRequest(&req)

	log.Printf(
		"[update] check request app_id=%s version=%s version_code=%d platform=%s arch=%s channel=%s",
		req.AppID,
		req.Version,
		req.VersionCode,
		req.Platform,
		req.Arch,
		req.Channel,
	)

	legal := h.evaluateLegal(project.ID, req)
	log.Printf("[update] legal allowed=%v code=%s message=%s", legal.Allowed, legal.Code, legal.Message)

	latest, pkg, found := h.findLatestRelease(project.ID, req)
	if found {
		log.Printf("[update] found release version=%s code=%d platform=%s arch=%s channel=%s",
			latest.Version, latest.VersionCode, latest.Platform, latest.Arch, latest.Channel)
	} else {
		log.Printf("[update] no matching release found")
		h.logCandidateReleases(project.ID)
	}

	resp := CheckUpdateResponse{
		RequestID:  uuid.NewString(),
		ServerTime: time.Now().Unix(),
		Legal:      legal,
	}

	if found {
		info, err := h.buildUpdateInfo(project, req, latest, pkg, legal)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp.Update = &info
	} else {
		resp.Update = &UpdateInfo{Available: false, UpdateType: "none"}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PublicHandler) DownloadPackage(c *gin.Context) {
	var pkg models.Package
	if err := h.DB.First(&pkg, c.Param("package_id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	if err := security.VerifyDownloadToken(h.Cfg.DownloadTokenSecret, pkg.ID, "", token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if pkg.ExternalURL != "" {
		c.Redirect(http.StatusFound, pkg.ExternalURL)
		return
	}
	if pkg.StoragePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "package has no local file"})
		return
	}
	abs, err := safeStoragePath(h.Cfg.StorageDir, pkg.StoragePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.FileAttachment(abs, pkg.FileName)
}

func (h *PublicHandler) evaluateLegal(projectID uint, req CheckUpdateRequest) LegalInfo {
	legal := LegalInfo{Allowed: true, Code: "ok", Message: "ok"}

	var policy models.VersionPolicy
	if err := h.DB.Where("project_id = ? AND platform = ? AND channel = ?", projectID, req.Platform, req.Channel).First(&policy).Error; err == nil {
		if policy.Maintenance {
			msg := policy.MaintenanceMsg
			if msg == "" {
				msg = "service is under maintenance"
			}
			return LegalInfo{Allowed: false, Code: "maintenance", Message: msg}
		}
		if policy.MinAllowedCode > 0 && req.VersionCode < policy.MinAllowedCode {
			msg := policy.Message
			if msg == "" {
				msg = "this app version is no longer supported"
			}
			return LegalInfo{Allowed: false, Code: "version_too_old", Message: msg}
		}
	}

	var blocked models.BlockedVersion
	if err := h.DB.Where("project_id = ? AND platform = ? AND channel = ? AND version_code = ? AND enabled = ?", projectID, req.Platform, req.Channel, req.VersionCode, true).First(&blocked).Error; err == nil {
		msg := blocked.Reason
		if msg == "" {
			msg = "this app version has been blocked"
		}
		return LegalInfo{Allowed: false, Code: "version_blocked", Message: msg}
	}

	return legal
}

func (h *PublicHandler) findLatestRelease(projectID uint, req CheckUpdateRequest) (models.Release, models.Package, bool) {
	log.Printf(
		"[update] find release project_id=%d platform=%s arch=%s channel=%s current_code=%d",
		projectID,
		req.Platform,
		req.Arch,
		req.Channel,
		req.VersionCode,
	)

	var releases []models.Release
	err := h.DB.Preload("Package").Where(
		"project_id = ? AND platform = ? AND channel = ? AND enabled = ? AND version_code > ? AND (arch = ? OR arch = ? OR arch = '')",
		projectID,
		req.Platform,
		req.Channel,
		true,
		req.VersionCode,
		req.Arch,
		"universal",
	).Order("version_code DESC").Limit(20).Find(&releases).Error
	if err != nil || len(releases) == 0 {
		return models.Release{}, models.Package{}, false
	}

	clientKey := req.ClientID
	if clientKey == "" {
		clientKey = req.InstallID
	}
	if clientKey == "" {
		clientKey = req.DeviceID
	}

	for _, r := range releases {
		if r.RolloutPercent <= 0 {
			continue
		}
		if r.RolloutPercent >= 100 || security.StablePercent(fmt.Sprintf("%d:%s:%d", projectID, clientKey, r.VersionCode)) < r.RolloutPercent {
			return r, r.Package, true
		}
	}
	return models.Release{}, models.Package{}, false
}

func (h *PublicHandler) logCandidateReleases(projectID uint) {
	var all []models.Release
	if err := h.DB.Preload("Package").Where("project_id = ?", projectID).Find(&all).Error; err != nil {
		log.Printf("[update] failed to list releases: %v", err)
		return
	}
	for _, r := range all {
		log.Printf(
			"[update] candidate release id=%d version=%s code=%d platform=%s arch=%s channel=%s enabled=%v min_supported=%d rollout=%d",
			r.ID,
			r.Version,
			r.VersionCode,
			r.Platform,
			r.Arch,
			r.Channel,
			r.Enabled,
			r.MinSupportedCode,
			r.RolloutPercent,
		)
	}
}

func (h *PublicHandler) buildUpdateInfo(project *models.Project, req CheckUpdateRequest, release models.Release, pkg models.Package, legal LegalInfo) (UpdateInfo, error) {
	clientKey := req.ClientID
	if clientKey == "" {
		clientKey = req.InstallID
	}
	if clientKey == "" {
		clientKey = req.DeviceID
	}

	downloadURL := pkg.ExternalURL
	if downloadURL == "" {
		token := security.IssueDownloadToken(h.Cfg.DownloadTokenSecret, pkg.ID, clientKey, h.Cfg.DownloadTokenTTL)
		downloadURL = fmt.Sprintf("%s/api/v1/packages/%d/download?token=%s", h.Cfg.PublicBaseURL, pkg.ID, token)
	}

	updateType := "recommended"
	force := release.Force || !legal.Allowed || (release.MinSupportedCode > 0 && req.VersionCode < release.MinSupportedCode)
	if force {
		updateType = "force"
	}

	payload := security.ManifestPayload{
		AppID:            project.AppID,
		Version:          release.Version,
		VersionCode:      release.VersionCode,
		Platform:         release.Platform,
		Arch:             release.Arch,
		Channel:          release.Channel,
		PackageURL:       downloadURL,
		PackageSize:      pkg.Size,
		PackageSHA256:    pkg.SHA256,
		Force:            force,
		MinSupportedCode: release.MinSupportedCode,
	}
	sig, err := security.SignManifest(project.Ed25519PrivateKey, payload)
	if err != nil {
		return UpdateInfo{}, err
	}

	return UpdateInfo{
		Available:        true,
		UpdateType:       updateType,
		Version:          release.Version,
		VersionCode:      release.VersionCode,
		Platform:         release.Platform,
		Arch:             release.Arch,
		Channel:          release.Channel,
		Title:            release.Title,
		Changelog:        release.Changelog,
		MinSupportedCode: release.MinSupportedCode,
		Force:            force,
		Package: PackageInfo{
			ID:          pkg.ID,
			FileName:    pkg.FileName,
			DownloadURL: downloadURL,
			Size:        pkg.Size,
			SHA256:      pkg.SHA256,
		},
		ManifestSignature: sig,
		CreatedAt:         release.CreatedAt,
	}, nil
}

func normalizeCheckRequest(req *CheckUpdateRequest) {
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Arch = strings.ToLower(strings.TrimSpace(req.Arch))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	if req.Arch == "" {
		req.Arch = "universal"
	}
	if req.Channel == "" {
		req.Channel = "stable"
	}
}

func safeStoragePath(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", errors.New("bad storage path")
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanPath, err := filepath.Abs(filepath.Join(cleanRoot, filepath.Clean(rel)))
	if err != nil {
		return "", err
	}
	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
		return "", errors.New("storage path escapes root")
	}
	return cleanPath, nil
}
