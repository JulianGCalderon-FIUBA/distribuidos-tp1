default: build

build: build-client build-gateway
.PHONY: build

build-client:
	go build -o bin/client ./client
.PHONY: build-client

build-gateway:
	go build -o bin/gateway ./server/gateway
.PHONY: build-gateway

build-genre-filter:
	go build -o bin/genre-filter ./server/filters/genreFilter
.PHONY: build-genre-filter

build-decade-filter:
	go build -o bin/decade-filter ./server/filters/decadeFilter
.PHONY: build-decade-filter

build-reviews-filter:
	go build -o bin/reviews-filter ./server/filters/reviewFilter
.PHONY: build-reviews-filter

build-language-filter:
	go build -o bin/language-filter ./server/filters/languageFilter
.PHONY: build-language-filter

run-client: build-client
	./bin/client
.PHONY: run-client

run-gateway: build-gateway
	./bin/gateway
.PHONY: run-gateway

run-genre-filter: build-genre-filter
	./bin/genre-filter
.PHONY: run-genre-filter

run-decade-filter: build-decade-filter
	./bin/decade-filter
.PHONY: run-decade-filter

run-reviews-filter: build-reviews-filter
	./bin/reviews-filter
.PHONY: run-reviews-filter

run-language-filter: build-language-filter
	./bin/language-filter
.PHONY: run-language-filter

docker-image:
	docker build -f ./server/gateway/Dockerfile -t "gateway:latest" .
	docker build -f ./client/Dockerfile -t "client:latest" .
.PHONY: docker-image

compose-up: docker-image
	docker compose -f compose.yaml up -d
.PHONY: docker-compose-up

compose-down:
	docker compose -f compose.yaml stop -t 1
	docker compose -f compose.yaml down
.PHONY: docker-compose-down

compose-logs:
	docker compose -f compose.yaml logs -f gateway client
.PHONY: docker-compose-logs
