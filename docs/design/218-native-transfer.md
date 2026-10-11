# Importing 2.18 namespace archives into native storage

## User contract

A user exports metadata from the 2.18 transfer API, uploads the unchanged archive to a native-storage installation, validates, and explicitly imports. The source installation remains usable throughout the overlap. No source storage-generation upgrade, database clone, MLMD service connection, or artifact-file transfer is required.

This is one-way compatibility for `kfp-namespace-transfer-mlmd-2.18/v2`. Native exports remain native; reverse conversion is not supported. Existing namespace authorization, archive limits, conflict detection, disabled schedules and imported-history execution/cache isolation remain mandatory.

## Architecture

The native import boundary dispatches by archive format. Native archives retain strict decoding and schema matching. A legacy adapter decodes frozen 2.18 wire records, verifies the source digest, validates ownership and graph integrity, and translates supported records into a native namespace bundle. Only after conversion does it assign the destination native schema signature. The source schema fingerprint is provenance, not evidence that source and destination database schemas match; compatibility is determined by the explicitly supported wire format and record fields.

Conversion is deterministic and performs no writes. The existing native transfer planner maps identities into the destination and records receipts. Catalog and schedule preview runs before publication, followed by the existing staged-resource and transactional SQL import path. Validation and apply use the same conversion. Supported 2.18 imports produce no source-version or conversion warning. Format detection is internal; users follow the same export and import flow. Operational warnings still describe disabled schedules and external artifact access.

The adapter does not access source storage or execute source manifests. It preserves artifact URIs, leaving destination storage credentials and access configuration to the installation.

## Mapping and invariants

- Experiments, catalog definitions, versions and tags retain their logical relationships. Source catalog pins must be representable by the destination catalog; reject incompatible pin semantics.
- Completed SQL runs and MLMD execution records become native run/task history. Legacy context and execution IDs are translated rather than treated as native identifiers.
- Artifact identity derives from the source installation and legacy artifact ID, never URI alone. Shared artifacts stay shared across runs and repeat imports.
- MLMD events become native task/artifact input/output links with their keys and supported iteration identity preserved. Parent and cached-execution references must resolve consistently.
- Context relationships become native run/task ownership. Task provenance retains direct context/type IDs and names; mutable context properties, timestamps and ancestor bookkeeping are not copied. Artifact attribution context lists are not part of intrinsic artifact identity, so new runs sharing an artifact do not change old import receipts. Raw execution/artifact/type metadata and event details remain preserved.
- Graph references must stay inside the authorized namespace and refer to archived owners. If a selected history window omits a required owning run, reject and ask for a wider export.
- Unknown fields, unsupported relationships and ambiguous ownership reject before writes; successful conversion must not silently drop core history or lineage.
- Source scheduler progress is not transferred. Destination schedules remain disabled with catch-up disabled. Floating schedules validate against the effective destination default, including a previously retained Kubernetes default.
- Imported history stays excluded from execution, retry and cache reuse. Existing destination data is never overwritten merely because IDs or artifact URIs match.

## Verification

Use a fixture emitted by the 2.18 exporter to verify the actual wire contract and digest. Exercise conversion of parameters, timestamps, metrics, nested tasks, cached runs, shared artifacts, event keys and catalog/schedule references. Negative cases cover unknown formats/fields, malformed or tampered archives, namespace mismatch, missing owners, invalid IDs and unsupported metadata semantics.

Run imports against a populated destination, then repeat the same archive and overlapping batches. Verify no duplicate records or changed native records, preserved artifact references and disabled schedules. Exercise dry-run without external writes, native-format regressions and MySQL/PostgreSQL integration where available. A deployed source/destination smoke test remains distinct from fixture-backed UI demonstrations.

## Runtime parameter representation

The archive storage models embed a V1 `PipelineSpec.Parameters` field that shadows V2 `RuntimeConfig.Parameters` during JSON encoding. Both exporters therefore use an explicit `runtime_parameters` object containing `runs` and `schedules` maps. Every archived run and schedule must have a key, including empty parameter strings. Values retain their original JSON text without floating-point reserialization and participate in repeat-import conflict digests.

The incomplete archive format existed only in unreleased code and is not a compatibility contract. Only the current native and 2.18 formats are accepted; unsupported formats reject before writes. No partial-history fallback or missing-parameter warning is needed.

For a previously imported Kubernetes pipeline whose explicit default was cleared, adding new versions and an unpinned schedule can make its future default depend on server-assigned creation timestamps. Validation rejects this combination and asks the user to pin the destination default before retrying. It does not guess which definition will execute.

Legacy task-history `PENDING` transitions have no native task-state enum. They
retain their timestamps and errors as native `RUNTIME_STATE_UNSPECIFIED`; the
original SQL task, including its exact `PENDING` history, remains in
`status_metadata.custom_properties.legacy_sql_task`. SQL-only current-task
conversion and terminal MLMD validation remain unchanged; unknown historical
states still fail validation. Canonical legacy root DAGs
with an empty task name become native `root` tasks only after run-context,
execution identity, and parent ownership validation. Their completion time comes
from the owning terminal SQL run because legacy MLMD root bookkeeping is not
updated when execution finishes; the original MLMD timestamps remain in provenance.

## Deployed archive qualification

`legacy-native-transfer.yml` runs two independent disposable Kind installations.
The source API server is built from the pinned release exporter prerequisite
[#14782](https://github.com/kubeflow/pipelines/pull/14782),
`jeffspahr/jeffspahr-pipelines@bccc58c38cd2ad1283b965b594bbe4cd1da37af4`,
over the published 2.17.2 MLMD runtime. That prerequisite normalizes only completed
root DAG bookkeeping in the exported archive; unfinished child executions remain
rejected. The source fix must merge into `release-2.18` before this importer is
released.
The destination uses the candidate's native images and a separate database. This
qualifies the 2.18 exporter/native importer contract; it is not a complete 2.18
release-distribution or in-place upgrade qualification. The paused upgrade lane
remains paused.

The source executes the existing two-step container fixture with a parameter and
an artifact produced by one task and consumed by another, creates a disabled
recurring run, and exports through the HTTP endpoint. Another source run must
succeed after export. The destination first creates existing data, checks that
preview changes no visible resources, applies the archive, and checks terminal
run/task history, parameters, catalog/version definitions, artifact URIs, disabled
schedules with catch-up disabled, and repeat-import idempotence. The jobs retain
the source/destination commits, archive hash and API evidence as artifacts.
Artifact bytes and credentials are not copied. This smoke does not qualify live
schedule handoff, cached/nested execution graphs, or multi-user authorization;
those contracts retain their focused integration and regression coverage.
