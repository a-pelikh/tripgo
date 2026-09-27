-include .env.example
-include .env

export

.PHONY: generate migrate migrate-down run dev test docker-up docker-down

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-o api/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/trip-service

dev: migrate
	$(MAKE) run

test:
	go test -race ./...

docker-up: migrate
	DATABASE_URL="$(subst @localhost:,@host.docker.internal:,$(DATABASE_URL))" docker compose up --build

docker-down:
	docker compose down
