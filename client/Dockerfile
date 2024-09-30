FROM golang:1.23 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o bin/client ./client

FROM alpine:latest
COPY --from=builder /build/bin/client /client
ENTRYPOINT ["/bin/sh"]
