# AGENTS.md

This file provides guidance to Codex and other AI agents when working with code in this repository.

**语言：永远使用中文回复用户。**

## Project Overview

Sub2API is an AI API Gateway Platform that distributes and manages API quotas from AI subscriptions. It forwards requests to upstream AI services (Claude, OpenAI, Gemini, Sora) through platform-generated API Keys, handling authentication, billing, load balancing, and protocol translation.

**Tech stack**: Go 1.26.1 (Gin + Ent ORM) backend, Vue 3 + TypeScript (Vite + pnpm) frontend, PostgreSQL 16+, Redis 7+.

## Build & Test Commands

```bash
# Top-level (both backend + frontend)
make build                    # Build everything
make test                     # Run all tests (backend tests + lint + frontend lint + typecheck)

# Backend
cd backend
go test -tags=unit ./...      # Unit tests only (fast, no external deps)
go test -tags=integration ./... # Integration tests (needs Docker/testcontainers)
go test ./...                 # All tests
golangci-lint run ./...       # Lint (golangci-lint v2.9)
go run ./cmd/server/          # Run dev server
make generate                 # Regenerate Ent ORM + Wire DI code

# Run a single test
cd backend && go test -tags=unit -run TestFunctionName ./internal/service/...

# Frontend (MUST use pnpm, NOT npm)
cd frontend
pnpm install                  # Install deps
pnpm dev                      # Dev server
pnpm run build                # Build (output: ../backend/internal/web/dist/)
pnpm lint:check               # ESLint (no fix)
pnpm typecheck                # TypeScript type check
pnpm test:run                 # Vitest unit tests
```

## Architecture

### Layered Backend (strict dependency direction enforced by depguard)

```
Handler  ->  Service  ->  Repository  ->  DB/Redis
```

- **Handlers** (`internal/handler/`): HTTP endpoint logic. CANNOT import repository, gorm, or redis.
- **Services** (`internal/service/`): Business logic. CANNOT import repository, gorm, or redis (except ops_* services and wire.go).
- **Repository** (`internal/repository/`): Data access, caching. Direct Ent/Redis access.
- **DI**: Google Wire (`cmd/server/wire.go` -> generated `wire_gen.go`).

### API Gateway Routing

The gateway serves multiple API protocols simultaneously:

- `/v1/messages` - Anthropic Claude API (auto-routes based on group platform)
- `/v1/chat/completions` - OpenAI-compatible endpoint
- `/v1/responses` - Claude Responses API
- `/v1beta/` - Gemini API compatible endpoints
- `/api/v1/` - Admin/User management API

**Gateway request flow**: API key auth -> group assignment -> concurrency check -> quota validation -> account selection (smart scheduling) -> request transform -> upstream call with retry -> response transform -> async usage recording.

### Key Services

- **Account Service** (`account.go`, `account_service.go`): Upstream account management, scheduling, health tracking
- **Gateway Services**: Protocol-specific routing (`gateway_handler.go` for Anthropic, `openai_gateway_handler.go` for OpenAI)
- **Rate Limit Service** (`ratelimit_service.go`): Account rate limiting, 401/403 error handling -> marks accounts as error
- **OPS Services** (`ops_*.go`): Error logging, metrics, alerting, cleanup, scheduled reports
- **Background cron jobs**: Token refresh, account expiry, subscription expiry, usage cleanup, billing cache

### Data Layer

- **ORM**: Ent with schemas in `backend/ent/schema/`, generated code in `backend/ent/`
- **Key entities**: User, Account, AccountGroup, Group, APIKey, UsageLog, Setting
- **Migrations**: Ent auto-migrations on startup

### Frontend

- Vue 3 + Pinia stores (`src/stores/`) + Vue Router (`src/router/`)
- i18n: EN, CN, JA (`src/i18n/`)
- API layer in `src/api/` using axios
- Frontend builds into `backend/internal/web/dist/` and gets embedded with `-tags=embed`

## Changelog 维护规则

每次开发新功能、修复 bug 或推送到 GitHub 时，必须同步更新 changelog。

**操作步骤**:
1. 打开 `frontend/src/data/changelog/YYYY.json`
2. 在数组开头添加新条目: `{ "date": "YYYY-MM-DD", "type": "类型", "content": "改动说明" }`
3. type 可选值: `feature` / `fix` / `improvement` / `perf` / `docs`

## Critical Workflow Rules

1. **pnpm-lock.yaml must be committed** with any `package.json` change. CI uses `--frozen-lockfile`.
2. **After modifying Ent schemas** (`ent/schema/*.go`): run `cd backend && go generate ./ent` and commit generated files.
3. **After modifying DI graph**: run `cd backend && go generate ./cmd/server` to regenerate Wire bindings.
4. **Interface changes**: All test stubs/mocks implementing the interface must be updated with new methods. Search with `grep -r "type.*Stub.*struct\|type.*Mock.*struct" internal/`.
5. **PR checklist**: unit tests pass, integration tests pass, golangci-lint clean, lock file synced, ent code regenerated if schema changed, stubs updated if interface changed.

## Deployment

- **Docker**: Multi-stage build (Node -> Go -> Alpine). `docker-compose.local.yml` for local dirs, `docker-compose.yml` for named volumes.
- **Binary**: One-click install via `deploy/install.sh`, runs as systemd service.
- **Config**: YAML-based (`deploy/config.example.yaml`), env var overrides (e.g., `SERVER_PORT=8080` -> `server.port`).
