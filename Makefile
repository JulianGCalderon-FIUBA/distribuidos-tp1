default: build

deps:
	go mod tidy
.PHONY: deps

build: deps
	go build -o bin/client ./client
	go build -o bin/gateway ./server/gateway
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
	docker compose -f compose.yaml logs -f gateway client \
		genre-filter review-filter decade-filter language-filter \
		q1-partitioner q1-1 q1-2 q1-joiner \
		q2-games-partitioner q2-1 q2-joiner \
	q3-games-partitioner q3-reviews-partitioner q3-group-1 q3-group-2 q3-1 q3-joiner \
		q4-games-partitioner q4-reviews-partitioner q4-group-1 q4 \
		q5-games-partitioner q5-reviews-partitioner q5-group-1 q5
.PHONY: compose-logs
