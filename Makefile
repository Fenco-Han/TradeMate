dev-api:
	go run ./services/api/cmd/api

dev-web:
	pnpm --filter @trademate/web dev

dev-extension:
	pnpm --filter @trademate/extension dev

build:
	pnpm build

typecheck:
	pnpm typecheck
