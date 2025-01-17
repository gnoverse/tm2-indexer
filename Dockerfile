FROM golang:1.22.6-alpine AS builder

RUN apk add --no-cache git

ENV GOCACHE=/root/.cache/go-build

WORKDIR /app

COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o /build/tm2-indexer ./cmd/tm2-indexer

# Final image
FROM alpine

WORKDIR /app

COPY --from=builder /build/tm2-indexer /usr/bin/tm2-indexer

ENTRYPOINT ["/usr/bin/tm2-indexer"]