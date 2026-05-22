package models

import "time"

type Project struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	AppID             string    `gorm:"uniqueIndex;size:128;not null" json:"app_id"`
	Name              string    `gorm:"size:128;not null" json:"name"`
	ClientSecret      string    `gorm:"size:256;not null" json:"client_secret,omitempty"`
	Ed25519PublicKey  string    `gorm:"size:256;not null" json:"ed25519_public_key"`
	Ed25519PrivateKey string    `gorm:"size:512;not null" json:"-"`
	Enabled           bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Release struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ProjectID        uint      `gorm:"index;not null" json:"project_id"`
	Project          Project   `json:"-"`
	Version          string    `gorm:"size:64;not null" json:"version"`
	VersionCode      int64     `gorm:"index;not null" json:"version_code"`
	Platform         string    `gorm:"size:64;index;not null" json:"platform"` // android, windows, macos, ios, linux
	Arch             string    `gorm:"size:64;index;not null;default:universal" json:"arch"`
	Channel          string    `gorm:"size:64;index;not null;default:stable" json:"channel"` // stable, beta, dev
	Title            string    `gorm:"size:256" json:"title"`
	Changelog        string    `gorm:"type:text" json:"changelog"`
	MinSupportedCode int64     `gorm:"not null;default:0" json:"min_supported_code"`
	Force            bool      `gorm:"not null;default:false" json:"force"`
	Enabled          bool      `gorm:"not null;default:true" json:"enabled"`
	RolloutPercent   int       `gorm:"not null;default:100" json:"rollout_percent"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Package          Package   `gorm:"constraint:OnDelete:CASCADE" json:"package"`
}

type Package struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ReleaseID   uint      `gorm:"uniqueIndex;not null" json:"release_id"`
	FileName    string    `gorm:"size:256;not null" json:"file_name"`
	StoragePath string    `gorm:"size:1024" json:"storage_path"` // relative path under storage dir
	ExternalURL string    `gorm:"size:2048" json:"external_url"`
	Size        int64     `gorm:"not null" json:"size"`
	SHA256      string    `gorm:"size:64;not null" json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type VersionPolicy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProjectID      uint      `gorm:"uniqueIndex:uniq_policy;not null" json:"project_id"`
	Platform       string    `gorm:"uniqueIndex:uniq_policy;size:64;not null" json:"platform"`
	Channel        string    `gorm:"uniqueIndex:uniq_policy;size:64;not null;default:stable" json:"channel"`
	MinAllowedCode int64     `gorm:"not null;default:0" json:"min_allowed_code"`
	Message        string    `gorm:"size:512" json:"message"`
	Maintenance    bool      `gorm:"not null;default:false" json:"maintenance"`
	MaintenanceMsg string    `gorm:"size:512" json:"maintenance_msg"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BlockedVersion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ProjectID   uint      `gorm:"index;not null" json:"project_id"`
	Platform    string    `gorm:"size:64;index;not null" json:"platform"`
	Channel     string    `gorm:"size:64;index;not null;default:stable" json:"channel"`
	VersionCode int64     `gorm:"index;not null" json:"version_code"`
	Reason      string    `gorm:"size:512" json:"reason"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RequestNonce struct {
	ID        uint      `gorm:"primaryKey"`
	ProjectID uint      `gorm:"uniqueIndex:uniq_nonce;not null"`
	Nonce     string    `gorm:"uniqueIndex:uniq_nonce;size:128;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}
