# TradeMate

TradeMate is an agent-native Amazon operations platform. The current codebase implements the first delivery slice:

1. Monorepo workspace for web, extension, shared types, and API
2. Go API with MySQL-backed auth, goal, suggestion, approval, task, notification, and audit modules
3. Vue web app and Chrome extension baseline connected to API
4. OpenClaw extension skeleton

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
4. Start web with `pnpm dev:web`
5. Start extension build watch with `pnpm dev:extension`
6. Start API with `go run ./services/api/cmd/api`

## Current status

The repository currently focuses on a DB-backed platform baseline:

1. Login with seed account and JWT session
2. View current user and store context from MySQL
3. Create and update ad goals (persisted)
4. Fetch suggestion list and detail
5. Approve/reject suggestions and generate tasks
6. List/cancel/retry tasks with status transition checks
7. Notification list/read and WebSocket push (`/api/v1/ws`)
8. Web pages for dashboard, approvals, tasks center, and goals

## API conventions

1. Base path: `/api/v1`
2. Auth: `Authorization: Bearer <token>`
3. Response envelope and fields use `snake_case`
