package handlers

import "time"

type CheckUpdateRequest struct {
	AppID       string `json:"app_id"`
	Platform    string `json:"platform" binding:"required"`
	Arch        string `json:"arch"`
	Channel     string `json:"channel"`
	Version     string `json:"version" binding:"required"`
	VersionCode int64  `json:"version_code" binding:"required"`
	ClientID    string `json:"client_id"`
	InstallID   string `json:"install_id"`
	DeviceID    string `json:"device_id"`
	OSVersion   string `json:"os_version"`
}

type CheckUpdateResponse struct {
	RequestID  string      `json:"request_id"`
	ServerTime int64       `json:"server_time"`
	Legal      LegalInfo   `json:"legal"`
	Update     *UpdateInfo `json:"update,omitempty"`
}

type LegalInfo struct {
	Allowed bool   `json:"allowed"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UpdateInfo struct {
	Available         bool        `json:"available"`
	UpdateType        string      `json:"update_type"` // none, recommended, force
	Version           string      `json:"version"`
	VersionCode       int64       `json:"version_code"`
	Platform          string      `json:"platform"`
	Arch              string      `json:"arch"`
	Channel           string      `json:"channel"`
	Title             string      `json:"title"`
	Changelog         string      `json:"changelog"`
	MinSupportedCode  int64       `json:"min_supported_code"`
	Force             bool        `json:"force"`
	Package           PackageInfo `json:"package"`
	ManifestSignature string      `json:"manifest_signature"`
	CreatedAt         time.Time   `json:"created_at"`
}

type PackageInfo struct {
	ID          uint   `json:"id"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}
