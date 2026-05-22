<a id="readme-top"></a>

[![Go][go-shield]][go-url]
[![Gin][gin-shield]][gin-url]
[![GORM][gorm-shield]][gorm-url]
[![SQLite][sqlite-shield]][sqlite-url]

<br />
<div align="center">
  <h3 align="center">gx-updater</h3>

  <p align="center">
    A reusable app auto-update / version blocking / package distribution server.
    <br />
    Built with Go + Gin + GORM + SQLite. One service, multiple apps, platforms, and channels.
  </p>
</div>

## Table of Contents

- [About The Project](#about-the-project)
- [Built With](#built-with)
- [Features](#features)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Release Workflow](#release-workflow)
- [Docker Deployment](#docker-deployment)
- [Client Implementation](#client-implementation)
- [Database Schema](#database-schema)

## About The Project

GX Update Server is a Go-based update management backend designed for self-distributed apps (APK, desktop packages, Flutter bundles) that are not published on app stores.

Key design decisions:

- **Config-file-driven** — no admin web UI or management API. All project data lives in `data/config/apps.yaml`. Edit the file, hit the reload endpoint or type `reload` in the console. No restart needed.
- **HMAC request signing** — client requests to `/api/v1/check` are signed with a per-project `client_secret`. Nonces prevent replay attacks.
- **Ed25519 manifest signing** — the server signs every update manifest with a per-project Ed25519 private key. The client verifies with the baked-in public key before installing.
- **Short-lived download tokens** — package download URLs carry an expiring HMAC token, preventing hotlinking.
- **SQLite** — zero-dependency database. No Postgres or MySQL needed for personal deployments.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Built With

- [Go](https://go.dev/) 1.23
- [Gin](https://github.com/gin-gonic/gin) — HTTP framework
- [GORM](https://gorm.io/) — ORM
- [SQLite](https://www.sqlite.org/) via [go-sqlite3](https://github.com/mattn/go-sqlite3)
- [yaml.v3](https://gopkg.in/yaml.v3) — config parsing
- [Ed25519](https://pkg.go.dev/crypto/ed25519) — manifest signing

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Features

- Multi-app — distinguish projects by `app_id`
- Multi-platform — `android`, `windows`, `macos`, `linux`, `ios`, etc.
- Multi-channel — `stable`, `beta`, `dev`
- Multi-arch — `universal`, `arm64-v8a`, `x86_64`, etc.
- Auto update detection — client submits current version, server returns latest
- Forced updates — per-release `force` flag or `min_supported_code` threshold
- Version blocking — set minimum allowed version per platform/channel, or block specific version codes
- Staged rollout — per-release `rollout_percent` (0–100)
- Package delivery — local file serving or external URL redirect
- Request signing — HMAC-SHA256 client authentication
- Manifest signing — Ed25519-signed update manifests
- Download tokens — expiring HMAC tokens for package URLs
- Config hot-reload — console `reload` command or `POST /-/reload`, no restart

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Project Structure

```text
gx_update_server/
├── cmd/
│   ├── server/main.go         # HTTP server entrypoint
│   └── genkeys/main.go        # Key generation CLI
├── examples/
│   ├── system.example.yaml    # System config template
│   └── apps.example.yaml      # Project config template
├── internal/
│   ├── appconfig/             # YAML config loading + DB sync
│   ├── config/                # System config (YAML + env overrides)
│   ├── db/                    # SQLite open + auto-migrate
│   ├── handlers/              # Public API handlers
│   ├── middleware/             # Client HMAC auth middleware
│   ├── models/                # GORM models
│   ├── router/                # Route registration
│   ├── security/              # HMAC, Ed25519, download tokens
│   └── versioning/            # version → version_code conversion
├── scripts/                   # Client signing examples
├── data/                      # Runtime data (gitignored)
│   └── config/
│       ├── system.yaml        # Actual system config (secrets)
│       └── apps.yaml          # Actual project config (secrets)
├── storage/                   # Package files (gitignored)
├── Dockerfile
├── .dockerignore
├── .gitignore
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Getting Started

### Prerequisites

- Go 1.23+
- GCC (required by go-sqlite3 CGO driver)

Check your environment:

```sh
go version
```

### Installation

```sh
git clone <repo-url>
cd gx_update_server

# Copy config templates
cp examples/system.example.yaml data/config/system.yaml
cp examples/apps.example.yaml  data/config/apps.yaml

# Generate keys for your project
go run ./cmd/genkeys

# Edit data/config/apps.yaml — paste the generated keys

# Edit data/config/system.yaml — set public_base_url and secrets

go mod tidy
go run ./cmd/server
```

### Health Check

```sh
curl http://127.0.0.1:8080/healthz
# → {"status":"ok"}
```

### Hot Reload

After editing `data/config/apps.yaml`, trigger a reload without restarting:

```sh
# Option A: type in the server console
reload

# Option B: HTTP call from another terminal
curl -X POST http://127.0.0.1:8080/-/reload
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Configuration

All config files live under `data/config/`. Template files live under `examples/` and are safe to commit.

| File | Purpose | Committed? |
|---|---|---|
| `examples/system.example.yaml` | System config template | Yes |
| `examples/apps.example.yaml` | Project config template | Yes |
| `data/config/system.yaml` | Actual system config (secrets) | No |
| `data/config/apps.yaml` | Actual project config (secrets) | No |

### system.yaml

Server runtime settings. Env vars take priority over file values.

```yaml
addr: ":8080"
db_path: "data/update.db"
storage_dir: "storage"
public_base_url: "http://127.0.0.1:8080"
apps_config_path: "data/config/apps.yaml"
download_token_secret: "replace-with-long-random-download-secret"
client_clock_skew_seconds: 300
download_token_ttl_seconds: 900
```

| Env Var | YAML Field | Default |
|---|---|---|
| `UPDATE_SYSTEM_CONFIG` | — | `data/config/system.yaml` |
| `UPDATE_SERVER_ADDR` | `addr` | `:8080` |
| `UPDATE_DB_PATH` | `db_path` | `data/update.db` |
| `UPDATE_STORAGE_DIR` | `storage_dir` | `storage` |
| `UPDATE_PUBLIC_BASE_URL` | `public_base_url` | `http://127.0.0.1:8080` |
| `UPDATE_APPS_CONFIG` | `apps_config_path` | `data/config/apps.yaml` |
| `UPDATE_DOWNLOAD_TOKEN_SECRET` | `download_token_secret` | — |
| `UPDATE_CLIENT_CLOCK_SKEW_SECONDS` | `client_clock_skew_seconds` | `300` |
| `UPDATE_DOWNLOAD_TOKEN_TTL_SECONDS` | `download_token_ttl_seconds` | `900` |

### apps.yaml

Defines projects, releases, policies, and blocked versions. See `examples/apps.example.yaml` for a complete example.

```yaml
projects:
  - app_id: me.cubevlmu.class_tool
    name: Class Tool
    enabled: true
    client_secret: "base64-32-random-bytes"
    ed25519_public_key: "base64-ed25519-pub"
    ed25519_private_key: "base64-ed25519-priv"

    policies:
      - platform: windows
        channel: stable
        min_allowed_version: "0.8.1"
        message: "Your version is too old. Please update."
        maintenance: false
        maintenance_msg: ""

    releases:
      - version: "0.8.1"
        platform: windows
        arch: x64
        channel: stable
        title: "0.8.1 Windows Update"
        changelog: |
          - Performance improvements
          - Bug fixes
        min_supported_code: 1
        force: true
        enabled: true
        rollout_percent: 100
        package:
          file_name: "class_tool_0.8.1_windows_x64.zip"
          storage_path: "me.cubevlmu.class_tool/windows/stable/x64/0.8.1/class_tool_0.8.1_windows_x64.zip"
          external_url: ""
          size: 12345678
          sha256: "abc123..."

    blocked_versions:
      - platform: windows
        channel: stable
        version: "0.8.0"
        reason: "Version 0.8.0 is no longer supported. Please update."
        enabled: true
```

### Version Code Auto-Generation

`version_code` is optional. When omitted, the server derives it from `version`:

| version | version_code |
|---|---|
| `0.8.0` | `8000` |
| `0.8.1` | `8001` |
| `0.10.0` | `10000` |
| `1.0.0` | `1000000` |
| `1.2.3` | `1002003` |

Formula: `major × 1,000,000 + minor × 1,000 + patch`

`min_allowed_version` and `blocked_versions.version` support the same auto-conversion. If `version_code` is explicitly set (>0), it takes precedence.

### Generating Keys

```sh
go run ./cmd/genkeys
```

Output:

```yaml
app_id: com.example.app
name: Example App
enabled: true
client_secret: "..."
ed25519_public_key: "..."
ed25519_private_key: "..."
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## API Reference

### POST /api/v1/check

Check for updates. Requires HMAC-SHA256 client authentication.

**Headers:**

```http
X-App-Id: com.example.app
X-Timestamp: 1715870000
X-Nonce: random-uuid
X-Signature: base64(hmac-sha256(client_secret, signing_payload))
Content-Type: application/json
```

**Signing payload:**

```text
METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(RAW_BODY)
```

> Sign the raw JSON bytes actually sent. Do not sign a hand-written JSON string and send a differently-encoded body.

**Request body:**

```json
{
  "app_id": "com.example.app",
  "platform": "windows",
  "arch": "x64",
  "channel": "stable",
  "version": "0.8.0",
  "version_code": 8000,
  "client_id": "device-or-install-id",
  "install_id": "install-uuid",
  "os_version": "Windows 11"
}
```

**Response:**

```json
{
  "request_id": "uuid",
  "server_time": 1715870001,
  "legal": {
    "allowed": false,
    "code": "version_too_old",
    "message": "Your version is too old. Please update."
  },
  "update": {
    "available": true,
    "update_type": "force",
    "version": "0.8.1",
    "version_code": 8001,
    "platform": "windows",
    "arch": "x64",
    "channel": "stable",
    "title": "0.8.1 Windows Update",
    "changelog": "Performance improvements\nBug fixes",
    "min_supported_code": 1,
    "force": true,
    "package": {
      "id": 1,
      "file_name": "class_tool_0.8.1_windows_x64.zip",
      "download_url": "https://update.example.com/api/v1/packages/1/download?token=...",
      "size": 12345678,
      "sha256": "abc123..."
    },
    "manifest_signature": "base64-ed25519-signature",
    "created_at": "2026-05-22T00:00:00Z"
  }
}
```

- `legal.allowed = false` — client should block usage and show `legal.message`.
- `update.update_type = "force"` — client should show a non-dismissable update dialog.
- `update.update_type = "recommended"` — client should show a dismissable update dialog.
- `update.available = false` — no update available, proceed normally.

### GET /api/v1/packages/:package_id/download

Download a package file. Requires a valid `?token=` query parameter from the check response.

```sh
curl -O "https://update.example.com/api/v1/packages/1/download?token=..."
```

### GET /healthz

Health check. No auth required.

### POST /-/reload

Hot-reload `apps.yaml` from disk and re-sync to the database. No auth required.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Release Workflow

```sh
# 1. Place the package file
mkdir -p storage/me.cubevlmu.class_tool/windows/stable/x64/0.8.1
cp app-release.zip storage/me.cubevlmu.class_tool/windows/stable/x64/0.8.1/class_tool_0.8.1_windows_x64.zip

# 2. Compute SHA256
# Linux / macOS:
sha256sum storage/me.cubevlmu.class_tool/windows/stable/x64/0.8.1/class_tool_0.8.1_windows_x64.zip

# Windows PowerShell:
Get-FileHash storage\me.cubevlmu.class_tool\windows\stable\x64\0.8.1\class_tool_0.8.1_windows_x64.zip -Algorithm SHA256

# 3. Edit data/config/apps.yaml — add/update the release entry with sha256 and size

# 4. Hot-reload (no restart)
curl -X POST http://127.0.0.1:8080/-/reload
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Docker Deployment

```sh
# 1. Clone and configure
git clone <repo-url>
cd gx_update_server
cp examples/system.example.yaml data/config/system.yaml
cp examples/apps.example.yaml  data/config/apps.yaml
# Edit both YAML files with your actual keys and settings

# 2. Build and start
docker compose build
docker compose up -d
```

**docker-compose.yml** mounts `data/` and `storage/` as volumes:

```yaml
services:
  update-server:
    build: .
    container_name: gx-update-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      UPDATE_SYSTEM_CONFIG: "/app/data/config/system.yaml"
      UPDATE_APPS_CONFIG: "/app/data/config/apps.yaml"
      UPDATE_PUBLIC_BASE_URL: "https://update.example.com"
    volumes:
      - ./data:/app/data
      - ./storage:/app/storage
```

**Hot-reload inside Docker:**

```sh
docker compose exec update-server wget -qO- http://127.0.0.1:8080/-/reload
```

Recommended: place behind Nginx or Caddy with HTTPS enabled.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Client Implementation

### Manifest Verification

The server signs a canonical manifest payload with Ed25519. The client must verify the signature before installing.

**Canonical payload:**

```text
gx-update-manifest-v1
app_id=<app_id>
version=<version>
version_code=<version_code>
platform=<platform>
arch=<arch>
channel=<channel>
package_url=<download_url>
package_size=<size>
package_sha256=<sha256>
force=<true|false>
min_supported_code=<min_supported_code>
```

**Client-side steps:**

1. Request `/api/v1/check` with HMAC signature.
2. If the request fails, allow entry but show "update check failed".
3. If `legal.allowed = false`, block usage and show `legal.message`.
4. If `update.available = false`, proceed normally.
5. If `update.update_type = "recommended"`, show a dismissable update dialog.
6. If `update.update_type = "force"`, show a non-dismissable update dialog.
7. Download the package.
8. Verify SHA256 of the downloaded file against `package.sha256`.
9. Verify `manifest_signature` using the baked-in `ed25519_public_key`.
10. If both pass, invoke the platform installer.

### Security Notes

- `client_secret` is embedded in the client binary. It can be extracted via reverse engineering. Use it only for rate-limiting and basic anti-replay, not as a hard security boundary.
- The real security comes from Ed25519 manifest signing + SHA256 verification. The client bakes in the public key and the package hash. An attacker who compromises the server cannot forge a valid manifest signature without the private key.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Database Schema

| Table | Description |
|---|---|
| `projects` | App ID, name, HMAC secret, Ed25519 keypair |
| `releases` | Version records with platform/arch/channel/rollout |
| `packages` | File name, storage path, SHA256, size, external URL |
| `version_policies` | Minimum allowed version, maintenance mode |
| `blocked_versions` | Explicitly blocked version codes |
| `request_nonces` | Anti-replay nonces with TTL |

<p align="right">(<a href="#readme-top">back to top</a>)</p>

[go-shield]: https://img.shields.io/badge/Go-1.23-00ADD8?style=for-the-badge&logo=go&logoColor=white
[go-url]: https://go.dev/
[gin-shield]: https://img.shields.io/badge/Gin-HTTP%20Framework-008ECF?style=for-the-badge&logo=go&logoColor=white
[gin-url]: https://github.com/gin-gonic/gin
[gorm-shield]: https://img.shields.io/badge/GORM-ORM-00ADD8?style=for-the-badge&logo=go&logoColor=white
[gorm-url]: https://gorm.io/
[sqlite-shield]: https://img.shields.io/badge/SQLite-Database-003B57?style=for-the-badge&logo=sqlite&logoColor=white
[sqlite-url]: https://www.sqlite.org/
