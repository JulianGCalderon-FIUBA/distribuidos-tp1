FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum .
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o bin/client ./client
RUN CGO_ENABLED=0 go build -o bin/gateway ./server/gateway
RUN CGO_ENABLED=0 go build -o bin/partitioner ./server/filters/partitioner
RUN CGO_ENABLED=0 go build -o bin/genre-filter ./server/filters/genreFilter
RUN CGO_ENABLED=0 go build -o bin/review-filter ./server/filters/reviewFilter
RUN CGO_ENABLED=0 go build -o bin/decade-filter ./server/filters/decadeFilter
RUN CGO_ENABLED=0 go build -o bin/language-filter ./server/filters/languageFilter
RUN CGO_ENABLED=0 go build -o bin/games-per-platform ./server/aggregators/gamesPerPlatform
RUN CGO_ENABLED=0 go build -o bin/games-per-platform-joiner ./server/joiners/gamesPerPlatformJoiner

FROM alpine:latest
COPY --from=builder /build/bin/client /client
COPY --from=builder /build/bin/gateway /gateway
COPY --from=builder /build/bin/partitioner /partitioner
COPY --from=builder /build/bin/genre-filter /genre-filter
COPY --from=builder /build/bin/review-filter /review-filter
COPY --from=builder /build/bin/decade-filter /decade-filter
COPY --from=builder /build/bin/language-filter /language-filter
COPY --from=builder /build/bin/games-per-platform /games-per-platform
COPY --from=builder /build/bin/games-per-platform-joiner /games-per-platform-joiner
ENTRYPOINT ["/bin/sh"]
