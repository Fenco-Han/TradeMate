.PHONY: infra-up infra-down migrate-up migrate-down migrate-reset dev dev-api dev-web dev-extension dev-worker worker-once build typecheck test

infra-up:
	docker compose up -d

infra-down:
	docker compose down

migrate-up:
	cd services/api && go run ./cmd/migrate -action up

migrate-down:
	cd services/api && go run ./cmd/migrate -action down

migrate-reset:
	cd services/api && go run ./cmd/migrate -action reset -seed

dev: infra-up migrate-up
	@set -e; \
	trap 'kill 0' INT TERM EXIT; \
	(cd services/api && go run ./cmd/api) & \
	pnpm --filter @trademate/web dev & \
	pnpm --filter @trademate/extension dev & \
	wait

dev-api:
	cd services/api && go run ./cmd/api

dev-web:
	pnpm --filter @trademate/web dev

dev-extension:
	pnpm --filter @trademate/extension dev

dev-worker:
	cd services/api && go run ./cmd/worker -mode loop -interval 10s

worker-once:
	cd services/api && go run ./cmd/worker -mode once

build:
	pnpm build

typecheck:
	pnpm typecheck

test:
	pnpm test
	cd services/api && go test ./...
