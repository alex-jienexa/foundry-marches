# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src

COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download

COPY src/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/foundry-api ./cmd/foundry-api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app -h /nonexistent app
WORKDIR /app
COPY --from=build --chown=app:app /out/foundry-api /app/foundry-api

USER app:app
EXPOSE 8080
ENTRYPOINT ["/app/foundry-api"]
