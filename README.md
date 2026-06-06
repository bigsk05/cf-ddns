# cf-ddns

[中文](#中文) | [English](#english)

# 中文

一个用 Go 编写的轻量级 Cloudflare DDNS（动态 DNS）客户端。它会定期检测本机的公网 IP（IPv4 / IPv6），并自动更新到 Cloudflare 上指定的 DNS 记录。

## 功能特性

- 支持 IPv4（A 记录）与 IPv6（AAAA 记录），可分别开启或关闭
- 多种公网 IP 查询源：内置 `ipip.net`、`gh.ink`，也支持任意自定义 Raw API
- 记录不存在时自动创建，存在时自动更新
- 基于 cron 的定时检测，周期可配置
- 通过 `SIGHUP` 信号热重载配置
- 结构化日志（zap），支持控制台输出与按大小滚动的文件日志（lumberjack）
- 所有 IP 查询请求都会带上标识性 `User-Agent`（如 `CF-DDNS/2.2.0 (linux; amd64)`）

## 环境要求

- Go 1.26 及以上
- 一个具备 DNS 编辑权限的 Cloudflare API Token

## 安装与构建

```bash
git clone <repo-url>
cd cf-ddns
go build -o cf-ddns .
```

## 配置

程序启动时会在工作目录下读取 `config.yaml`。若同时存在 `config_debug.yaml`，则会在加载 `config.yaml` 之后再加载它进行覆盖，并自动开启 Debug 日志级别。

`config.yaml` 示例：

```yaml
cf:
  token: "apiToken"      # Cloudflare API Token
  zone: "zoneID"         # 目标域名所在的 Zone ID
api:
  # 取值可以是 "ipip.net"、"gh.ink" 或自定义 API
  # "ipip.net" 与 "gh.ink" 均支持 IPv4 与 IPv6
  # 自定义 API 必须是返回单个 IP 字符串的 Raw 接口（如响应体为 "1.1.1.1"）
  # 自定义 API 地址需要带协议（http:// 或 https://）
  # 留空表示关闭对应协议
  v4: "ipip.net"
  v6: "https://example.com"
record:
  name: "test.example.com"  # 需要维护的 DNS 记录名
  period: 60                # 检测周期，单位：秒
log:
  file:
    all: "logs/app.log"     # 全量日志文件，留空则不写文件
    err: "logs/err.log"     # 仅 Warn 及以上级别的日志文件，留空则不写文件
  max_size: 5               # 单个日志文件最大大小（MB），超过后滚动
  max_backups: 100          # 保留的历史日志文件数量
  max_age: 30               # 历史日志保留天数
  compress: true            # 是否压缩历史日志
```

### 配置项说明

| 字段 | 说明 |
| --- | --- |
| `cf.token` | Cloudflare API Token，需要有目标 Zone 的 DNS 编辑权限 |
| `cf.zone` | 目标域名的 Zone ID |
| `api.v4` | IPv4 查询源：`ipip.net` / `gh.ink` / 自定义 Raw API URL，留空关闭 IPv4 |
| `api.v6` | IPv6 查询源：`ipip.net` / `gh.ink` / 自定义 Raw API URL，留空关闭 IPv6 |
| `record.name` | 要创建/更新的 DNS 记录名称（如 `home.example.com`） |
| `record.period` | 定时检测周期（秒） |
| `log.file.all` | 全量日志文件路径，留空则只输出到控制台 |
| `log.file.err` | 错误日志文件路径（Warn 及以上），留空则不单独记录 |
| `log.max_size` / `max_backups` / `max_age` / `compress` | 日志滚动策略 |

### 关于 IP 查询源

- `ipip.net`：请求 `https://myip.ipip.net` 并用正则提取 IP，支持 IPv4 与 IPv6。
- `gh.ink`：IPv4 使用 `https://v4-myip.gh.ink`，IPv6 使用 `https://v6-myip.gh.ink`。
- 自定义 Raw API：响应体应仅为单个 IP 字符串（结尾可带空白，程序会自动去除）。请求会校验 HTTP 状态码为 200。

> **协议绑定**：由于 `myip.ipip.net` 等站点现已双栈（同时有 A/AAAA 记录），不加约束时返回的 IP 会随系统选路而随机。为此程序内部为 IPv4 与 IPv6 各维护一个 HTTP 客户端，分别强制只走 `tcp4` / `tcp6` 拨号。因此 v4 查询一定返回 IPv4、v6 查询一定返回 IPv6；若本机没有对应协议的出口地址，该次查询会直接报错（不会误返回另一族地址）。此约束对所有查询源（ipip.net、gh.ink、自定义 Raw API）一致生效。

## 运行

```bash
./cf-ddns
```

程序会：

1. 加载配置并启动配置热重载监听（`SIGHUP`）
2. 初始化日志与 Cloudflare API 客户端
3. 注册定时任务，按 `record.period` 周期执行检测与更新
4. 主协程阻塞，持续运行

### 热重载配置

修改 `config.yaml` 后，向进程发送 `SIGHUP` 即可重新加载配置：

```bash
kill -HUP <pid>
```

> 热重载会更新 `config.Get()` 返回的全部配置（如 Cloudflare token、记录名、IP 源等），并在下一次检测时生效。`record.period` 变化时，cron 任务也会自动按新周期重新注册，无需重启进程。

### Debug 模式

在工作目录放置 `config_debug.yaml` 即可进入 Debug 模式：日志级别变为 `Debug`，且该文件的配置会覆盖 `config.yaml` 中的同名项。适合本地调试。

## 工作原理

每个检测周期内，`Checker` 会：

1. 根据 `api.v4` / `api.v6` 配置查询当前公网 IP（任一为空则跳过该协议）。
2. 通过 Cloudflare API 列出对应类型（A / AAAA）且同名的 DNS 记录。
3. 若记录不存在则创建（TTL 60，未开启代理）；若已存在则更新第一条记录的内容。
4. 全程记录结构化日志，失败时仅告警并继续，不会中断后续轮次。

## 项目结构

```
.
├── main.go                     # 程序入口：初始化各模块并阻塞运行
├── config.yaml                 # 运行配置
└── internal/
    ├── checker/
    │   ├── checker.go          # 检测任务编排（IPv4 / IPv6 分支）
    │   ├── get.go              # 公网 IP 查询（含 User-Agent 设置）
    │   └── update.go           # Cloudflare DNS 记录的创建/更新
    ├── config/
    │   ├── model.go            # 配置结构体定义
    │   └── viper.go            # 配置加载、热重载、信号监听
    ├── cron/
    │   └── cron.go             # 定时任务调度
    ├── logger/
    │   └── zap.go              # zap + lumberjack 日志初始化
    └── meta/
        └── version.go          # 版本号与 User-Agent
```

## 依赖

- [cloudflare-go](https://github.com/cloudflare/cloudflare-go) — Cloudflare API 客户端
- [gocron](https://github.com/go-co-op/gocron) — 定时任务调度
- [viper](https://github.com/spf13/viper) — 配置管理
- [zap](https://go.uber.org/zap) — 结构化日志
- [lumberjack](https://gopkg.in/natefinch/lumberjack.v2) — 日志文件滚动

## 安全提示

`config.yaml` 中包含 Cloudflare API Token，请勿提交到版本库或公开分享。建议为该 Token 仅授予目标 Zone 的最小 DNS 编辑权限。

---

# English

# cf-ddns (English)

A lightweight Cloudflare DDNS (Dynamic DNS) client written in Go. It periodically detects the machine's public IP (IPv4 / IPv6) and automatically updates the specified DNS record on Cloudflare.

## Features

- Supports both IPv4 (A records) and IPv6 (AAAA records); each can be enabled or disabled independently
- Multiple public-IP query sources: built-in `ipip.net` and `gh.ink`, plus any custom Raw API
- Automatically creates the record if it does not exist, or updates it if it does
- Cron-based periodic checks with a configurable interval
- Hot-reload of configuration via the `SIGHUP` signal
- Structured logging (zap) with console output and size-rolling file logs (lumberjack)
- Every IP query request carries an identifying `User-Agent` (e.g. `CF-DDNS/2.2.0 (linux; amd64)`)

## Requirements

- Go 1.26 or later
- A Cloudflare API Token with DNS edit permission

## Install & Build

```bash
git clone <repo-url>
cd cf-ddns
go build -o cf-ddns .
```

## Configuration

On startup the program reads `config.yaml` from the working directory. If `config_debug.yaml` also exists, it is loaded after `config.yaml` to override values, and the Debug log level is enabled automatically.

Example `config.yaml`:

```yaml
cf:
  token: "apiToken"      # Cloudflare API Token
  zone: "zoneID"         # Zone ID of the target domain
api:
  # Value can be "ipip.net", "gh.ink", or a custom API
  # Both "ipip.net" and "gh.ink" support IPv4 and IPv6
  # A custom API must be a Raw endpoint returning a single IP string (e.g. body "1.1.1.1")
  # A custom API address must include the protocol (http:// or https://)
  # Leave blank to disable the corresponding protocol
  v4: "ipip.net"
  v6: "https://example.com"
record:
  name: "test.example.com"  # DNS record name to maintain
  period: 60                # Check interval, in seconds
log:
  file:
    all: "logs/app.log"     # Full log file; leave blank to disable file output
    err: "logs/err.log"     # Warn-and-above log file; leave blank to disable
  max_size: 5               # Max size (MB) of a single log file before rolling
  max_backups: 100          # Number of rotated log files to keep
  max_age: 30               # Days to retain rotated logs
  compress: true            # Whether to compress rotated logs
```

### Configuration reference

| Field | Description |
| --- | --- |
| `cf.token` | Cloudflare API Token; must have DNS edit permission for the target Zone |
| `cf.zone` | Zone ID of the target domain |
| `api.v4` | IPv4 source: `ipip.net` / `gh.ink` / custom Raw API URL; blank disables IPv4 |
| `api.v6` | IPv6 source: `ipip.net` / `gh.ink` / custom Raw API URL; blank disables IPv6 |
| `record.name` | DNS record name to create/update (e.g. `home.example.com`) |
| `record.period` | Check interval (seconds) |
| `log.file.all` | Path of the full log file; blank means console output only |
| `log.file.err` | Path of the error log file (Warn and above); blank disables it |
| `log.max_size` / `max_backups` / `max_age` / `compress` | Log rotation policy |

### About IP query sources

- `ipip.net`: requests `https://myip.ipip.net` and extracts the IP with a regex; supports both IPv4 and IPv6.
- `gh.ink`: IPv4 uses `https://v4-myip.gh.ink`, IPv6 uses `https://v6-myip.gh.ink`.
- Custom Raw API: the response body should be a single IP string (trailing whitespace is trimmed automatically). The HTTP status code is validated to be 200.

> **Protocol pinning**: because sites like `myip.ipip.net` are now dual-stack (they have both A and AAAA records), the returned IP would otherwise be non-deterministic depending on the OS routing choice. To avoid this, the program keeps a separate HTTP client for IPv4 and IPv6, each forced to dial over `tcp4` / `tcp6` only. A v4 query therefore always returns an IPv4 address and a v6 query an IPv6 address; if the host has no egress address of that family, that query fails outright (it never falls back to the other family). This applies uniformly to all sources (ipip.net, gh.ink, and custom Raw APIs).

## Running

```bash
./cf-ddns
```

The program will:

1. Load the config and start the hot-reload listener (`SIGHUP`)
2. Initialise the logger and the Cloudflare API client
3. Register the scheduled task and run the check/update on each `record.period` interval
4. Block the main goroutine and keep running

### Hot-reloading configuration

After editing `config.yaml`, send `SIGHUP` to the process to reload:

```bash
kill -HUP <pid>
```

> Hot-reload updates the entire config returned by `config.Get()` (Cloudflare token, record name, IP sources, etc.) and takes effect on the next check. When `record.period` changes, the cron task is automatically re-registered with the new interval—no restart required.

### Debug mode

Place a `config_debug.yaml` in the working directory to enter Debug mode: the log level becomes `Debug`, and the values in that file override the matching keys in `config.yaml`. Useful for local debugging.

## How it works

In each check cycle, `Checker` will:

1. Query the current public IP based on `api.v4` / `api.v6` (skip a protocol if its value is blank).
2. List the same-named DNS records of the matching type (A / AAAA) via the Cloudflare API.
3. Create the record if it does not exist (TTL 60, proxy disabled); otherwise update the content of the first matching record.
4. Log everything in a structured form; on failure it only warns and continues, without interrupting subsequent cycles.

## Project structure

```
.
├── main.go                     # Entry point: initializes modules and blocks
├── config.yaml                 # Runtime configuration
└── internal/
    ├── checker/
    │   ├── checker.go          # Check orchestration (IPv4 / IPv6 branches)
    │   ├── get.go              # Public IP query (with User-Agent set)
    │   └── update.go           # Create/update Cloudflare DNS records
    ├── config/
    │   ├── model.go            # Config struct definitions
    │   └── viper.go            # Config loading, hot-reload, signal handling
    ├── cron/
    │   └── cron.go             # Scheduled task management
    ├── logger/
    │   └── zap.go              # zap + lumberjack logger setup
    └── meta/
        └── version.go          # Version and User-Agent
```

## Dependencies

- [cloudflare-go](https://github.com/cloudflare/cloudflare-go) — Cloudflare API client
- [gocron](https://github.com/go-co-op/gocron) — scheduled task management
- [viper](https://github.com/spf13/viper) — configuration management
- [zap](https://go.uber.org/zap) — structured logging
- [lumberjack](https://gopkg.in/natefinch/lumberjack.v2) — log file rotation

## Security note

`config.yaml` contains your Cloudflare API Token. Do not commit it to version control or share it publicly. It is recommended to grant the token only the minimum DNS edit permission for the target Zone.
