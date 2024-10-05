default: build

deps:
	go mod tidy
.PHONY: deps

build: deps
	go build -o bin/client ./client
	go build -o bin/gateway ./server/gateway
	go build -o bin/partitioner ./server/partitioner
	go build -o bin/genre-filter ./server/filters/genreFilter
	go build -o bin/review-filter ./server/filters/reviewFilter
.PHONY: build

compose-build:
	docker compose -f compose.yaml build
.PHONY: compose-build

compose-up: compose-down compose-build
	docker compose -f compose.yaml up -d
.PHONY: compose-up

compose-down:
	docker compose -f compose.yaml stop -t 1
	docker compose -f compose.yaml down
.PHONY: compose-down

compose-logs:
	docker compose -f compose.yaml logs -f gateway client partitioner
.PHONY: compose-logs
