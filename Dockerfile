FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o .build/ ./cmd/...

FROM alpine:latest
COPY --from=builder /build/.build/ /build

WORKDIR /work
ENTRYPOINT ["/bin/sh"]
