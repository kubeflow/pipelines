# Importing 2.18 namespace archives into native storage

## User contract

A user exports metadata from the released 2.18 transfer API, uploads the unchanged archive to a native-storage installation, validates, and explicitly imports. The source installation remains usable throughout the overlap. No source storage-generation upgrade, database clone, MLMD service connection, or artifact-file transfer is required. Full parameter fidelity requires the updated 2.18 transfer exporter described below.

This is one-way compatibility for `kfp-namespace-transfer-mlmd-2.18/v2`, with restricted reading of v1 history. Native exports remain native; reverse conversion is not supported. Existing namespace authorization, archive limits, conflict detection, disabled schedules and imported-history execution/cache isolation remain mandatory.

## Architecture

The native import boundary dispatches by archive format. Native archives retain strict decoding and schema matching. A legacy adapter decodes frozen 2.18 wire records, verifies the source digest, validates ownership and graph integrity, and translates supported records into a native namespace bundle. Only after conversion does it assign the destination native schema signature. The source schema fingerprint is provenance, not evidence that source and destination database schemas match; compatibility is determined by the explicitly supported wire format and record fields.

Conversion is deterministic and performs no writes. The existing native transfer planner maps identities into the destination and records receipts. Catalog and schedule preview runs before publication, followed by the existing staged-resource and transactional SQL import path. Validation and apply use the same conversion. Warnings describe legacy conversion and any preserved historical metadata whose native representation differs.

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

## Runtime parameter wire correction

The original archive embeds storage models whose V1 `PipelineSpec.Parameters` field shadows V2 `RuntimeConfig.Parameters` during JSON encoding. An importer cannot reconstruct omitted runtime overrides. Both exporters therefore write v2 archives with an explicit `runtime_parameters` object containing `runs` and `schedules` maps. Every archived run and schedule must have a key, including empty parameter strings. Values retain their original JSON text without floating-point reserialization. Parameter values also participate in repeat-import conflict digests.

V1 archives containing schedules are rejected with an instruction to update the source exporter and export again. V1 history-only imports warn that V2 runtime overrides are unavailable. This is an explicit limitation, not a promise of lossless v1 recovery. Namespace identity and all other validations still apply. V2 legacy archives are decoded with frozen 2.18 DTOs; the optional field is appended after the existing digest field so the v1 checksum serialization stays unchanged when it is absent.

For a previously imported Kubernetes pipeline whose explicit default was cleared, adding new versions and an unpinned schedule can make its future default depend on server-assigned creation timestamps. Validation rejects this combination and asks the user to pin the destination default before retrying. It does not guess which definition will execute.
