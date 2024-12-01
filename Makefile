default: build

deps:
	go mod tidy
.PHONY: deps

clean:
	docker run --rm -v $(shell pwd):/work -w /work alpine sh -c 'rm -rf .backup .results-*'
.PHONY: clean

build: deps
	go build -o .build/ ./cmd/...

.PHONY: build

docker-build:
	docker build -t "tp1:latest" .
.PHONY: compose-build

docker-build-stress:
	docker build --build-arg GO_TAGS=stress -t "tp1:latest" .
.PHONY: compose-build

compose-up:
	docker compose -f compose.yaml up -d
.PHONY: compose-up

compose-down:
	docker compose -f compose.yaml stop -t 2
	docker compose -f compose.yaml down --remove-orphans
.PHONY: compose-down

compose-logs:
	docker compose -f compose.yaml logs -f
.PHONY: compose-logs

run: docker-build compose-down clean compose-up compose-logs
.PHONY: run

run-stress: clean docker-build-stress compose-down compose-up compose-logs
.PHONY: run-stress

write-compose:
	go run ./scripts/compose > compose.yaml
.PHONY: write-compose

write-compose-no-volume:
	go run ./scripts/compose -volumes=false > compose.yaml
.PHONY: write-compose

docker-tree:
	tail -n +2 .node-config.csv | xargs -I _ docker exec _ sh -c 'printf "\n--- _ ---\n\n"; tree *'
.PHONY: write-compose
