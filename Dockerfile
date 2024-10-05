FROM golang:1.23 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o bin/client ./client
RUN CGO_ENABLED=0 go build -o bin/gateway ./server/gateway
RUN CGO_ENABLED=0 go build -o bin/game-partitioner ./server/filters/gamePartitioner
RUN CGO_ENABLED=0 go build -o bin/review-partitioner ./server/filters/reviewPartitioner
RUN CGO_ENABLED=0 go build -o bin/genre-filter ./server/filters/genreFilter
RUN CGO_ENABLED=0 go build -o bin/review-filter ./server/filters/reviewFilter
RUN CGO_ENABLED=0 go build -o bin/decade-filter ./server/filters/decadeFilter
RUN CGO_ENABLED=0 go build -o bin/language-filter ./server/filters/languageFilter

FROM alpine:latest
COPY --from=builder /build/bin/client /client
COPY --from=builder /build/bin/gateway /gateway
COPY --from=builder /build/bin/game-partitioner /game-partitioner
COPY --from=builder /build/bin/review-partitioner /review-partitioner
COPY --from=builder /build/bin/genre-filter /genre-filter
COPY --from=builder /build/bin/review-filter /review-filter
COPY --from=builder /build/bin/decade-filter /decade-filter
COPY --from=builder /build/bin/language-filter /language-filter
ENTRYPOINT ["/bin/sh"]
