FROM golang:1.23 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o bin/gateway ./server/gateway


FROM alpine:latest
COPY --from=builder /build/bin/gateway /gateway
ENTRYPOINT ["/bin/sh"]
