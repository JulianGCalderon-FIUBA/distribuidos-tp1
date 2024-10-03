default: build

deps:
	go mod tidy

build:
	go build -o bin/client ./client
	go build -o bin/partitioner ./server/partitioner
	go build -o bin/gateway ./server/gateway
.PHONY: build

compose-build:
	docker compose -f compose.yaml build
.PHONY: docker-compose-up

compose-up: compose-down compose-build
	docker compose -f compose.yaml up -d
.PHONY: docker-compose-up

compose-down:
	docker compose -f compose.yaml stop -t 1
	docker compose -f compose.yaml down
.PHONY: docker-compose-down

compose-logs:
	docker compose -f compose.yaml logs -f gateway client partitioner
.PHONY: docker-compose-logs
