.PHONY: dev test build web clean

VERSION ?= 0.1.0-dev

dev:
	go run ./cmd/repolens serve

test:
	go test ./...

web:
	cd web && npm ci && npm run build

build: web
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/repolens.exe ./cmd/repolens

clean:
	go clean
