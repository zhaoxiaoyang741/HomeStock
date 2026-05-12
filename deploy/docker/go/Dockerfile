# syntax=docker/dockerfile:1
# Stage 1: Build Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download -x

COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

# Stage 2: Minimal runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server /usr/local/bin/server

EXPOSE 8888

ENTRYPOINT ["server", "--config", "/etc/homestock/config.json"]
