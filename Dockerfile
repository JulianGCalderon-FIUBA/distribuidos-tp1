FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o .build/ ./cmd/...

# todo: eventualy remove these
RUN CGO_ENABLED=0 go build -o bin/top-n-historic-avg ./server/aggregators/topNHistoricAvg
RUN CGO_ENABLED=0 go build -o bin/games-per-platform ./server/aggregators/gamesPerPlatform
RUN CGO_ENABLED=0 go build -o bin/games-per-platform-joiner ./server/joiners/gamesPerPlatformJoiner
RUN CGO_ENABLED=0 go build -o bin/top-n-historic-avg-joiner ./server/joiners/topNHistoricAvgJoiner
RUN CGO_ENABLED=0 go build -o bin/top-n-reviews-joiner ./server/joiners/topNReviewsJoiner

FROM alpine:latest
COPY --from=builder /build/.build/ /build

# todo: eventualy remove these
COPY --from=builder /build/bin/top-n-historic-avg /top-n-historic-avg
COPY --from=builder /build/bin/games-per-platform /games-per-platform
COPY --from=builder /build/bin/games-per-platform-joiner /games-per-platform-joiner
COPY --from=builder /build/bin/top-n-historic-avg-joiner /top-n-historic-avg-joiner
COPY --from=builder /build/bin/top-n-reviews-joiner /top-n-reviews-joiner

ENTRYPOINT ["/bin/sh"]
