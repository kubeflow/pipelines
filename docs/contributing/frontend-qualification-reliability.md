# Frontend qualification reliability

Broader qualification remains observational until its reliability is established.
The offline reporter preserves first-attempt job failures, including failures
inside cancelled workflows. A successful rerun of the same run and source SHA is
recorded as recovery, **not** proof that the test is flaky: application,
infrastructure, and harness causes still require investigation.

## Collect evidence

Requires authenticated `gh`, `jq`, and Python 3. Run from the repository root.
Collect a bounded date window, then every attempt's paginated job response. Use a
completed window; repeat collection if a selected run is still executing. Do not
use `gh run view`'s latest result as first-attempt evidence.

```bash
mkdir -p /tmp/kfp-qualification-evidence
repo=kubeflow/pipelines
evidence=/tmp/kfp-qualification-evidence
gh api --method GET --paginate --slurp "repos/$repo/actions/runs" \
  -f created='2026-10-01..2026-10-08' -f per_page=100 > "$evidence/runs.json"
jq -r '.[] | .workflow_runs[] | select(
  .path == ".github/workflows/frontend-browser-qualification.yml" or
  .path == ".github/workflows/frontend-deployment-qualification.yml" or
  .path == ".github/workflows/frontend-performance-qualification.yml") |
  [.id, .run_attempt] | @tsv' "$evidence/runs.json" |
while read -r run_id attempts; do
  for attempt in $(seq 1 "$attempts"); do
    gh api --paginate --slurp \
      "repos/$repo/actions/runs/$run_id/attempts/$attempt/jobs?per_page=100" \
      > "$evidence/$run_id-attempt-$attempt.json" || exit 1
  done
done
python3 .github/resources/scripts/frontend_qualification_reliability.py "$evidence" \
  --workflow frontend-browser-qualification.yml \
  --workflow frontend-deployment-qualification.yml \
  --workflow frontend-performance-qualification.yml > "$evidence/report.json"
```

Change the window to the evaluation period. GitHub limits filtered run searches
to 1,000 results; split busy periods into smaller windows rather than interpreting
a truncated sample as complete. Preserve raw responses beside the report.

## Interpret the report

`lanes` groups exact workflow filenames and job names; job names carry browser,
OS, and device identity. Only first-attempt `success` and explicit failures count
as observed pass/fail outcomes. Skipped, cancelled, pending, and missing results
remain `unknown`. `evidence` retains source SHA, run/job links, attempts, and
successful reruns. `incomplete_attempts` and `missing_workflows` expose gaps;
absence of evidence never establishes reliability. No failure-rate target or
promotion decision is inferred from this sample, and the reporter does not retry
jobs or alter checks.

[Issue #14754](https://github.com/kubeflow/pipelines/issues/14754) tracks the
remaining work: scheduled evidence collection and failure notifications with a
named owner, reproduction and classification of intermittent failures,
missing/stalled-run detection, and an agreed clean-run sample before promoting
qualification to required gates. This reporter alone does not provide monitoring.
