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

## E2E smoke test

`make e2e-smoke` builds both binaries and tests them against a dedicated Kind
cluster with the `ecr-secret-operator` OLMv0 fixture. The test follows the
catalog dry-run and migration, operator check and dry-run, conversion, and
acknowledged rollback. It verifies JSONL results, the resulting OLMv1 objects,
and preservation of the operator Deployment. It refuses a cluster whose
current context is not `kind-$E2E_CLUSTER_NAME` or that has the OLMv0 controller.

CI provisions the cluster with the committed fixtures from the pinned
`library-olm` revision in `go.mod`; the binaries under test are built from this
repository. To run locally, prepare that fixture cluster using
`library-olm`'s `migration/e2e-fixture-setup` and
`migration/e2e-install-fixture-v0 E2E_OPERATOR=ecr-secret-operator` targets,
then set `E2E_CLUSTER_NAME` and `E2E_KUBECONFIG` and run `make e2e-smoke`.
Use `library-olm`'s `migration/e2e-teardown` target to delete the cluster.

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
