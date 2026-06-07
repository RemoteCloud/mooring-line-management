# Mooring Line Management — dev tasks.
# Backend lives in api/ (Go), frontend in web/ (React+TS, added in a later slice).

.PHONY: help build run-onboard run-shore migrate-up migrate-down openapi gen-ts \
        onboard-up onboard-down shore-up shore-down test tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?#' $(MAKEFILE_LIST) | sed 's/:.*#/\t/' | sort

build: # build the api binary
	cd api && go build -o ../bin/server ./cmd/server

test: # run backend tests
	cd api && go test ./...

tidy: # go mod tidy
	cd api && go mod tidy

run-onboard: # run api locally as onboard (needs VESSEL_ID + local db)
	cd api && SCOPE=onboard VESSEL_ID=$(VESSEL_ID) go run ./cmd/server

run-shore: # run api locally as shore (fleet)
	cd api && SCOPE=shore go run ./cmd/server

migrate-up: # apply migrations (uses DATABASE_URL or default)
	cd api && go run ./cmd/server migrate up

migrate-down: # roll back migrations
	cd api && go run ./cmd/server migrate down

seed: # load Norwegian Luna demo data (use RESET=1 to wipe first)
	cd api && go run ./cmd/server seed $(if $(RESET),--reset,)

seed-docker: # reseed inside the running onboard api container
	docker compose -f deploy/docker-compose.onboard.yml exec -T api /server seed --reset

openapi: # emit api/openapi/openapi.json (the 3.1 spec, deliverable 2)
	cd api && go run ./cmd/server dump-openapi

gen-ts: openapi # regenerate frontend TS types from the emitted spec
	cd web && npx --yes openapi-typescript@latest ../api/openapi/openapi.json -o src/api/schema.ts

onboard-up: # bring up the onboard stack (db + minio + api)
	docker compose -f deploy/docker-compose.onboard.yml up -d --build

onboard-down:
	docker compose -f deploy/docker-compose.onboard.yml down

shore-up: # bring up the shore stack (db + minio + api)
	docker compose -f deploy/docker-compose.shore.yml up -d --build

shore-down:
	docker compose -f deploy/docker-compose.shore.yml down
