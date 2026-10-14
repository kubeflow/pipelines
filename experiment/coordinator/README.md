# Hosted representative APF smoke and paired coordinator candidate

No live operations were run while preparing these files. The immediate entrypoint
is `hosted-smoke.yml`: publish it only on a separate upstream experiment branch
matching `codex/apf-representative-smoke-*`. It does not change PR test matrices.

Bundle under `experiment/`:

- `coordinator/`: these Python files (including coordinator.py) and this README.
- `provision/`: provision.py plus download_images.py from the provision candidate.
- `fixtures/`: manifest.json, three prepared YAML files, evaluate.py.
- `inputs/`: portable discovery-manifest.json from verified image evidence.

The workflow checks the experiment branch out as `harness`, independently checks
out source merge `6bea51e2003f9072a55068fb85e03cdb9391435d` as `source`, downloads exact
artifacts with same-repository GITHUB_TOKEN (actions:read), verifies them, deploys
an owned disposable Kind cluster, creates actual four-tenant Profiles, and
uploads fixtures. It then creates one flip_coin run with real tenant credentials,
waits at most 900 seconds plus bounded HTTP latency, and validates task outputs
using the strict fixture evaluator. CreateRun is never retried; accepted ID is
saved immediately. A successful run with incomplete task persistence gets up to
60 seconds to satisfy output assertions. Tokens stay in private runner state.

Evidence upload includes only the designated smoke evidence subtree: sanitized
provisioning-stage statuses, controlled fixture GetRun response, accepted run ID,
output assertions, and capture hashes. It excludes private state, rendered
manifests, token files, raw server logs, producer logs, and artifact archives.
The hosted runner is ephemeral; there is no automatic cluster deletion before
capture and no claim that an interrupted accepted run was reconciled.

Offline verification: `python3 -m unittest -v test_smoke.py` (4 tests), and
`python3 -m py_compile *.py`. No hosted smoke success is claimed yet.

## Longer paired test (not the smoke acceptance)

`coordinator.py validate --config CONFIG --out OUTPUT` validates frozen inputs;
`run` performs three counterbalanced20/-1 pairs with a common future timeline,
observer baseline hash handshake, guarded deployment switch before readiness
and new owned port-forward, peer-failure cancellation and per-arm capture.
Read its config validation before assembling a full paired run. It requires
independently verified prewarm evidence and workload/observer templates.

The paired evaluator is intentionally incomplete and refuses upstream-default
acceptance. Missing gates include actual KFP APF response-header attribution,
during-load denied-write/no-side-effect checks, all fixture artifact semantics,
recurring workflow lifecycle, direct etcd/restart health,60-second recovery,
server termination after client disconnect/no-deadline requests, minimum
Kubernetes and APF-disabled fallback coverage. Separate priority levels cannot
establish shared-priority fairness. Smoke success resolves infrastructure and
one real fixture only; it is not evidence to remove production client limits.
