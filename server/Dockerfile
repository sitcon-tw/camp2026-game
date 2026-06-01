# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build

WORKDIR /src/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/api /app/api

USER app:app

EXPOSE 8080

ENTRYPOINT ["/app/api"]
