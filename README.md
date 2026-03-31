# hatch

`hatch` is a reusable Go application framework extracted from `sparkcloud/server`.

Hatch provides a strongly opinionated stack for building Go services with:

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

The repository contains both the reusable framework packages and the `hatch` CLI used to scaffold and maintain Hatch applications.

## Install the CLI

Install the published CLI:

```bash
go install github.com/ix64/hatch/cmd/hatch@latest
```

During local development inside this repository, you can also run the CLI without installing it:

```bash
go run ./cmd/hatch --help
```

## Bootstrap a project

Use `hatch init` as the canonical way to start a new service:

```bash
hatch init ./demo \
  --module example.com/acme/demo \
  --name "Demo Service" \
  --binary demo
```

This generates:

- a runnable server entrypoint under `cmd/server`
- `hatch.toml` project metadata
- config and logging wiring
- HTTP route registration via Fx
- Ent, Atlas, and migration scaffolding
- local `proto/` sources with `buf.yaml`
- a local development Compose file at `dev/compose.yaml`

When developing `hatch` itself and testing generated projects against your local checkout, add `--hatch-replace-path`:

```bash
hatch init ./demo \
  --module example.com/acme/demo \
  --name "Demo Service" \
  --binary demo \
  --hatch-replace-path /path/to/hatch
```

## Prerequisites

The generated project assumes:

- Go `1.26.x`
- Docker for local Postgres and migration formatting/linting

Inside a generated project, run `hatch tools install` to install codegen and lint dependencies such as `ent`, `buf`, `protoc-gen-go`, `protoc-gen-connect-go`, `atlas`, `golangci-lint`, and `air`.
You can create a local-only `hatch.local.toml` alongside `hatch.toml` to override project metadata on your machine; generated projects ignore it by default.
Generated `hatch.toml` files include a Taplo schema directive pointing at `https://github.com/ix64/hatch/raw/main/hatch.schema.json` for editor completion and validation.

## Typical workflow

After generating a project:

```bash
cd demo
hatch tools install
hatch env start --project-dir .
hatch gen ent --project-dir .
go run ./cmd/server serve
```

Common follow-up commands:

- `hatch build --project-dir .` builds the production binary
- `hatch start --project-dir .` runs the built binary using `[run].command` from `hatch.toml`
- `hatch dev --project-dir .` runs the app with live reload through Air using `[run].command` from `hatch.toml`
- `hatch env add minio --project-dir .` and similar commands add local dependencies such as MinIO, Mailpit, Valkey, and OpenBao
- `hatch migrate generate --project-dir . --name init` creates a new Atlas migration
- `hatch migrate apply --project-dir . --env dev` applies migrations to the configured database
- `hatch gen rpc --project-dir .` runs protobuf and Connect code generation

Ent feature flags are configured in `hatch.toml`:

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

Set `features = []` to disable the default Ent feature set for a project.

## CLI commands

The `hatch` CLI currently provides eight command groups:

- `hatch init <dir>` initializes a new Hatch application
- `hatch build` builds the application binary using `hatch.toml`
- `hatch start` runs the built application binary using `[run].command` from `hatch.toml`
- `hatch dev` runs the application with live reload via Air using `[run].command` from `hatch.toml`
- `hatch env start|stop|clean|add` manages local development dependencies via Docker Compose
- `hatch gen ent [--scratch]` and `hatch gen rpc` manage code generation
- `hatch migrate generate|hash|lint|apply` manages Atlas migrations
- `hatch tools install` installs local codegen and lint tools

## Generated layout

The scaffolded project is organized around these key paths:

- `cmd/server`: service entrypoint and runtime commands such as `serve`
- `internal/config`: application config loading
- `internal/register`: Fx module assembly, routes, and server wiring
- `ddl/schema`: Ent schemas
- `ddl/ent`: generated Ent code
- `proto`: protobuf source files and `buf.yaml`
- `ddl/composite`: schema dumps used for migration diffing
- `ddl/migrations`: Atlas migration files
- `dev/compose.yaml`: local development dependencies
- `.air.toml`: Air live-reload configuration
- `hatch.toml`: Hatch project metadata consumed by CLI commands

Packages are organized by capability:

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

By default, generated projects keep `.proto` files in the application repository. `sparkcloud/server` is the advanced reference app that instead points Hatch at a separate proto repository with a local override.
