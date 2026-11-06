# OLMv0 to OLMv1 migration CLI: interface and UX specification

**Status:** Current interface proposed in [olm-0to1 PR #1](https://github.com/operator-framework/olm-0to1/pull/1), reviewed as a separate document. The [library-olm migration design](https://github.com/operator-framework/library-olm/tree/main/specs/20260821-migration-v0-to-v1) describes the intended behavior; differences are called out below.

**Source baseline:** The two command packages in that PR and the migration packages in `operator-framework/library-olm` at `2a8dfa6ecbffc5ef69e94b238ed59456bac99d3d`. The dry-run kind-ordering change in [library-olm PR #89](https://github.com/operator-framework/library-olm/pull/89) is included.

**Audience:** Operators planning a migration, reviewers of the CLI, and authors of scripts that consume its output.

This document describes the user-visible interface as implemented. It covers invocation, selection and validation, displayed results, JSON Lines records, and exit behavior. It does not specify the internal Kubernetes migration algorithm.

## User journey and entry points

| Executable | Purpose |
| --- | --- |
| `migrate-catalogs-v0-to-v1` | Create or adopt OLMv1 catalogs from image-backed OLMv0 sources. |
| `migrate-operators-v0-to-v1` | Check, convert, roll back, or clean up operator migrations. |

The catalog executable takes flags and no positional target. The operator executable uses `migrate-operators-v0-to-v1 [global flags] <command> [target or --all] [flags]`.

The intended sequence is to run the catalog tool first, then check operators, preview conversion, and convert eligible operators:

```sh
migrate-catalogs-v0-to-v1 --dry-run
migrate-catalogs-v0-to-v1
migrate-operators-v0-to-v1 check --all
migrate-operators-v0-to-v1 convert widget-operator -n operators --dry-run
migrate-operators-v0-to-v1 convert widget-operator -n operators
```

Both tools are non-interactive. `--help` displays command help without contacting a cluster. The operator executable with no subcommand displays its command list. Cobra also provides `help` and `completion` commands for the operator executable.

### Shared conventions

- **Cluster credentials:** `--kubeconfig PATH` selects an explicit file. Otherwise normal `KUBECONFIG` and default kubeconfig loading applies. If neither the flag nor `KUBECONFIG` is set and kubeconfig loading fails, the tools try in-cluster ServiceAccount credentials. An invalid explicit configuration does not trigger this fallback.
- **Output selection:** `--output=text` is the default; `--output=jsonl` writes one JSON object per line. Other values are rejected.
- **Targets:** Operator `check` and `convert` take a Subscription name plus `-n/--namespace`, or `--all`. `rollback` and `cleanup` take a cluster-scoped ClusterExtension name, or `--all`.
- **Errors:** Text mode writes the terminal `Error: ...` line to stderr. JSONL mode writes a terminal `type: "error"` record to stdout. Successful commands exit 0; command or migration failures exit nonzero.
- **Progress:** Text mode prints colored symbols and may rewrite a waiting line using carriage returns. JSONL mode emits `progress` records. Scripts should select JSONL rather than parse the text presentation.

## Catalog migration executable

The catalog command scans `CatalogSource` objects in **all namespaces**. It accepts no positional arguments and has no subcommands.

### Options

| Option | Default | Effect |
| --- | --- | --- |
| `--kubeconfig PATH` | Normal lookup | Select cluster credentials. |
| `--output text\|jsonl` | `text` | Select presentation format. |
| `--dry-run` | Off | Preview catalog creation or adoption. |
| `--delete-catalogsource` | Off | Delete an unreferenced source after success. |
| `--acknowledge-priority-overflow` | Off | Cap out-of-range priority instead of skipping the source. |
| `-h`, `--help` | — | Show help and exit. |

With `--delete-catalogsource`, the tool rechecks whether a Subscription references the source before deletion. A dry run reports the proposed deletion without changing it. Without `--acknowledge-priority-overflow`, an out-of-range source is skipped with a reason; with it, the priority is capped at the signed 32-bit boundary.

Only `grpc` CatalogSources with `spec.image` are candidates. `configmap`, `internal`, address-only `grpc`, and other unsupported source types are reported as skipped. A source with a matching existing catalog image may be adopted. Sources with the same name and image can share a ClusterCatalog; when their images differ, proposed names include the source namespace. A name collision with a different image is reported as a per-source error. Creation waits for the new ClusterCatalog to become `Serving=True` before reporting success. Registry polling is converted to whole minutes, except that a digest image reference cannot carry a poll interval. Notes disclose source settings without an OLMv1 equivalent, including `spec.secrets` and `spec.grpcPodConfig`.

### Text presentation

The tool first announces a dry run or migration, then displays scan and per-source progress. Final outcomes are grouped in this order when present:

1. `Would migrate` for dry-run candidates, with the source `namespace/name`, target catalog, and reason.
2. `Created` with a count and source-to-target mapping.
3. `Adopted existing` with a count and mapping.
4. `Skipped` with a count and reason for each source.
5. `Errors` with a count and reason for each failed source.

Notes appear below the associated source. On success, the tool prints a count of ready ClusterCatalogs, `Dry run complete.`, or `No migratable CatalogSources found.` as applicable. Any per-source `error` outcome makes the overall command fail after outcomes are printed; `skipped` outcomes alone do not.

### JSONL catalog outcomes

Each source gets a `result` record whose `target` is `namespace/name`. Its `data` object has:

| Field | Meaning |
| --- | --- |
| `catalog_source_namespace`, `catalog_source_name` | Source identity. |
| `cluster_catalog_name` | Selected OLMv1 name, if any. |
| `outcome` | `created`, `adopted`, `skipped`, `error`, or `dry-run`. |
| `reason` | Explanation, omitted when empty. |
| `notes` | Optional notices, including source deletion. |

The record's envelope `status` is `completed` for created, adopted, and dry-run outcomes; `warning` for skipped; and `failed` for error. A failed source also has an envelope `error`. If any source fails, a final `error` record reports the total failures. Scan and per-source progress records can precede the outcomes.

An illustrative dry-run result is:

```json
{"type":"result","command":"migrate-catalogs","target":"operators/widget-catalog","status":"completed","message":"would create ClusterCatalog widget-catalog from image example.com/widget-catalog:v1","data":{"catalog_source_namespace":"operators","catalog_source_name":"widget-catalog","cluster_catalog_name":"widget-catalog","outcome":"dry-run","reason":"would create ClusterCatalog widget-catalog from image example.com/widget-catalog:v1"}}
```

## Operator migration executable

Global options `--kubeconfig PATH`, `--output text|jsonl`, and `-h/--help` apply to its subcommands. The four migration commands are:

| Command | Target | Effect |
| --- | --- | --- |
| `check` | Subscription or `--all` | Report readiness and compatibility. |
| `convert` | Subscription or `--all` | Migrate or preview with `--dry-run`. |
| `rollback` | ClusterExtension or `--all` | Restore the backed-up Subscription. |
| `cleanup` | ClusterExtension or `--all` | Remove remaining OLMv0 artifacts. |

Subscription targets require a name and namespace. `rollback` and `cleanup` take a cluster-scoped ClusterExtension name. Cleanup retains that ClusterExtension.

Every command accepts at most one positional target. A target and `--all` together are rejected; omitting both is rejected. Invalid arguments show command usage. Once execution reaches cluster access, failures report an error without repeating usage.

### `check`

```text
migrate-operators-v0-to-v1 check <subscription-name> -n <namespace>
migrate-operators-v0-to-v1 check --all
```

| Option | Effect |
| --- | --- |
| `-n`, `--namespace NAME` | Required for one Subscription. |
| `--all` | Scan Subscriptions and migrated ClusterExtensions across namespaces. |

A single check prints each readiness, compatibility, and catalog check as passed or failed, followed by the eligibility reason and warnings. A failed eligibility check is a **reported result**: if the check itself completed, the command exits 0. A cluster/API failure exits nonzero.

`check --all` groups scan results in this display order: `Conflict`, `Ineligible`, `AlreadyMigrated`, then `Eligible`. A Conflict means a Subscription and its migration-annotated ClusterExtension both exist. AlreadyMigrated means the annotated ClusterExtension exists and its source Subscription is gone. Ineligible entries show failed checks and errors; Eligible entries show source and package. Warnings can accompany entries. These classifications do not by themselves make the scan fail.

Checks cover the ClusterObjectSet API, Subscription and CSV state, package uniqueness, catalog availability, install mode and namespace scope, APIService and dependency compatibility, OLMv0 API access, OperatorCondition use, and related service account constraints. The displayed check names and messages come from the migration library.

### `convert`

```text
migrate-operators-v0-to-v1 convert <subscription-name> -n <namespace> [flags]
migrate-operators-v0-to-v1 convert --all [flags]
```

- **`-n`, `--namespace NAME`:** Required for one Subscription; rejected with `--all`.
- **`--all`:** Scan every namespace and select only `Eligible` Subscriptions.
- **`--dry-run`:** Check and preview without mutating cluster resources. A single ineligible target is refused with a nonzero exit.
- **`--continue-on-error`:** With `--all`, attempt later eligible operators after a failure. Without it, stop at the first failure. The command still exits nonzero if any attempt fails.
- **`--backup DIR`:** Write OLM resource backup YAML before deletion during conversion. Dry-run describes the plan without writing it.
- **`--delete-operatorgroup`:** Delete an OperatorGroup only when no Subscriptions remain; otherwise retain it.
- **`--ce-name NAME`:** Name the ClusterExtension for one target; defaults to the Subscription name and is rejected with `--all`.
- **`--install-namespace NAME`:** For one target, select the install namespace; defaults to the Subscription namespace. A nonempty value is rejected with `--all`.
- **`--system-managed-install-namespace`:** For one target, use the experimental controller mode: omit `ClusterExtension.spec.namespace` so a compatible OLMv1 controller chooses it from bundle metadata. The controller CRD capability is checked before mutation. Cannot be combined with `--install-namespace` or source namespace deletion.
- **`--acknowledge-namespace-delete`:** For one cross-namespace migration, permit deletion of the source namespace. Requires `--install-namespace` to differ from `-n/--namespace`.
- **`--acknowledge-watch-scope-change`:** Accept an operator changing from scoped watch to `AllNamespaces`.
- **`--acknowledge-operator-condition`:** Accept active OperatorCondition use.
- **`--acknowledge-olmv0-api-access`:** Accept OLMv0 API RBAC without an OLMv1 equivalent.
- **`--acknowledge-scoped-serviceaccount`:** Accept replacing scoped OperatorGroup ServiceAccount behavior with cluster-admin behavior.
- **`--acknowledge-not-steady-state`:** Accept an operator that is not at steady state.

Acknowledgment flags are explicit operator decisions. They do not override every hard incompatibility. A single `--dry-run` runs eligibility checks, gathers information, and then describes the planned resources and cleanup. Its text preview includes package/version/channel; a ClusterObjectSet or ClusterObjectDeployment and resource counts by kind; the proposed ClusterExtension namespace, package, version pinning, channel, and collision protection; OLMv0 cleanup actions; backup handling; and an explicit statement that no resources were modified. Resource kinds are printed alphabetically. The JSONL preview contains counts and cleanup actions rather than collected resource contents, which may contain Secrets.

The design targets close-to-zero downtime when the install namespace stays the same. Changing namespaces may cause downtime while resources move.

With `--all`, the command first prints the four-state scan summary, then processes eligible entries. Ineligible, conflict, and already migrated entries are not converted. If none are eligible, it reports that fact and exits 0. In text mode a successful single conversion ends with a completion banner; batch progress identifies each target. In JSONL mode each attempted target gets a terminal `result` record. Batch dry-run emits a preview and a result for each eligible target.

### `rollback`

```text
migrate-operators-v0-to-v1 rollback <clusterextension-name> [--acknowledge-installed]
migrate-operators-v0-to-v1 rollback --all [--acknowledge-installed]
```

`--all` selects ClusterExtensions carrying the migration-from-Subscription annotation. `--acknowledge-installed` is required when a target ClusterExtension is `Installed=True`; it is accepted for one target or a batch. Rollback requires a usable Subscription backup annotation and the original namespace to exist. It stops any associated ClusterObjectDeployment, removes the ClusterExtension and migration ClusterObjectSet revisions while orphaning workloads, releases their ownership metadata, then restores the Subscription from its backup. Text output announces the operation and reports success or failure. `--all` continues through targets after individual failures and exits nonzero if any failed; with no targets, it reports that none were found and exits 0.

### `cleanup`

```text
migrate-operators-v0-to-v1 cleanup <clusterextension-name>
migrate-operators-v0-to-v1 cleanup --all
```

Cleanup is for a `Conflict` state where both the Subscription and an annotated ClusterExtension exist after a partial migration. It removes the remaining OLMv0 artifacts and leaves the ClusterExtension intact. `--all` finds Conflict entries and matches them to annotated ClusterExtensions. Text output announces each operation and reports its outcome. Batch cleanup continues after individual failures and exits nonzero if any failed; with no matches, it reports that none were found and exits 0.

## JSON Lines contract for both executables

For a valid invocation, `--output=jsonl` emits one JSON object per line on stdout. Fields absent from a record are omitted. An argument validation failure can also print Cobra usage text before its error record. An unsupported `--output` value falls back to a plain-text error. The common envelope is:

| Field | Type | Meaning |
| --- | --- | --- |
| `type` | string | `progress`, `scan`, `check`, `dry-run`, `result`, or `error`. |
| `command` | string | Command emitting the event. |
| `target` | string | Subscription, CatalogSource, or ClusterExtension identity. |
| `step` | string | Progress phase. |
| `status` | string | Progress or result status. |
| `message` | string | Human-readable explanation. |
| `error` | string | Failure detail. |
| `data` | object or array | Command-specific payload. |

`command` is `migrate-catalogs` for catalog records or the operator subcommand name for operator records; a terminal `error` record omits it. `target` is `namespace/name` for a Subscription or CatalogSource, or a ClusterExtension name for rollback and cleanup. Aggregate records omit `target`. Progress `step` values include `scan`, `check`, `catalog`, `collect`, `backup`, `create`, `cleanup`, and `rollback`. Statuses include `started`, `waiting`, `completed`, `failed`, `warning`, and `note`, as applicable.

The operator record types use these payloads:

- **`scan`:** Array of scan results. Each has `namespace`, `name`, `status`, and optional `package`, `version`, `reason`, `checks`, `warnings`, `error`. The classification status is `Eligible`, `Ineligible`, `AlreadyMigrated`, or `Conflict`.
- **`check`:** One scan-result object with the same fields; each `checks` entry has `name`, `passed`, and `message`. A dry-run `convert` also emits this record before its preview.
- **`dry-run`:** Object with `package`, `version`, optional `channel`, optional `cluster_object_set` or `cluster_object_deployment`, `cluster_extension`, `install_namespace`, `system_namespace`, `manual_approval`, `kind_counts`, `cleanup_actions`, and optional `backup_directory`.
- **`result`:** No `data` payload. `status` is `completed` or `failed`; failed results include `error`. A no-target result has a `message` and completed status.
- **`error`:** No `data` payload. Contains the terminal `error` string.

For example, a successful single dry-run can produce the following *shape* (progress records and values vary by cluster):

```jsonl
{"type":"check","command":"convert","target":"operators/widget","data":{"namespace":"operators","name":"widget","status":"Eligible","reason":"passes all readiness, compatibility, and catalog-availability checks"}}
{"type":"dry-run","command":"convert","target":"operators/widget","data":{"package":"widget","version":"1.0.0","cluster_object_set":"widget-1","cluster_extension":"widget","install_namespace":"operators","system_namespace":"olmv1-system","manual_approval":false,"kind_counts":{"Deployment":1},"cleanup_actions":["Delete Subscription operators/widget with orphan propagation (operator workloads remain)."]}}
{"type":"result","command":"convert","target":"operators/widget","status":"completed"}
```

The example omits other fields and progress events for readability; it is not a literal transcript. A JSONL consumer should use `type`, `command`, `target`, and `status` to distinguish an intermediate event from an outcome. The number of progress records varies. A terminal `error` can follow per-target failed `result` records.

## Exit and review notes

- Exit 0 includes successful work, a completed check that reports `Ineligible` or `Conflict`, a scan with mixed classifications, a catalog run with only skipped sources, and an `--all` run with no matching targets.
- A failed cluster operation, invalid invocation, single ineligible `convert --dry-run`, or one or more catalog or batch operation failures exits nonzero. Batch conversion stops on its first failure unless `--continue-on-error`; rollback and cleanup batches continue and return a failure after processing.
- A progress output write failure exits nonzero in either output mode. The operator CLI must not emit a successful check, scan, or conversion result after that failure.
- `check --all` currently accepts `-n/--namespace` but scans every namespace; the namespace value is unused. `convert --all` rejects `-n/--namespace` and `--ce-name`.
- `--continue-on-error` is accepted on a single `convert` invocation but has no effect there.
- Text output includes ANSI escape sequences and Unicode symbols even when redirected. JSONL is the script-facing format; the text format is optimized for an operator reading progress and summaries.

## Design alignment and open gap

The command set, flags, non-interactive flow, and batch behavior reflect the [migration requirements](https://github.com/operator-framework/library-olm/blob/main/specs/20260821-migration-v0-to-v1/requirements.md). This document records the current CLI behavior where it differs from the intended design.

**Dry-run resource detail:** [Validation criterion V1.2](https://github.com/operator-framework/library-olm/blob/main/specs/20260821-migration-v0-to-v1/validation.md) requires `convert <operator> -n <namespace> --dry-run` to list every resource being migrated, grouped by kind, including CRDs. The current text and JSONL previews report counts by kind instead. This is an unmet design criterion, not a change to that requirement. Reviewers should decide whether to expand the preview or revise V1.2 before treating the CLI interface as finalized.
