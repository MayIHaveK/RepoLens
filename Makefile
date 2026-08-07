.PHONY: dev test build web clean

dev:
	go run ./cmd/repolens serve

test:
	go test ./...

web:
	cd web && npm ci && npm run build

build: web
	go build -trimpath -ldflags="-s -w" -o bin/repolens.exe ./cmd/repolens

clean:
	go clean

