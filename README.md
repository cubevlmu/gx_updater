# GX Update Server

一个可复用的 App 自动更新 / 版本封禁 / 更新包分发服务端，基于 Go + Gin + GORM + SQLite。

适合不上架应用商店、自己分发 APK / 桌面端安装包 / Flutter 包的项目。一个服务可以管理多个 App 项目、多平台、多渠道。

## 核心能力

- 多项目：通过 `app_id` 区分不同 App。
- 多平台：`android` / `windows` / `macos` / `linux` / `ios` 等都可以。
- 多渠道：`stable` / `beta` / `dev`。
- 多架构：`universal` / `arm64-v8a` / `x86_64` 等。
- 自动检测更新：客户端提交当前版本，服务端返回最新版本。
- 强制更新：某个 release 可设置 `force=true`。
- 旧版本封禁：可以设置某平台某渠道最低允许版本号。
- 精确封禁：可以封禁单个 `version_code`。
- 灰度发布：release 可设置 `rollout_percent`。
- 更新包分发：支持直接上传文件，也支持外链包。
- 请求签名：客户端调用 `/api/v1/check` 必须使用 HMAC-SHA256 签名。
- 更新清单签名：服务端使用 Ed25519 私钥签名更新清单，客户端内置公钥校验。
- 短时下载 token：更新包下载链接默认带过期 token。

## 重要安全说明

客户端内置的 `client_secret` 可以被逆向提取，所以它只能用于防刷、防普通抓包重放，不能当成绝对安全边界。

真正关键的是更新清单和安装包校验：

1. 服务端返回包的 `sha256`。
2. 服务端用 Ed25519 私钥签名清单。
3. 客户端内置 Ed25519 公钥。
4. 客户端下载包后先校验 SHA256，再校验清单签名。
5. 校验失败就拒绝安装。

## 快速启动

```bash
cd gx_update_server

# 1. 复制系统配置模板
cp examples/system.example.yaml data/config/system.yaml

# 2. 生成密钥
go run ./cmd/genkeys

# 3. 复制项目配置模板
cp examples/apps.example.yaml data/config/apps.yaml
# 把 genkeys 输出的密钥填入 apps.yaml

# 4. 启动服务
go run ./cmd/server
```

健康检查：

```bash
curl http://127.0.0.1:8080/healthz
```

## 配置文件

| 位置 | 说明 |
|---|---|
| `examples/system.example.yaml` | 系统配置模板（可提交 git） |
| `examples/apps.example.yaml` | 项目配置模板（可提交 git） |
| `data/config/system.yaml` | 实际系统配置（含密钥，不提交 git） |
| `data/config/apps.yaml` | 实际项目配置（含密钥、sha256，不提交 git） |

### system.yaml

系统运行参数，参考 `examples/system.example.yaml`：

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

环境变量优先级高于 system.yaml：

| 环境变量 | system.yaml 字段 | 默认值 |
|---|---|---:|
| `UPDATE_SYSTEM_CONFIG` | — | `data/config/system.yaml` |
| `UPDATE_SERVER_ADDR` | `addr` | `:8080` |
| `UPDATE_DB_PATH` | `db_path` | `data/update.db` |
| `UPDATE_STORAGE_DIR` | `storage_dir` | `storage` |
| `UPDATE_PUBLIC_BASE_URL` | `public_base_url` | `http://127.0.0.1:8080` |
| `UPDATE_APPS_CONFIG` | `apps_config_path` | `data/config/apps.yaml` |
| `UPDATE_DOWNLOAD_TOKEN_SECRET` | `download_token_secret` | — |
| `UPDATE_CLIENT_CLOCK_SKEW_SECONDS` | `client_clock_skew_seconds` | `300` |
| `UPDATE_DOWNLOAD_TOKEN_TTL_SECONDS` | `download_token_ttl_seconds` | `900` |

## 项目管理

服务端通过 `data/config/apps.yaml` 配置文件管理所有项目和版本。

### 生成密钥

```bash
go run ./cmd/genkeys
```

输出 YAML 片段，复制到 `data/config/apps.yaml` 中：

```yaml
app_id: com.example.app
name: Example App
enabled: true
client_secret: "..."
ed25519_public_key: "..."
ed25519_private_key: "..."
```

### apps.yaml 结构

参考 `examples/apps.example.yaml`：

```yaml
projects:
  - app_id: me.cubevlmu.class_tool
    name: Class Tool
    enabled: true
    client_secret: "..."
    ed25519_public_key: "..."
    ed25519_private_key: "..."

    policies:
      - platform: windows
        channel: stable
        min_allowed_version: "0.8.1"
        message: "版本过低，请更新客户端版本。"
        maintenance: false

    releases:
      - version: "0.8.1"
        platform: windows
        arch: x64
        channel: stable
        title: "0.8.1 Windows 更新"
        changelog: |
          - 测试 Windows 自动更新
        min_supported_code: 1
        force: true
        enabled: true
        rollout_percent: 100
        package:
          file_name: "class_tool_0.8.1_windows_x64.zip"
          storage_path: "me.cubevlmu.class_tool/windows/stable/x64/0.8.1/class_tool_0.8.1_windows_x64.zip"
          external_url: ""
          size: 12345678
          sha256: "..."

    blocked_versions:
      - platform: windows
        channel: stable
        version: "0.8.0"
        reason: "已停止支持，请更新后继续使用。"
        enabled: true
```

#### 版本号自动转换

`version_code` 可以不手写，服务端自动根据 `version` 生成：

| version | version_code |
|---|---:|
| `0.8.0` | `8000` |
| `0.8.1` | `8001` |
| `0.9.0` | `9000` |
| `0.10.0` | `10000` |
| `1.0.0` | `1000000` |
| `1.2.3` | `1002003` |

规则：`major * 1_000_000 + minor * 1_000 + patch`。

`min_allowed_version` 和 `blocked_versions.version` 也同样支持自动转换。
如果显式写 `version_code` > 0，则优先使用手写值。
```

### 发布新版本流程

```bash
# 1. 放安装包
mkdir -p storage/com.example.class_tool/android/stable/universal/1.2.0
cp app-release.apk storage/com.example.class_tool/android/stable/universal/1.2.0/class_tool_1.2.0.apk

# 2. 计算 sha256
# Windows PowerShell:
Get-FileHash .\storage\com.example.class_tool\android\stable\universal\1.2.0\class_tool_1.2.0.apk -Algorithm SHA256

# Linux/macOS:
sha256sum ./storage/com.example.class_tool/android/stable/universal/1.2.0/class_tool_1.2.0.apk

# 3. 修改 data/config/apps.yaml 的 releases 列表，填入 sha256 和 size

# 4. 重载配置（无需重启，控制台输入 reload 或 curl）
curl -X POST http://127.0.0.1:8080/-/reload
```

## 客户端检测更新接口

### 请求

```http
POST /api/v1/check
X-App-Id: com.example.class_tool
X-Timestamp: 1779430000
X-Nonce: random-uuid
X-Signature: base64(hmac_sha256(client_secret, payload))
Content-Type: application/json
```

Body 示例：

```json
{
  "app_id": "com.example.class_tool",
  "platform": "android",
  "arch": "universal",
  "channel": "stable",
  "version": "1.0.0",
  "version_code": 100,
  "client_id": "device-or-install-id",
  "install_id": "install-uuid",
  "os_version": "Android 14"
}
```

签名 payload 规则：

```text
METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(RAW_BODY)
```

例如：

```text
POST
/api/v1/check
1779430000
random-uuid
<body_sha256_hex>
```

注意：必须对实际发送的 JSON 原始字节做 SHA256。不要手写一个 JSON 字符串签名，然后又用另一个 JSON 编码结果发送。

### 响应

```json
{
  "request_id": "uuid",
  "server_time": 1779430001,
  "legal": {
    "allowed": true,
    "code": "ok",
    "message": "ok"
  },
  "update": {
    "available": true,
    "update_type": "recommended",
    "version": "1.2.0",
    "version_code": 120,
    "platform": "android",
    "arch": "universal",
    "channel": "stable",
    "title": "1.2.0 更新",
    "changelog": "优化课程表小组件；修复下一节课程显示错误。",
    "min_supported_code": 100,
    "force": false,
    "package": {
      "id": 1,
      "file_name": "app-release.apk",
      "download_url": "https://update.example.com/api/v1/packages/1/download?token=...",
      "size": 12345678,
      "sha256": "..."
    },
    "manifest_signature": "base64-ed25519-signature",
    "created_at": "2026-05-22T00:00:00Z"
  }
}
```

`legal.allowed=false` 时，客户端应阻止进入主功能，并提示 `legal.message`。如果同时有 `update.update_type=force`，应引导用户更新。

## 客户端验签规则

服务端签名的 canonical payload 是：

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

客户端用配置文件中的 `ed25519_public_key` 校验 `manifest_signature`。

下载完成后，还要计算文件 SHA256，与响应里的 `package.sha256` 完全一致才允许安装。

## 服务器部署

```bash
# 1. clone 仓库到服务器
git clone <repo> && cd gx_update_server

# 2. 准备配置文件
cp examples/system.example.yaml data/config/system.yaml
cp examples/apps.example.yaml  data/config/apps.yaml
# 编辑两个 yaml，填入密钥和 sha256

# 3. 打包镜像并启动
docker compose build
docker compose up -d
```

更新后热重载（无需重启容器）：

```bash
docker compose exec update-server wget -qO- http://127.0.0.1:8080/-/reload
```

生产环境建议放在 Nginx / Caddy 后面，并开启 HTTPS。

## 数据库表

- `projects`：项目、客户端 HMAC 密钥、Ed25519 公私钥。
- `releases`：版本记录。
- `packages`：安装包文件、SHA256、大小、外链。
- `version_policies`：最低允许版本、维护模式。
- `blocked_versions`：精确封禁某个版本号。
- `request_nonces`：请求 nonce，防重放。

## 推荐客户端逻辑

1. App 启动后请求 `/api/v1/check`。
2. 如果请求失败，不要直接崩溃；可以允许进入，但提示“更新检查失败”。
3. 如果 `legal.allowed=false`，阻止继续使用。
4. 如果 `update.available=false`，正常进入。
5. 如果 `update.update_type=recommended`，弹出可跳过更新弹窗。
6. 如果 `update.update_type=force`，弹出不可跳过更新弹窗。
7. 下载更新包。
8. 校验 SHA256。
9. 校验 Ed25519 manifest signature。
10. 调用平台安装逻辑。

