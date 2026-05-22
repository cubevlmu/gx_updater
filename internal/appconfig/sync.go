package appconfig

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gx-update-server/internal/models"
	"gx-update-server/internal/versioning"

	"gorm.io/gorm"
)

func SyncToDB(database *gorm.DB, cfg AppsConfig) error {
	return database.Transaction(func(tx *gorm.DB) error {
		for _, projCfg := range cfg.Projects {
			if err := validateProjectConfig(projCfg); err != nil {
				return fmt.Errorf("project %s: %w", projCfg.AppID, err)
			}

			project, err := upsertProject(tx, projCfg)
			if err != nil {
				return fmt.Errorf("project %s: %w", projCfg.AppID, err)
			}

			if err := tx.Where("release_id IN (SELECT id FROM releases WHERE project_id = ?)", project.ID).Delete(&models.Package{}).Error; err != nil {
				return fmt.Errorf("project %s: delete packages: %w", projCfg.AppID, err)
			}
			if err := tx.Where("project_id = ?", project.ID).Delete(&models.Release{}).Error; err != nil {
				return fmt.Errorf("project %s: delete releases: %w", projCfg.AppID, err)
			}
			if err := tx.Where("project_id = ?", project.ID).Delete(&models.VersionPolicy{}).Error; err != nil {
				return fmt.Errorf("project %s: delete policies: %w", projCfg.AppID, err)
			}
			if err := tx.Where("project_id = ?", project.ID).Delete(&models.BlockedVersion{}).Error; err != nil {
				return fmt.Errorf("project %s: delete blocked versions: %w", projCfg.AppID, err)
			}

			for _, relCfg := range projCfg.Releases {
				if err := validateReleaseConfig(relCfg); err != nil {
					return fmt.Errorf("project %s release %s: %w", projCfg.AppID, relCfg.Version, err)
				}
				if err := createRelease(tx, projCfg.AppID, project.ID, relCfg); err != nil {
					return fmt.Errorf("project %s release %s: %w", projCfg.AppID, relCfg.Version, err)
				}
			}

			for _, polCfg := range projCfg.Policies {
				if err := validatePolicyConfig(polCfg); err != nil {
					return fmt.Errorf("project %s policy %s/%s: %w", projCfg.AppID, polCfg.Platform, polCfg.Channel, err)
				}
				if err := createPolicy(tx, project.ID, polCfg); err != nil {
					return fmt.Errorf("project %s policy %s/%s: %w", projCfg.AppID, polCfg.Platform, polCfg.Channel, err)
				}
			}

			for _, blkCfg := range projCfg.BlockedVersions {
				if err := validateBlockedVersionConfig(blkCfg); err != nil {
					return fmt.Errorf("project %s blocked_version: %w", projCfg.AppID, err)
				}
				if err := createBlockedVersion(tx, project.ID, blkCfg); err != nil {
					return fmt.Errorf("project %s blocked_version: %w", projCfg.AppID, err)
				}
			}
		}
		return nil
	})
}

func validateProjectConfig(cfg ProjectConfig) error {
	if strings.TrimSpace(cfg.AppID) == "" {
		return errors.New("app_id is required")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return errors.New("client_secret is required")
	}
	if strings.TrimSpace(cfg.Ed25519PrivateKey) == "" {
		return errors.New("ed25519_private_key is required")
	}
	if strings.TrimSpace(cfg.Ed25519PublicKey) == "" {
		return errors.New("ed25519_public_key is required")
	}
	return nil
}

func upsertProject(tx *gorm.DB, cfg ProjectConfig) (models.Project, error) {
	var project models.Project
	err := tx.Where("app_id = ?", cfg.AppID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		project = models.Project{AppID: cfg.AppID}
		if err := tx.Create(&project).Error; err != nil {
			return models.Project{}, err
		}
	} else if err != nil {
		return models.Project{}, err
	}

	project.Name = cfg.Name
	project.ClientSecret = cfg.ClientSecret
	project.Ed25519PublicKey = cfg.Ed25519PublicKey
	project.Ed25519PrivateKey = cfg.Ed25519PrivateKey
	if cfg.Enabled != nil {
		project.Enabled = *cfg.Enabled
	}
	if err := tx.Save(&project).Error; err != nil {
		return models.Project{}, err
	}
	return project, nil
}

func validateReleaseConfig(cfg ReleaseConfig) error {
	if strings.TrimSpace(cfg.Version) == "" {
		return errors.New("version is required")
	}
	if cfg.VersionCode <= 0 {
		if _, err := versioning.ParseVersionCode(cfg.Version); err != nil {
			return fmt.Errorf("version %s cannot be parsed and version_code not set: %w", cfg.Version, err)
		}
	}
	if strings.TrimSpace(cfg.Platform) == "" {
		return errors.New("platform is required")
	}
	if strings.TrimSpace(cfg.Package.SHA256) == "" {
		return errors.New("package.sha256 is required")
	}
	if cfg.Package.StoragePath == "" && cfg.Package.ExternalURL == "" {
		return errors.New("package.storage_path or package.external_url is required")
	}
	return nil
}

func createRelease(tx *gorm.DB, appID string, projectID uint, cfg ReleaseConfig) error {
	platform := norm(cfg.Platform)
	arch := norm(cfg.Arch)
	channel := norm(cfg.Channel)
	if arch == "" {
		arch = "universal"
	}
	if channel == "" {
		channel = "stable"
	}

	rollout := cfg.RolloutPercent
	if rollout <= 0 {
		rollout = 100
	}
	if rollout > 100 {
		rollout = 100
	}

	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}

	versionCode := cfg.VersionCode
	if versionCode <= 0 {
		var err error
		versionCode, err = versioning.ParseVersionCode(cfg.Version)
		if err != nil {
			return fmt.Errorf("release %s parse version code: %w", cfg.Version, err)
		}
	}

	pkg := models.Package{
		FileName:    packageFileName(appID, cfg),
		StoragePath: cfg.Package.StoragePath,
		ExternalURL: cfg.Package.ExternalURL,
		Size:        cfg.Package.Size,
		SHA256:      cfg.Package.SHA256,
	}

	release := models.Release{
		ProjectID:        projectID,
		Version:          cfg.Version,
		VersionCode:      versionCode,
		Platform:         platform,
		Arch:             arch,
		Channel:          channel,
		Title:            cfg.Title,
		Changelog:        cfg.Changelog,
		MinSupportedCode: cfg.MinSupportedCode,
		Force:            cfg.Force,
		Enabled:          enabled,
		RolloutPercent:   rollout,
	}

	if err := tx.Create(&release).Error; err != nil {
		return err
	}
	pkg.ReleaseID = release.ID
	if err := tx.Create(&pkg).Error; err != nil {
		return err
	}
	return nil
}

func packageFileName(appID string, cfg ReleaseConfig) string {
	name := cfg.Package.FileName
	if name != "" {
		return name
	}
	if cfg.Package.StoragePath != "" {
		name = filepath.Base(cfg.Package.StoragePath)
	}
	if name == "" && cfg.Package.ExternalURL != "" {
		name = filepath.Base(strings.Split(cfg.Package.ExternalURL, "?")[0])
	}
	if name == "." || name == "/" || name == "" || name == string(filepath.Separator) {
		name = fmt.Sprintf("%s-%s.bin", appID, cfg.Version)
	}
	return name
}

func validatePolicyConfig(cfg PolicyConfig) error {
	if strings.TrimSpace(cfg.Platform) == "" {
		return errors.New("platform is required")
	}
	return nil
}

func createPolicy(tx *gorm.DB, projectID uint, cfg PolicyConfig) error {
	platform := norm(cfg.Platform)
	channel := norm(cfg.Channel)
	if channel == "" {
		channel = "stable"
	}

	minAllowedCode := cfg.MinAllowedCode
	if minAllowedCode <= 0 && strings.TrimSpace(cfg.MinAllowedVersion) != "" {
		var err error
		minAllowedCode, err = versioning.ParseVersionCode(cfg.MinAllowedVersion)
		if err != nil {
			return fmt.Errorf("policy min_allowed_version %s parse: %w", cfg.MinAllowedVersion, err)
		}
	}

	p := models.VersionPolicy{
		ProjectID:      projectID,
		Platform:       platform,
		Channel:        channel,
		MinAllowedCode: minAllowedCode,
		Message:        cfg.Message,
		Maintenance:    cfg.Maintenance,
		MaintenanceMsg: cfg.MaintenanceMsg,
	}
	return tx.Create(&p).Error
}

func validateBlockedVersionConfig(cfg BlockedVersionConfig) error {
	if strings.TrimSpace(cfg.Platform) == "" {
		return errors.New("platform is required")
	}
	return nil
}

func createBlockedVersion(tx *gorm.DB, projectID uint, cfg BlockedVersionConfig) error {
	platform := norm(cfg.Platform)
	channel := norm(cfg.Channel)
	if channel == "" {
		channel = "stable"
	}

	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}

	versionCode := cfg.VersionCode
	if versionCode <= 0 && strings.TrimSpace(cfg.Version) != "" {
		var err error
		versionCode, err = versioning.ParseVersionCode(cfg.Version)
		if err != nil {
			return fmt.Errorf("blocked version %s parse: %w", cfg.Version, err)
		}
	}
	if versionCode <= 0 {
		return errors.New("blocked version requires version or version_code")
	}

	b := models.BlockedVersion{
		ProjectID:   projectID,
		Platform:    platform,
		Channel:     channel,
		VersionCode: versionCode,
		Reason:      cfg.Reason,
		Enabled:     enabled,
	}
	return tx.Create(&b).Error
}

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
