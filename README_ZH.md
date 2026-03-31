# hatch

[English](README.md) | [中文](README_ZH.md)

`hatch` 是从 `sparkcloud/server` 中抽离出来的一套可复用 Go 应用框架。

Hatch 为构建 Go 服务提供了一套强约束、开箱即用的技术栈：

- `net/http` + `http.ServeMux`
- Connect RPC
- Uber Fx
- Zap / slog
- Ent
- PostgreSQL + Atlas
- Valkey
- S3 / MinIO
- OpenBao
- gocron

这个仓库同时包含可复用的框架包，以及用于初始化和维护 Hatch 应用的 `hatch` CLI。

## 安装 CLI

安装已发布的 CLI：

```bash
go install github.com/ix64/hatch/cmd/hatch@latest
```

在本仓库中进行本地开发时，也可以不安装，直接运行：

```bash
go run ./cmd/hatch --help
```

## 初始化项目

使用 `hatch init` 作为创建新服务的标准入口：

```bash
hatch init ./demo \
  --module example.com/acme/demo \
  --name "Demo Service" \
  --binary demo
```

它会生成：

- 位于 `cmd/server` 的可运行服务入口
- `hatch.toml` 项目元数据
- 配置与日志 wiring
- 基于 Fx 的 HTTP 路由注册
- Ent、Atlas 和迁移脚手架
- 本地 `proto/` 源文件与 `buf.yaml`
- 位于 `dev/compose.yaml` 的本地开发 Compose 文件

如果你正在开发 `hatch` 本身，并希望让生成项目直接指向你本地 checkout 的 `hatch`，可以加上 `--hatch-replace-path`：

```bash
hatch init ./demo \
  --module example.com/acme/demo \
  --name "Demo Service" \
  --binary demo \
  --hatch-replace-path /path/to/hatch
```

## 前置条件

生成的项目默认依赖：

- Go `1.26.x`
- Docker，用于本地 Postgres 以及迁移格式化和 lint

在生成项目目录中，运行 `hatch tools install` 安装代码生成和 lint 依赖，例如 `ent`、`buf`、`protoc-gen-go`、`protoc-gen-connect-go`、`atlas`、`golangci-lint` 和 `air`。

你也可以在 `hatch.toml` 旁边创建一个仅本机生效的 `hatch.local.toml` 来覆盖项目元数据；生成项目默认会忽略它。

生成出来的 `hatch.toml` 会包含一个 Taplo schema 指令，指向 `https://raw.githubusercontent.com/ix64/hatch/refs/heads/main/hatch.schema.json`，便于编辑器做补全和校验。

## 常见流程

生成项目后：

```bash
cd demo
hatch tools install
hatch env start
hatch gen ent
go run ./cmd/server serve
```

常用后续命令：

- `hatch build` 构建生产二进制
- `hatch start` 按 `hatch.toml` 中的 `[run].command` 运行已构建程序
- `hatch dev` 按 `hatch.toml` 中的 `[run].command` 通过 Air 启动热重载开发
- `hatch env add minio` 等命令可添加 MinIO、Mailpit、Valkey、OpenBao 等本地依赖
- `hatch migrate generate --name init` 创建一条新的 Atlas migration
- `hatch migrate apply --env dev` 将迁移应用到指定数据库环境
- `hatch gen rpc` 执行 protobuf 和 Connect 代码生成

Ent feature flag 在 `hatch.toml` 中配置：

```toml
[ent]
features = [
  "intercept",
  "sql/versioned-migration",
  "sql/modifier",
  "sql/execquery",
  "sql/upsert",
]
```

设置 `features = []` 可以关闭项目默认启用的 Ent feature 集合。

## CLI 命令

目前 `hatch` CLI 提供八组命令：

- `hatch init <dir>` 初始化一个新的 Hatch 应用
- `hatch build` 使用 `hatch.toml` 构建应用二进制
- `hatch start` 按 `hatch.toml` 中的 `[run].command` 运行构建产物
- `hatch dev` 按 `hatch.toml` 中的 `[run].command` 通过 Air 启动热重载开发
- `hatch env start|stop|clean|add` 通过 Docker Compose 管理本地开发依赖
- `hatch gen ent [--scratch]` 和 `hatch gen rpc` 用于代码生成
- `hatch migrate generate|hash|lint|apply` 用于管理 Atlas 迁移
- `hatch tools install` 用于安装本地代码生成和 lint 工具

## 生成目录结构

脚手架项目主要包含这些路径：

- `cmd/server`：服务入口和 `serve` 等运行命令
- `internal/config`：应用配置加载
- `internal/register`：Fx 模块装配、路由和服务 wiring
- `ddl/schema`：Ent schema
- `ddl/ent`：生成后的 Ent 代码
- `proto`：protobuf 源文件和 `buf.yaml`
- `ddl/composite`：用于 migration diff 的 schema dump
- `ddl/migrations`：Atlas migration 文件
- `dev/compose.yaml`：本地开发依赖
- `.air.toml`：Air 热重载配置
- `hatch.toml`：供 CLI 消费的 Hatch 项目元数据
- `hatch.schema.json`：`hatch.toml` / `hatch.local.toml` 的 JSON Schema

代码包按能力划分：

- `core`
- `logging`
- `httpserver`
- `connectrpc`
- `sql`
- `ent`
- `cache`
- `storage`
- `secret`
- `cron`
- `health`
- `observability`
- `testkit`

默认情况下，生成项目会把 `.proto` 文件保存在应用仓库中。`sparkcloud/server` 是一个更高级的参考实现，它会让 Hatch 指向独立的 proto 仓库，并支持本地 override。
