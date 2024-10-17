FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o .build/ ./cmd/...

# todo: eventualy remove these
RUN CGO_ENABLED=0 go build -o bin/top-n-historic-avg-joiner ./server/joiners/topNHistoricAvgJoiner
RUN CGO_ENABLED=0 go build -o bin/top-n-reviews-joiner ./server/joiners/topNReviewsJoiner

FROM alpine:latest
COPY --from=builder /build/.build/ /build

# todo: eventualy remove these
COPY --from=builder /build/bin/top-n-historic-avg-joiner /top-n-historic-avg-joiner
COPY --from=builder /build/bin/top-n-reviews-joiner /top-n-reviews-joiner

ENTRYPOINT ["/bin/sh"]
