# Frontend qualification reliability

Broader qualification remains observational until its reliability is established.
The offline reporter preserves first-attempt job failures, including failures
inside cancelled workflows. A successful rerun of the same run and source SHA is
recorded as recovery, **not** proof that the test is flaky: application,
infrastructure, and harness causes still require investigation.

## Collect evidence

Requires authenticated `gh` and Python 3. Run from the repository root. The
collector retrieves all attempts and all job pages for each selected workflow;
it does not substitute the latest result for first-attempt evidence.

```bash
python3 .github/resources/scripts/frontend_qualification_reliability.py \
  /tmp/kfp-qualification-evidence-20261008 \
  --collect --repository kubeflow/pipelines \
  --since 2026-10-01 --until 2026-10-08 \
  --workflow frontend-browser-qualification.yml \
  --workflow frontend-deployment-qualification.yml \
  --workflow frontend-performance-qualification.yml \
  > /tmp/kfp-qualification-report.json
```

Use a new empty evidence directory and a completed date window. Windows are
limited to 31 inclusive days, 200 runs per workflow, 20 attempts per run and
1,000 jobs per attempt. Split a busy window instead of treating truncated
results as complete. Each API call has a 120-second timeout. The collector fails
on active runs, duplicate or missing pages, identity mismatches, or a run that
changes while its attempts are collected. It preserves raw API responses and
marks `collection.json` incomplete on failure; the offline reporter rejects
that directory until collection is repeated into a new one. To regenerate a
report from retained completed evidence, omit `--collect`, `--repository`,
`--since`, and `--until`. Legacy manually captured evidence without a collection
manifest remains supported and missing attempts remain explicitly unknown.

Collection is automated when invoked; scheduling, notifications and ownership
are separate outstanding work. No GitHub issue, comment or check is modified.

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
