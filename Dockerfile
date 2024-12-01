FROM golang:1.23 AS builder

ARG GO_TAGS=""

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -tags "${GO_TAGS}" -o .build/ ./cmd/...

FROM alpine:latest
RUN apk add docker
COPY --from=builder /build/.build/ /build

WORKDIR /work
ENTRYPOINT ["/bin/sh"]
