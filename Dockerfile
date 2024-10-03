FROM golang:1.23 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o bin/client ./client
RUN CGO_ENABLED=0 go build -o bin/gateway ./server/gateway
RUN CGO_ENABLED=0 go build -o bin/partitioner ./server/partitioner

FROM alpine:latest
COPY --from=builder /build/bin/client /client
COPY --from=builder /build/bin/gateway /gateway
COPY --from=builder /build/bin/partitioner /partitioner
ENTRYPOINT ["/bin/sh"]
