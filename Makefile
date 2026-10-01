.PHONY: dev test build web clean

VERSION ?= 0.2.0-dev

dev:
	go run ./cmd/repolens serve

test:
	go test ./...
	cd web && npm test

web:
	cd web && npm ci && npm run build

build: web
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/repolens.exe ./cmd/repolens

clean:
	go clean
