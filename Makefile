-include .env.example
-include .env

export

.PHONY: generate migrate migrate-down run dev lint test test-integration docker-up docker-down

generate:
	go tool oapi-codegen \
		-generate types,chi-server,spec \
		-package api \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-o api/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml
	go generate ./...

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/service

dev: migrate
	$(MAKE) run

lint:
	golangci-lint run

test:
	go test -race -count=2 -cover ./...

test-integration: migrate
	go test -race -count=2 -tags=integration ./internal/repository/trip/...

docker-up: migrate
	DATABASE_URL="$(subst @localhost:,@host.docker.internal:,$(DATABASE_URL))" \
	OTEL_EXPORTER_OTLP_ENDPOINT="$(subst localhost,host.docker.internal,$(OTEL_EXPORTER_OTLP_ENDPOINT))" \
		docker compose up --build

docker-down:
	docker compose down
