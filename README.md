# TradeMate

TradeMate is an agent-native Amazon operations platform. The current codebase implements the first delivery slice:

1. Monorepo workspace for web, extension, shared types, and API
2. Go API with MySQL-backed auth, goal, suggestion, approval, task, notification, and audit modules
3. Vue web app and Chrome extension baseline connected to API
4. OpenClaw extension with relay RPC set and browser fallback action contract stubs

## Structure

```text
apps/
  web/
  extension/
packages/
  shared-types/
services/
  api/
.openclaw/
  extensions/trademate/
docs/
```

## Development

1. Copy `.env.example` to `.env`
2. Start infra with `docker compose up -d`
3. Install packages with `pnpm install`
4. Run migration with `pnpm migrate:up` (or `make migrate-up`)
5. Start all services with `make dev`

Useful commands:

1. `make dev-api` / `make dev-web` / `make dev-extension` / `make dev-worker`
2. `make migrate-up` / `make migrate-down` / `make migrate-reset`
3. `make worker-once` (run task worker once)
4. `pnpm typecheck` / `pnpm build`

## Current status

The repository currently focuses on a DB-backed platform baseline:

1. Login with seed account and JWT session
2. View current user and store context from MySQL
3. Create and update ad goals (persisted)
4. Goal CRUD and store list APIs are available for management scenarios
5. Fetch suggestion list and detail
6. Approve/reject suggestions and generate tasks
7. List/cancel/retry tasks with status transition checks
8. Task Worker closes the loop from `queued -> running -> succeeded/failed`
9. Manual task run trigger endpoint: `POST /api/v1/tasks/run-once`
10. Notification list/read and WebSocket push (`/api/v1/ws`)
11. Task review snapshot generation and review query endpoint (`/api/v1/agents/ad/reviews/:task_id`)
12. Ads data preview endpoint with API client + mock fallback (`/api/v1/agents/ad/data-preview`)
13. Web pages for dashboard, approvals, tasks center, review center, notifications, audit logs, and goals
14. Extension popup supports suggestions/tasks/reviews/notifications workflows and configurable options

## API conventions

1. Base path: `/api/v1`
2. Auth: `Authorization: Bearer <token>`
3. Response envelope and fields use `snake_case`

## CI

1. GitHub Actions workflow: `.github/workflows/ci.yml`
2. Runs Node checks (`pnpm typecheck`, `pnpm build`) and Go checks (`go test ./...`)
