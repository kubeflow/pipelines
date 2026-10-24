# Legacy v2 API coverage migration

Tracking: https://github.com/kubeflow/pipelines/issues/14950

The legacy workflow is retired after the recorded replacement execution and
trusted-publisher transition. This ledger maps active legacy assertions to replacement
specifications; code presence or successful compilation is not cluster parity.
All new specs inherit `ApiServerTests`, `PipelineCacheEnabled`, and
`LegacyMigration`. Unique scenario labels identify executed results in the
Ginkgo JSON report. Pending/disabled tests and explicit `Skip` bodies do not
count as replacements.

## API assertion ledger

Legacy source paths below describe the removed harness at commit
`9c7763b4c6cebfb89d74c3f9d983833ab8cc9c5f` and are relative to
`backend/test/v2`. `api/legacy_api_migration_test.go`
contains the `LegacyAPI*` scenarios.

| Legacy source and assertion group | Replacement | Execution status |
| --- | --- | --- |
| `integration/healthz_api_test.go`, `TestHealthzAPI`: successful nonnil health response and configured `PipelineStore` | `LegacyAPIHealth`; TLS client options match suite; expected store uses deployment flag | Verified in [CI evidence](#verified-ci-evidence) |
| `integration/pipeline_api_test.go`, `TestPipelineAPI`: YAML upload, name/display-name metadata and description | Existing `api/pipeline_upload_api_test.go` upload specs plus `LegacyAPIURLImports` distinct name/display/description assertions | Verified in [CI evidence](#verified-ci-evidence) |
| Same: duplicate pipeline rejection and ZIP upload/display name | `LegacyAPIUploads`, archive uploaded intact with tenant namespace, duplicate error asserted | Verified in [CI evidence](#verified-ci-evidence) |
| Same: URL YAML pipeline+version creation; inherited version name/display/description, parent ID, package URL | `LegacyAPIURLImports` | Verified in [CI evidence](#verified-ci-evidence) |
| Same: ZIP URL version creation; display/description, parent ID and package URL | `LegacyAPIURLImports` | Verified in [CI evidence](#verified-ci-evidence) |
| Same: exact list counts, total sizes, name ascending/descending and creation-order pagination, nonempty intermediate/empty final tokens | `LegacyAPIPagination`, owned prefix plus namespace; exact identity and display order across pages | Verified in [CI evidence](#verified-ci-evidence) |
| Same: unsupported sort key; Get display name/creation timestamp | `LegacyAPISortValidation`; `LegacyAPIPagination` | Verified in [CI evidence](#verified-ci-evidence) |
| `integration/pipeline_version_api_test.go`, `TestPipelineSpec`: implicit version display name from filename, duplicate upload rejection, ZIP version upload | `LegacyAPIUploads` | Verified in [CI evidence](#verified-ci-evidence) |
| Same: YAML and ZIP URL version creation and returned names | `LegacyAPIURLImports` | Verified in [CI evidence](#verified-ci-evidence) |
| Same: exact version list counts/totals, names, timestamps, all three pagination orders and token boundaries | `LegacyAPIPagination`, owned parent pipeline includes automatically created first version | Verified in [CI evidence](#verified-ci-evidence) |
| Same: unsupported version sort key | `LegacyAPISortValidation` | Verified in [CI evidence](#verified-ci-evidence) |
| Same and `TestV2Spec`: Get metadata and exact arguments/hello-world manifest equality | `LegacyAPIUploads` uploads raw YAML versions and compares both upload/Get manifests, with legacy store-assigned pipeline-name normalization | Verified in [CI evidence](#verified-ci-evidence) |
| `integration/run_api_test.go`, `TestRunAPIs`/`checkHelloWorldRunDetail`: version-backed Create/Get and identity, metadata, service account | Existing `api/pipeline_run_api_test.go` active creation specs and `LegacyAPITermination`; legacy timing-dependent fields are copied, not fixed expected values | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `checkArgParamsRunDetail`: embedded-manifest Create/Get, exact manifest, runtime parameters, description and experiment | `LegacyAPIEmbeddedRun` | Verified in [CI evidence](#verified-ci-evidence) |
| Same: paged run list count/total/order, experiment scope and created_at LESS_THAN excludes newer run | `LegacyAPIRunLists`, owned experiment and exact expected run IDs; existing active namespace list specs retain namespace boundary coverage | Verified in [CI evidence](#verified-ci-evidence) |
| Same: invalid sort field; archived run remains in List with ARCHIVED storage state | `LegacyAPISortValidation`; `LegacyAPIRunLists` | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `checkTerminatedRunDetail`: terminate succeeds, CANCELING/CANCELED state and stable full run metadata | `LegacyAPITermination`, bounded polling handles controller timing; explicit tenant SA in multiuser mode | Verified in [CI evidence](#verified-ci-evidence) |

## Cache and recurring-run ledger

| Legacy source / scenario | Replacement label(s) and preserved assertions | Execution status |
| --- | --- | --- |
| `integration/cache_test.go`, `TestCacheSingleRun` | `LegacyCacheSingleRun`: first run executes, repeat is CACHED, no executor pod | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `TestCacheRecurringRun` | `LegacyCacheRecurringRun`: first two chronological children succeed, first executes and second caches without executor pod | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `TestCacheSingleRunWithPVC_SameName_Caches` | `LegacyCachePVC`: both tasks execute initially, both cache with same claim, both execute with different claim | Verified in [CI evidence](#verified-ci-evidence) |
| `integration/recurring_run_api_test.go`, `TestRecurringRunApis` list/filter assertions | `LegacyRecurringList`: owned experiment, exact counts, name/created_at sorting, pagination, invalid sort and time filter | Verified in [CI evidence](#verified-ci-evidence) |
| Same, reference/inline execution and child assertions | `LegacyRecurringExecution`: persisted reference and inline parameters, scheduled children and parent association | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `TestRecurringRunApisUseLatest` | `LegacyRecurringLatest`: upload second version, unpinned recurring run resolves latest | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `TestRecurringRunApis_noCatchupOption` | Four `LegacyRecurringCatchupPeriodic{true,false}NoCatchup{true,false}` specs preserve periodic/cron and catchup combinations | Verified in [CI evidence](#verified-ci-evidence) |
| Same, `TestRecurringRunApis_DisabledCreatesScheduledWorkflow` and `TestRecurringRunApis_SwfNotFound` | `LegacyRecurringLifecycle`: disabled schedule/ScheduledWorkflow metadata, enable/disable roundtrip, delete after owned ScheduledWorkflow removed, Get returns 404 | Verified in [CI evidence](#verified-ci-evidence) |
| `integration/recurring_run_service_account_test.go`, `TestRecurringRunCustomServiceAccount` | Conditional `LegacyRecurringCustomAccountPinnedfalse` / `LegacyRecurringCustomAccountPinnedtrue`: tampered ScheduledWorkflow, persisted account/parameters/source, successful child | Both passed on MySQL/PostgreSQL cached multi-user shard B; required manifest rejects skips |

Cache and schedule fixtures use unique inputs/identifiers and owned-resource
cleanup. Namespace-wide deletion in the legacy harness is intentionally replaced
by experiment/schedule-owned cleanup to permit parallel execution.

## Nonexecuting and shared code

- `integration/experiment_api_test.go` had entry point `V2TestExperimentAPI`,
  which `go test` did not discover. It was removed with the retired harness
  and is not counted as active legacy workflow coverage.
  The newer `api/experiment_api_test.go` also uses only `PDescribe` containers.
  This is a pre-existing dormant coverage gap, not active replacement coverage
  and not a regression introduced by retiring the legacy workflow.
- Commented-out run-name assertions and disabled/new empty `PDescribe` or
  `PContext` bodies do not count toward either baseline or parity.
- New URL imports replace the newer suite's explicitly skipped placeholder URL
  cases. CI should use `KFP_TEST_FIXTURE_BASE_URL` to serve checked-out fixtures,
  as the retired legacy workflow did, so this tests imports without public-network
  dependence.
- Preserve `resources/` YAML/ZIP fixtures referenced by the replacement tests.
- Preserve `integration/webhook_test.go`: its three `WEBHOOK_INTEGRATION`-gated
  admission tests have a separate caller and are not obsolete API-harness code.
- `TestCacheRunSucceeded`, `TestCacheRecurringRunsSucceeded`, and
  `cache_diagnostics_unit_test.go` tested legacy support code, not deployed API
  behavior. They were removed together with the unused harness helpers after
  workflow retirement. Shared readiness, client configuration, TLS, upload
  helpers, and database initialization tests remain.

## Retirement evidence

Retirement requires all of the following, recorded on the tracking issue:

1. Old and new suites run against the same revision in separate clusters, for
   database/MySQL, database/MySQL+TLS, database/PostgreSQL, and Kubernetes/MySQL.
2. Ginkgo JSON results show each required migration scenario actually passed;
   pending, skipped, absent, or zero-selected scenarios fail the parity gate.
   Conditional custom-account specs retain an explicit opt-in lane/obligation.
3. Use existing CI logs and reports; investigate failures before deleting any
   legacy coverage. Additional retention infrastructure or repeated full
   comparisons are not prerequisites.
4. Review fixture and resource isolation, retained webhook callers, workflow
   inventory, aggregate check selection and CI contract tests.
5. Transition merge-policy requirements before deleting the legacy workflow:
   the expected-workflow selector reads the PR base revision. Test both sides
   of that transition and update the CI guide.

The implementation and compilation checks establish candidate replacements.
The API workflow continues enforcing the replacement manifests after retirement.

## Verified CI evidence

[Issue #14950 evidence](https://github.com/kubeflow/pipelines/issues/14950#issuecomment-6374576130)
records tested merge source `b68e3ece6e02da53cc221b3fb95a94bc84d245f7`:

- [Legacy run](https://github.com/kubeflow/pipelines/actions/runs/40020025186):
  all eight matrix jobs passed across the four legacy configurations.
- [API run](https://github.com/kubeflow/pipelines/actions/runs/40020025188):
  the 19-scenario manifest passed in the inspected MySQL, MySQL/TLS,
  PostgreSQL and Kubernetes-storage reports. Both custom-account scenarios
  passed exactly once in each cached MySQL/PostgreSQL multi-user shard B.
- The standalone proxy/Argo4.1.5 cache case failed during SDK installation
  because the GitHub clone received a proxy CONNECT 500. This failed attempt
  remains recorded; it does not establish cache parity. Passing non-proxy
  MySQL and other required configuration reports establish those assertions.

The required manifest checks continue on subsequent PRs. Existing job logs and
reports suffice for this audit; no new success-report retention is required.
