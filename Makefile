default: build

deps:
	go mod tidy
.PHONY: deps

build: deps
	go build -o .build/ ./cmd/...

.PHONY: build

docker-build:
	docker build -t "tp1:latest" .
.PHONY: compose-build

compose-up: compose-down docker-build
	docker compose -f compose.yaml up -d
.PHONY: compose-up

compose-down:
	docker compose -f compose.yaml stop -t 20
	docker compose -f compose.yaml down
.PHONY: compose-down

compose-logs:
	docker compose -f compose.yaml logs -f
.PHONY: compose-logs

write-compose:
	go run ./scripts/compose > compose.yaml
.PHONY: write-compose
