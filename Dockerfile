FROM golang:1.23 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o bin/client ./client
RUN CGO_ENABLED=0 go build -o bin/gateway ./server/gateway
RUN CGO_ENABLED=0 go build -o bin/game-partitioner ./server/filters/gamePartitioner
RUN CGO_ENABLED=0 go build -o bin/review-partitioner ./server/filters/reviewPartitioner

FROM alpine:latest
COPY --from=builder /build/bin/client /client
COPY --from=builder /build/bin/gateway /gateway
COPY --from=builder /build/bin/game-partitioner /game-partitioner
COPY --from=builder /build/bin/review-partitioner /review-partitioner
ENTRYPOINT ["/bin/sh"]
