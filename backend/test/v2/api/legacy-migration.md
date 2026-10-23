# Legacy v2 API coverage migration

Tracking: https://github.com/kubeflow/pipelines/issues/14950

The legacy workflow remains a retirement gate until replacement execution has
been demonstrated. This ledger maps active legacy assertions to replacement
specifications; code presence or successful compilation is not cluster parity.
All new specs inherit `ApiServerTests`, `PipelineCacheEnabled`, and
`LegacyMigration`. Unique scenario labels identify executed results in the
Ginkgo JSON report. Pending/disabled tests and explicit `Skip` bodies do not
count as replacements.

## API assertion ledger

Paths below are relative to `backend/test/v2`. `api/legacy_api_migration_test.go`
contains the `LegacyAPI*` scenarios.

| Legacy source and assertion group | Replacement | Execution status |
| --- | --- | --- |
| `integration/healthz_api_test.go`, `TestHealthzAPI`: successful nonnil health response and configured `PipelineStore` | `LegacyAPIHealth`; TLS client options match suite; expected store uses deployment flag | Cluster proof required |
| `integration/pipeline_api_test.go`, `TestPipelineAPI`: YAML upload, name/display-name metadata and description | Existing `api/pipeline_upload_api_test.go` upload specs plus `LegacyAPIURLImports` distinct name/display/description assertions | Cluster proof required |
| Same: duplicate pipeline rejection and ZIP upload/display name | `LegacyAPIUploads`, archive uploaded intact with tenant namespace, duplicate error asserted | Cluster proof required |
| Same: URL YAML pipeline+version creation; inherited version name/display/description, parent ID, package URL | `LegacyAPIURLImports` | Cluster proof required |
| Same: ZIP URL version creation; display/description, parent ID and package URL | `LegacyAPIURLImports` | Cluster proof required |
| Same: exact list counts, total sizes, name ascending/descending and creation-order pagination, nonempty intermediate/empty final tokens | `LegacyAPIPagination`, owned prefix plus namespace; exact identity and display order across pages | Cluster proof required |
| Same: unsupported sort key; Get display name/creation timestamp | `LegacyAPISortValidation`; `LegacyAPIPagination` | Cluster proof required |
| `integration/pipeline_version_api_test.go`, `TestPipelineSpec`: implicit version display name from filename, duplicate upload rejection, ZIP version upload | `LegacyAPIUploads` | Cluster proof required |
| Same: YAML and ZIP URL version creation and returned names | `LegacyAPIURLImports` | Cluster proof required |
| Same: exact version list counts/totals, names, timestamps, all three pagination orders and token boundaries | `LegacyAPIPagination`, owned parent pipeline includes automatically created first version | Cluster proof required |
| Same: unsupported version sort key | `LegacyAPISortValidation` | Cluster proof required |
| Same and `TestV2Spec`: Get metadata and exact arguments/hello-world manifest equality | `LegacyAPIUploads` uploads raw YAML versions and compares both upload/Get manifests, with legacy store-assigned pipeline-name normalization | Cluster proof required |
| `integration/run_api_test.go`, `TestRunAPIs`/`checkHelloWorldRunDetail`: version-backed Create/Get and identity, metadata, service account | Existing `api/pipeline_run_api_test.go` active creation specs and `LegacyAPITermination`; legacy timing-dependent fields are copied, not fixed expected values | Cluster proof required |
| Same, `checkArgParamsRunDetail`: embedded-manifest Create/Get, exact manifest, runtime parameters, description and experiment | `LegacyAPIEmbeddedRun` | Cluster proof required |
| Same: paged run list count/total/order, experiment scope and created_at LESS_THAN excludes newer run | `LegacyAPIRunLists`, owned experiment and exact expected run IDs; existing active namespace list specs retain namespace boundary coverage | Cluster proof required |
| Same: invalid sort field; archived run remains in List with ARCHIVED storage state | `LegacyAPISortValidation`; `LegacyAPIRunLists` | Cluster proof required |
| Same, `checkTerminatedRunDetail`: terminate succeeds, CANCELING/CANCELED state and stable full run metadata | `LegacyAPITermination`, bounded polling handles controller timing; explicit tenant SA in multiuser mode | Cluster proof required |

## Cache and recurring-run ledger

| Legacy source / scenario | Replacement label(s) and preserved assertions | Execution status |
| --- | --- | --- |
| `integration/cache_test.go`, `TestCacheSingleRun` | `LegacyCacheSingleRun`: first run executes, repeat is CACHED, no executor pod | Cluster proof required |
| Same, `TestCacheRecurringRun` | `LegacyCacheRecurringRun`: first two chronological children succeed, first executes and second caches without executor pod | Cluster proof required |
| Same, `TestCacheSingleRunWithPVC_SameName_Caches` | `LegacyCachePVC`: both tasks execute initially, both cache with same claim, both execute with different claim | Cluster proof required |
| `integration/recurring_run_api_test.go`, `TestRecurringRunApis` list/filter assertions | `LegacyRecurringList`: owned experiment, exact counts, name/created_at sorting, pagination, invalid sort and time filter | Cluster proof required |
| Same, reference/inline execution and child assertions | `LegacyRecurringExecution`: persisted reference and inline parameters, scheduled children and parent association | Cluster proof required |
| Same, `TestRecurringRunApisUseLatest` | `LegacyRecurringLatest`: upload second version, unpinned recurring run resolves latest | Cluster proof required |
| Same, `TestRecurringRunApis_noCatchupOption` | Four `LegacyRecurringCatchupPeriodic{true,false}NoCatchup{true,false}` specs preserve periodic/cron and catchup combinations | Cluster proof required |
| Same, `TestRecurringRunApis_DisabledCreatesScheduledWorkflow` and `TestRecurringRunApis_SwfNotFound` | `LegacyRecurringLifecycle`: disabled schedule/ScheduledWorkflow metadata, enable/disable roundtrip, delete after owned ScheduledWorkflow removed, Get returns 404 | Cluster proof required |
| `integration/recurring_run_service_account_test.go`, `TestRecurringRunCustomServiceAccount` | Conditional `LegacyRecurringCustomAccountPinnedfalse` / `LegacyRecurringCustomAccountPinnedtrue`: tampered ScheduledWorkflow, persisted account/parameters/source, successful child | Requires the same opt-in service-account/multiuser environment as legacy; excluded from unconditional execution count |

Cache and schedule fixtures use unique inputs/identifiers and owned-resource
cleanup. Namespace-wide deletion in the legacy harness is intentionally replaced
by experiment/schedule-owned cleanup to permit parallel execution.

## Nonexecuting and shared code

- `integration/experiment_api_test.go` has entry point `V2TestExperimentAPI`,
  which `go test` does not discover. It is not active legacy workflow coverage.
  The newer `api/experiment_api_test.go` also uses only `PDescribe` containers.
  This is a pre-existing dormant coverage gap, not active replacement coverage
  and not a regression introduced by retiring the legacy workflow.
- Commented-out run-name assertions and disabled/new empty `PDescribe` or
  `PContext` bodies do not count toward either baseline or parity.
- New URL imports replace the newer suite's explicitly skipped placeholder URL
  cases. CI should use `KFP_TEST_FIXTURE_BASE_URL` to serve checked-out fixtures,
  as the legacy workflow does, so this tests imports without public-network
  dependence.
- Preserve `resources/` YAML/ZIP fixtures referenced by the replacement tests.
- Preserve `integration/webhook_test.go`: its three `WEBHOOK_INTEGRATION`-gated
  admission tests have a separate caller and are not obsolete API-harness code.
- `TestCacheRunSucceeded`, `TestCacheRecurringRunsSucceeded`, and
  `cache_diagnostics_unit_test.go` test legacy support code, not deployed API
  behavior. Remove only together with the now-unused helpers after retirement;
  keep them while the old harness runs.

## Retirement evidence

Retirement requires all of the following, recorded on the tracking issue:

1. Old and new suites run against the same revision in separate clusters, for
   database/MySQL, database/MySQL+TLS, database/PostgreSQL, and Kubernetes/MySQL.
2. Ginkgo JSON results show each required migration scenario actually passed;
   pending, skipped, absent, or zero-selected scenarios fail the parity gate.
   Conditional custom-account specs retain an explicit opt-in lane/obligation.
3. Two consecutive complete successful comparisons, with timing and report
   artifacts. Investigate failures before deleting any legacy coverage.
4. Review fixture and resource isolation, retained webhook callers, workflow
   inventory, aggregate check selection and CI contract tests.
5. Transition merge-policy requirements before deleting the legacy workflow:
   the expected-workflow selector reads the PR base revision. Test both sides
   of that transition and update the CI guide.

The implementation and compilation checks establish candidate replacements.
Until these execution gates pass, leave the legacy workflow enabled.
