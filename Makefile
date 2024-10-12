default: build

deps:
	go mod tidy
.PHONY: deps

build: deps
	go build -o .build/ ./cmd/...

	# todo: eventualy remove these
	go build -o bin/partitioner ./server/filters/partitioner
	go build -o bin/genre-filter ./server/filters/genreFilter
	go build -o bin/review-filter ./server/filters/reviewFilter
	go build -o bin/decade-filter ./server/filters/decadeFilter
	go build -o bin/language-filter ./server/filters/languageFilter
	go build -o bin/games-per-platform ./server/aggregators/gamesPerPlatform
	go build -o bin/games-per-platform-joiner ./server/joiners/gamesPerPlatformJoiner
	go build -o bin/group-by-game ./server/aggregators/groupByGame
	go build -o bin/more-than-n-reviews ./server/aggregators/moreThanNReviews
	go build -o bin/90-percentile ./server/aggregators/90Percentile
	go build -o bin/top-n-historic-avg ./server/aggregators/topNHistoricAvg
	go build -o bin/top-n-reviews-joiner ./server/joiners/topNReviewsJoiner
	go build -o bin/top-n-historic-avg-joiner ./server/joiners/topNHistoricAvgJoiner

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
	docker compose -f compose.yaml logs -f
.PHONY: compose-logs

write-compose:
	go run ./scripts/compose > compose.yaml
.PHONY: write-compose
