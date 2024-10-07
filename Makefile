default: build

deps:
	go mod tidy
.PHONY: deps

build: deps
	go build -o bin/client ./client
	go build -o bin/gateway ./server/gateway
	go build -o bin/game-partitioner ./server/filters/gamePartitioner
	go build -o bin/review-partitioner ./server/filters/reviewPartitioner
	go build -o bin/genre-filter ./server/filters/genreFilter
	go build -o bin/review-filter ./server/filters/reviewFilter
	go build -o bin/decade-filter ./server/filters/decadeFilter
	go build -o bin/language-filter ./server/filters/languageFilter
	go build -o bin/games-per-platform ./server/aggregators/gamesPerPlatform
	go build -o bin/games-per-platform-joiner ./server/joiners/gamesPerPlatformJoiner
	go build -o bin/group-by-game ./server/aggregators/groupByGame
.PHONY: build

docker-build:
	docker build -t "tp1:latest" .
.PHONY: compose-build

compose-up: compose-down docker-build
	docker compose -f compose.yaml up -d
.PHONY: compose-up

compose-down:
	docker compose -f compose.yaml stop -t 1
	docker compose -f compose.yaml down
.PHONY: compose-down

compose-logs:
	docker compose -f compose.yaml logs -f gateway client partitioner genre-filter review-filter decade-filter language-filter games-per-platform-1 games-per-platform-2 games-per-platform-3 games-per-platform-joiner top-n-amount-games-partitioner top-n-amount-reviews-partitioner
.PHONY: compose-logs
