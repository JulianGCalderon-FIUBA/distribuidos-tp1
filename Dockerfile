FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o .build/ ./cmd/...

# todo: eventualy remove these
RUN CGO_ENABLED=0 go build -o bin/partitioner ./server/filters/partitioner
RUN CGO_ENABLED=0 go build -o bin/genre-filter ./server/filters/genreFilter
RUN CGO_ENABLED=0 go build -o bin/review-filter ./server/filters/reviewFilter
RUN CGO_ENABLED=0 go build -o bin/decade-filter ./server/filters/decadeFilter
RUN CGO_ENABLED=0 go build -o bin/language-filter ./server/filters/languageFilter
RUN CGO_ENABLED=0 go build -o bin/top-n-historic-avg ./server/aggregators/topNHistoricAvg
RUN CGO_ENABLED=0 go build -o bin/games-per-platform ./server/aggregators/gamesPerPlatform
RUN CGO_ENABLED=0 go build -o bin/games-per-platform-joiner ./server/joiners/gamesPerPlatformJoiner
RUN CGO_ENABLED=0 go build -o bin/top-n-historic-avg-joiner ./server/joiners/topNHistoricAvgJoiner
RUN CGO_ENABLED=0 go build -o bin/group-by-game ./server/aggregators/groupByGame
RUN CGO_ENABLED=0 go build -o bin/top-n-reviews ./server/aggregators/topNReviews
RUN CGO_ENABLED=0 go build -o bin/top-n-reviews-joiner ./server/joiners/topNReviewsJoiner
RUN CGO_ENABLED=0 go build -o bin/more-than-n-reviews ./server/aggregators/moreThanNReviews
RUN CGO_ENABLED=0 go build -o bin/90-percentile ./server/aggregators/90Percentile
RUN CGO_ENABLED=0 go build -o bin/group-joiner ./server/aggregators/groupJoiner

FROM alpine:latest
COPY --from=builder /build/.build/ /build

# todo: eventualy remove these
COPY --from=builder /build/bin/partitioner /partitioner
COPY --from=builder /build/bin/genre-filter /genre-filter
COPY --from=builder /build/bin/review-filter /review-filter
COPY --from=builder /build/bin/decade-filter /decade-filter
COPY --from=builder /build/bin/language-filter /language-filter
COPY --from=builder /build/bin/top-n-historic-avg /top-n-historic-avg
COPY --from=builder /build/bin/games-per-platform /games-per-platform
COPY --from=builder /build/bin/games-per-platform-joiner /games-per-platform-joiner
COPY --from=builder /build/bin/top-n-historic-avg-joiner /top-n-historic-avg-joiner
COPY --from=builder /build/bin/group-by-game /group-by-game
COPY --from=builder /build/bin/top-n-reviews /top-n-reviews
COPY --from=builder /build/bin/more-than-n-reviews /more-than-n-reviews
COPY --from=builder /build/bin/90-percentile /90-percentile
COPY --from=builder /build/bin/top-n-reviews-joiner /top-n-reviews-joiner
COPY --from=builder /build/bin/group-joiner /group-joiner

ENTRYPOINT ["/bin/sh"]
