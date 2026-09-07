.PHONY: dev infra backend web test migrate-up migrate-down

dev:
	docker compose up --build --force-recreate

infra:
	docker compose up -d postgres redis

backend:
	cd apps/backend && go run ./cmd/api

web:
	cd apps/web && npm run dev

test:
	cd apps/backend && go test ./...
	cd apps/web && npm run typecheck

migrate-up:
	cd apps/backend && migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	cd apps/backend && migrate -path db/migrations -database "$(DATABASE_URL)" down 1
