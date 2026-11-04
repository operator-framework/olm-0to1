.PHONY: build test

build:
	mkdir -p bin
	go build -o bin/migrate-catalogs-v0-to-v1 ./cmd/migrate-catalogs-v0-to-v1
	go build -o bin/migrate-operators-v0-to-v1 ./cmd/migrate-operators-v0-to-v1

test:
	go test ./cmd/...
