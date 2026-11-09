# olm-0to1

Command line tools for migrating OLMv0 catalogs and operators to OLMv1. The
commands live in this repository under `cmd/`. Their reusable migration logic
comes from the pinned `github.com/operator-framework/library-olm` Go module.

## Build

Go 1.26.5 or newer is required.

```sh
make build
```

This creates `bin/migrate-catalogs-v0-to-v1` and
`bin/migrate-operators-v0-to-v1`. Run `make test` to test both command packages.

## Use

Create OLMv1 ClusterCatalogs before migrating operators:

```sh
bin/migrate-catalogs-v0-to-v1 --dry-run
bin/migrate-catalogs-v0-to-v1
bin/migrate-operators-v0-to-v1 check --all
bin/migrate-operators-v0-to-v1 convert my-operator -n operators --dry-run
```

The catalog command supports `--kubeconfig`, `--output`, and optional source
CatalogSource deletion. The operator command has `check`, `convert`, `rollback`,
and `cleanup` subcommands. Use each command's `--help` for all flags and safety
acknowledgments. Migration behavior and prerequisites are documented in the
[library-olm migration guide](https://github.com/operator-framework/library-olm/tree/main/specs/20260821-migration-v0-to-v1).

## Development

The command sources were synced from `library-olm/migration/examples/cmd/` at
revision `4c37ce218c2eecf9eb5231e038340ce63012ab99`. The deterministic
dry-run preview fix is also proposed in
[library-olm PR #89](https://github.com/operator-framework/library-olm/pull/89).
Update the `library-olm` version in `go.mod` when taking future library
changes, then run `go mod tidy` and `make test`.
