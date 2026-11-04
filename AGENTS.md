# Repository Guidelines

## Project Structure & Module Organization

`olm-0to1` is the standalone Go module for the OLMv0-to-OLMv1 migration CLIs.
The catalog and operator commands live in `cmd/migrate-catalogs-v0-to-v1/` and
`cmd/migrate-operators-v0-to-v1/`. Reusable migration logic stays in
`github.com/operator-framework/library-olm/migration/pkg/` and is consumed as
a Go module dependency. Keep command-specific code and tests beside each
command. Put changes to shared migration behavior in `library-olm`.

## Build, Test, and Development Commands

- `make build` builds both CLI binaries into `bin/`.
- `make test` runs unit tests for both command packages (`go test ./cmd/...`).
- `go vet ./cmd/...` checks the command packages.
- `go mod tidy` updates module requirements and checksums after dependency changes.

## Coding Style & Testing

Use `gofmt` and idiomatic Go names. Exported APIs use `CamelCase`, helpers use
`camelCase`, and tests follow `TestThingBehavior`. Wrap errors with useful
context and preserve error identity with `%w`. Keep Kubernetes resource
mutations explicit and validate destructive preconditions before changing
OLMv0 resources.

Add unit coverage beside affected command code. Use fake controller-runtime
clients where practical. The cluster E2E suites live in `library-olm`; run or
update them there when a change affects migration behavior or Kubernetes
integration. Do not treat CLI unit tests as cluster-level validation.

## Commit & Pull Request Guidelines

Use concise conventional-style subjects such as `fix: ...`, `feat: ...`,
`test: ...`, or `docs: ...`. Sign commits off with `git commit --signoff` and
preserve existing trailers. Keep PRs focused; describe behavior and risk, link
related issues or upstream changes, list validation commands, and call out
required E2E or documentation updates.
