package appconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AppsConfig struct {
	Projects []ProjectConfig `yaml:"projects"`
}

type ProjectConfig struct {
	AppID             string                 `yaml:"app_id"`
	Name              string                 `yaml:"name"`
	Enabled           *bool                  `yaml:"enabled"`
	ClientSecret      string                 `yaml:"client_secret"`
	Ed25519PublicKey  string                 `yaml:"ed25519_public_key"`
	Ed25519PrivateKey string                 `yaml:"ed25519_private_key"`
	Policies          []PolicyConfig         `yaml:"policies"`
	Releases          []ReleaseConfig        `yaml:"releases"`
	BlockedVersions   []BlockedVersionConfig `yaml:"blocked_versions"`
}

type PolicyConfig struct {
	Platform          string `yaml:"platform"`
	Channel           string `yaml:"channel"`
	MinAllowedCode    int64  `yaml:"min_allowed_code"`
	MinAllowedVersion string `yaml:"min_allowed_version"`
	Message           string `yaml:"message"`
	Maintenance       bool   `yaml:"maintenance"`
	MaintenanceMsg    string `yaml:"maintenance_msg"`
}

type ReleaseConfig struct {
	Version          string        `yaml:"version"`
	VersionCode      int64         `yaml:"version_code"`
	Platform         string        `yaml:"platform"`
	Arch             string        `yaml:"arch"`
	Channel          string        `yaml:"channel"`
	Title            string        `yaml:"title"`
	Changelog        string        `yaml:"changelog"`
	MinSupportedCode int64         `yaml:"min_supported_code"`
	Force            bool          `yaml:"force"`
	Enabled          *bool         `yaml:"enabled"`
	RolloutPercent   int           `yaml:"rollout_percent"`
	Package          PackageConfig `yaml:"package"`
}

type PackageConfig struct {
	FileName    string `yaml:"file_name"`
	StoragePath string `yaml:"storage_path"`
	ExternalURL string `yaml:"external_url"`
	Size        int64  `yaml:"size"`
	SHA256      string `yaml:"sha256"`
}

type BlockedVersionConfig struct {
	Platform    string `yaml:"platform"`
	Channel     string `yaml:"channel"`
	Version     string `yaml:"version"`
	VersionCode int64  `yaml:"version_code"`
	Reason      string `yaml:"reason"`
	Enabled     *bool  `yaml:"enabled"`
}

func Load(path string) (AppsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppsConfig{}, fmt.Errorf("read apps config %s: %w", path, err)
	}
	var cfg AppsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return AppsConfig{}, fmt.Errorf("parse apps config %s: %w", path, err)
	}
	return cfg, nil
}
