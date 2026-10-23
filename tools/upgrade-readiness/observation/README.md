# Optional source 2.17.2 observation adapter

Inventory cannot establish which APIs clients exercise or whether SDK uploads omit
their namespace. This optional adapter observes existing backend request and
authorization boundaries on **exactly KFP 2.17.2**, commit
`2511cdbd74cd531c6633f2e094d235082a32917d`. It is a separate source patch, not part
of the ordinary read-only scanner and not a security audit-mode bypass.

The adapter makes no additional authentication, authorization, network, replay or
workload calls. Existing request results and enforcement remain unchanged,
including historical bypass/error behavior. Request hooks enqueue bounded evidence
without waiting for the writer. The recorder is disabled by default.

## Build and enable explicitly

Use a clean, separate checkout at the exact source commit. The helper validates
the revision, cleanliness and patch before modifying it; without `--apply` it only
checks. It refuses a different patch release or existing observer package.

```bash
python3 tools/upgrade-readiness/observation/apply_adapter.py \
  --source /path/to/clean-2.17.2
python3 tools/upgrade-readiness/observation/apply_adapter.py \
  --source /path/to/clean-2.17.2 --apply

cd /path/to/clean-2.17.2
go test -race ./backend/src/apiserver/readinessobservation
go test ./backend/src/apiserver/server ./backend/src/apiserver/resource \
  ./backend/src/apiserver \
  -run 'Test.*(Authoriz|Upload|CreateRun|CreateJob|Recurring|RunLog|Artifact)' -count=1
```

Build the patched API-server image using the source release's normal build
procedure, test it in an isolated installation, and roll out through the operator's
normal process. This tool does not build/deploy the image or change an installation.
The patch is deliberately small; the recorder is copied as a separate package.
Current master must not be substituted for the source checkout.

Set both administrator-controlled environment variables on the patched API server:

```text
KFP_READINESS_OBSERVATION_FILE=/private-observation/interval-001.json
KFP_READINESS_OBSERVATION_SECONDS=3600
```

The output's parent directory must already exist, be private (`0700`) and not be
a symlink. The absolute output path must be new: existing files/symlinks are
refused. Files and temporary snapshots use `0600`. Give each replica/restart a
unique path; do not share a filename. Observation duration is 1–604800 seconds.
Invalid settings disable observation without changing server enforcement. Removing
the settings and restarting disables the adapter; the original source image is
the rollback path.

The queue holds at most 128 events, the report retains at most 512 records and
1 MiB, and a background writer checkpoints once a second. Operation counters
continue when the record budget is exhausted; dropped/invalid/write evidence is
reported. Upload namespace parsing has its own bounded query work. Graceful
shutdown attempts a final checkpoint for at most two seconds. Abrupt termination
or persistent write failure may leave an active/stale checkpoint or no usable
file. Absence of a report is an observation failure, never a quiet successful run.

Reports are local operator-controlled files. They contain already-authenticated
caller identities and namespaces when available, so protect them accordingly.
They exclude request bodies, pipeline inputs, raw URLs, other query values,
credential headers, tokens and raw authorization errors. There is no external
export or new metrics endpoint. Aggregate counter keys are fixed operation enums;
identities, names, namespaces and URLs are never metric labels.

## Interpret and import evidence

The observer records new pipeline/version uploads, create run/recurring run,
retry, backend log/artifact request boundaries, and existing backend authorization
results. A request observation is not a successful response or completed workload.
Authorization observations are separate events: they must not be joined to request
events by timing or inferred to identify a particular upload/run.

Source 2.17 sometimes returns before authenticating, including shared reads and
namespace-less uploads. Those callers stay unknown. The source authentication API
does not expose complete caller groups. Frontend-specific log/TensorBoard/artifact
operations, actual artifact destinations, target-policy results, workload execution
and operations outside the interval remain unsupported. These gaps are explicit
in every report. No additional authentication is attempted to fill them.

Transfer a completed checkpoint through the operator's existing secure process,
then include it in a normal readiness assessment:

```bash
python3 tools/upgrade-readiness/readiness.py \
  --inventory cluster-inventory.json --system-namespace kubeflow \
  --namespace team-a --source-version 2.17.2 \
  --source-observation /secure/interval-001.json --format json > readiness.json
```

The importer validates the source revision, interval, bounded schema, counters
and coverage declarations. It adds interval/health/operation summaries and omitted
upload-namespace findings, withholding raw caller and namespace records. Active,
interrupted, dropped, stale, quiet or unsupported evidence never becomes a passing
target check. Even a completed healthy interval leaves the overall report
incomplete. Read multiple replica/interval reports separately and account for
infrequent schedules; one file cannot establish installation-wide traffic coverage.

CI checks exact-source patch application, recorder race/bound tests and the pinned
source request/authentication suites. This is separate from final-candidate
upgrade acceptance and is not evidence that an operator has deployed the adapter.
