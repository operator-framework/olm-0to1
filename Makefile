.PHONY: build test e2e-smoke

build:
	mkdir -p bin
	go build -o bin/migrate-catalogs-v0-to-v1 ./cmd/migrate-catalogs-v0-to-v1
	go build -o bin/migrate-operators-v0-to-v1 ./cmd/migrate-operators-v0-to-v1

test:
	go test ./cmd/...

# Run against a dedicated Kind cluster prepared with library-olm's fixture setup.
e2e-smoke: build
	go test -tags=e2e -count=1 -timeout=20m ./test/e2e -run '^TestMigrationCLISmoke$$'
