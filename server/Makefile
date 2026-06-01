.PHONY: dev run generate swagger fmt test tidy

dev:
	air

run:
	go run ./cmd/api

generate: swagger

swagger:
	swag init -g main.go -d cmd/api,internal/http -o docs --parseInternal

fmt:
	go fmt ./...

test:
	go test ./...

tidy:
	go mod tidy
