# Disposable representative-overload provisioning candidate

**Prepared only. No cluster, deployment, Docker image import, upload, or GitHub
mutation has been executed. This is an integration candidate, not verified
infrastructure and not evidence for an upstream default change.**

`provision.py` uses source commit
`6bea51e2003f9072a55068fb85e03cdb9391435d`. It has no source-checkout writes;
`create` obtains a private `git archive` snapshot. It deliberately does not run
`deploy-kfp.sh`: that script's successful `collect_artifacts` and error diagnostic
paths capture raw pod logs. Instead, the candidate mirrors its pinned multi-user
prerequisites and renders its database-backed multi-user overlay, with explicit
local image identities and initial QPS configuration. It never sets user tokens
on command-line arguments or prints child output, HTTP bodies, or raw errors.

## Inputs that must be supplied and independently verified

Required tools: Python3.11+ and PyYAML (`/private/tmp/kfp-api-shards-venv/bin/python`
can provide it), Git, tar, Docker, Kind and kubectl. Kubernetes node image must be
an immutable `kindest/node@sha256:...` for v1.36.1 initially. Existing standalone
`kind-kfp-apf-experiment` is NOT adopted; a fresh
`kfp-apf-representative-<suffix>` cluster is required.

All seven KFP images must come from the same source SHA: `apiserver`,
`persistenceagent`, `scheduledworkflow`, `frontend`, `viewer-crd-controller`,
`driver`, `launcher`. Existing exact build recipes are the image matrix in
`.github/workflows/image-builds.yml`: backend/Dockerfile,
Dockerfile.persistenceagent, Dockerfile.scheduledworkflow, frontend/Dockerfile,
backend/Dockerfile.viewercontroller, Dockerfile.driver and Dockerfile.launcher;
all build contexts are repository root. Existing CI images are docker-save tar
artifacts, not necessarily registry-published digests. Reuse successful current
head artifacts only after existing image-artifact provenance checks bind them to
the source/run/artifact IDs. `wait-for-image-artifacts.sh` and
`load-ready-image-artifacts.sh` implement the CI path; this candidate intentionally
does not discover/download artifacts or assume GH_TOKEN/run metadata.

Provide archive SHA256, Docker config ID, original artifact ZIP, its exact tar
member name, and saved authenticated GitHub artifact/run API responses for each
image. Validation binds the GitHub ZIP digest to the ZIP member tar digest and
the tar config bytes to the expected image ID **before Docker load**. An expected
image already in Docker can no longer mask a wrong archive. The producer must
be a successful reusable image-build job in kubeflow/pipelines at the frozen merge SHA.
Missing GitHub artifact digest fails closed.

No OCI revision label is required: the real CI recipes do not emit one. The
verifier instead requires authenticated run/job/artifact metadata, successful
fresh-build and upload steps, the producer log identifying the reusable image
workflow and exact checked-out merge SHA, and the GitHub commit parents binding
that merge to PR head `f6c500fa138bfc506856006ded58a211254531d5`. The workflow run's
head is the PR head; the checked-out build tree is merge
`6bea51e2003f9072a55068fb85e03cdb9391435d`, tree
`2709cff301082e001a7201b4d15b57f277d0ec22`. Conflating these SHAs is incorrect.
A completed successful producer job is sufficient even while unrelated consumer
jobs in its workflow remain running.

For this comparison all seven API and seven E2E producer jobs freshly built;
retained-image validation was skipped. The selected API image archives therefore
need no cache-origin inference. A future cache consumer must be traced back to
its original fresh producer and matching build tree; this bounded verifier
refuses to pretend a cache consumer is that producer. Saved records must come
from authenticated GitHub API, not caller-written attestations; offline checks
cannot authenticate fabricated JSON.

The provisioner retags validated images to
`kfp-apf-local/<component>:<source-sha>` and records identities. It also sets
FRONTEND_IMAGE/FRONTEND_TAG on the KFP profile hook so generated tenant frontends
use the same loaded frontend image (the hook's source uses IfNotPresent).

Fixtures must be precompiled exact-source versions of flip_coin,
collected_parameters and if_else_with_oneof_artifacts with SHA256/source binding.
The checked-in `test_data/sdk_compiled_pipelines/valid/critical/*.yaml` are useful
reference IR but embed `kfp==2.14.3` plus `python:3.11`; blindly calling those
exact-checkout SDK coverage would be incorrect. Regenerate/source-pin and
prewarm using the established test fixture path before measuring. Do not rewrite
fixture coverage while comparing20 with-1. Preserve fixture bytes/hash, compiler
and SDK identities, component image digests, and runtime parameter requirements.
This script does not compile SDK fixtures or validate their output semantics.

Example config shape (replace every placeholder):

```json
{
  "source_sha": "6bea51e2003f9072a55068fb85e03cdb9391435d",
  "source_worktree": "/Users/jspahr/.codex/worktrees/api-multiuser-shards/pipelines",
  "cluster": "kfp-apf-representative-r1",
  "node_image": "kindest/node@sha256:REPLACE64HEX",
  "qps": 20,
  "port": 18888,
  "images": {
    "apiserver": {
      "source_sha": "6bea51e2003f9072a55068fb85e03cdb9391435d",
      "archive": "/private/tmp/verified-images/apiserver.tar",
      "archive_sha256": "REPLACE64HEX",
      "image_id": "sha256:REPLACE64HEX",
      "artifact_zip": "/private/tmp/verified-images/apiserver.zip",
      "archive_member": "REPLACE_WITH_EXACT_ZIP_MEMBER",
      "artifact_metadata": "/private/tmp/verified-images/apiserver-artifact.json",
      "producer_run_metadata": "/private/tmp/verified-images/producer-run.json"
    }
  },
  "fixtures": {
    "flip_coin": {
      "source_sha": "6bea51e2003f9072a55068fb85e03cdb9391435d",
      "path": "/private/tmp/verified-fixtures/flip_coin.yaml",
      "sha256": "REPLACE64HEX"
    }
  }
}
```

The config must include all seven image keys and all three fixture keys, not only
the illustrated entries. Burst stays40 for both arms. No image build or pull is
automatically authorized by running `validate`.

## Candidate commands, NOT executed

```sh
PY=/private/tmp/kfp-api-shards-venv/bin/python
PROVISION=/private/tmp/kfp-apf-overload-provision/provision.py
CONFIG=/private/tmp/apf-provision-input.json
STATE=/private/tmp/apf-representative-r1
$PY "$PROVISION" validate --config "$CONFIG" --state "$STATE"
$PY "$PROVISION" create   --config "$CONFIG" --state "$STATE"
$PY "$PROVISION" deploy   --config "$CONFIG" --state "$STATE"
$PY "$PROVISION" tenants  --config "$CONFIG" --state "$STATE"
$PY "$PROVISION" upload   --config "$CONFIG" --state "$STATE"
```

- `create` refuses any existing cluster or state directory. Its dedicated
  kubeconfig never changes the user's current context; later operations require
  both exact context and recorded kube-system namespace UID. Partial creation
  without a saved UID requires explicit manual ownership reconciliation.
- `deploy` imports verified images, installs cluster CRDs, Istio and actual
  Profile Controller using Kubeflow-manifests SHA
  `88716b3f7f62b12f98d82bcfc59635bb07e7845c`, applies the KFP multiuser/default
  rendered overlay, and waits boundedly for API/persistence/scheduledworkflow
  deployments plus profile-controller. All readiness calls are bounded, as are
  individual subprocesses. No remote manifest HEAD references are used. It saves
  APF policies without reading Secrets or pod logs. Third-party image tags from
  pinned manifests still need digest snapshots/prewarm for experiment repeatability.
- `tenants` creates four actual Profiles (one noisy/three quiet), waits up to300s
  each for Profile-created default-editor and mlpipeline-minio-artifact, applies
  the repository kubeflow-edit aggregate ClusterRole, and namespace-specific
  network access. It does NOT grant tenants cluster-admin or share permission
  to other tenants. Uploads are namespace-scoped, so the existing CI shared-pipeline
  RoleBinding in kubeflow is intentionally unnecessary. IAM dependency failure
  is an explicit stop, not an assumed artifact-secret success.
- `upload` starts/owns a localhost-only port-forward, obtains projected tenant
  default-editor tokens with audience `pipelines.kubeflow.org`/1h expiration,
  saves tokens0600, and uses real authenticated KFP HTTP APIs:
  POST `/apis/v2beta1/experiments`, multipart POST
  `/apis/v2beta1/pipelines/upload?namespace=...&name=...`, then multipart POST
  `/apis/v2beta1/pipelines/upload_version?pipelineid=...&name=exact-source`.
  It saves returned experiment/pipeline/version IDs after each response and
  never retries a failed or ambiguous write. `upload-started.json` prevents
  blind rerun. Restore/reconcile manually if a write response is lost.
- `upload` terminates its owned port-forward. Restart a guarded foreground
  port-forward for the workload runner; skeleton endpoint is not proof it remains
  available. Tokens must be renewed before they expire between paired tests.
  `workload-config-skeleton.json` deliberately has empty phases; choose/calibrate
  and freeze workload settings from the experiment plan first.

Cleanup is never automatic on failure. Capture permitted evidence before making
an explicit `evidence-captured.json` manifest in STATE (include capture paths and
hashes), then run the `cleanup` action. It rechecks owned cluster UID and deletes
only that newly created named Kind cluster, removing private tenant token files.
It does not delete a namespace or cluster based on a name-only prefix and does
not clean up ambiguous KFP IDs in a reused cluster. No blanket Docker prune.

## Unresolved integration requirements / limitations

- No live render, Profile reconciliation, image import, KFP upload, workload, or
  cleanup has been exercised. Only Python syntax/offline unit checks were run.
- Immutable source image provenance and node image digest must still be resolved;
  availability/architecture of current GitHub image artifacts is unverified.
- Remote Istio/Profile Controller versions from pinned manifests may need host
  resource limits or image-prepull; don't replace with standalone assumptions.
- CI IAM conntrack workaround/readiness diagnostics are not reimplemented. Profile
  secret failure requires bounded sanitized diagnostics and may require invoking
  the existing IAM helper without its raw-log capture path.
- Broad readiness needs all backing database/SeaweedFS/Argo infrastructure;
  selected Deployment waits and first authenticated uploads are not a complete
  prewarm gate. Verify full readiness, sourceSDK bootstrap, workflow execution
  and output assertions before any measurement window.
- The script only creates a normal APF-enabled Kind cluster. It does not implement
  same-priority competitor, APF metric sampling, workload image prewarm,
  counterbalanced setting switches, no-deadline watchdog, scheduling fairness,
  APF-disabled fallback cluster, or threshold evaluation. Those remain separate
  required slices in the journal plan.
- Tenant Profiles span distinct namespaces but still share KFP API-server
  Kubernetes identity; this does not by itself validate tenant fairness.
- A successful provisioning command is not a passed overload decision gate.

## Hardening verification (offline)

All POST redirects (301/302/303/307/308) are refused; bearer credentials are never
forwarded through a redirect. No POST retries are introduced. All rendered
Deployment/StatefulSet/DaemonSet controllers must finish bounded rollout,
including MySQL, SeaweedFS, Argo, metacontroller and KFP profile hook. The external
Profile controller and Profiles CRD are also waited on. Before Profiles are
created, SeaweedFS IAM DNS:8111 must pass three consecutive socket probes from
the KFP profile-hook container within 120 seconds, matching the source deployment
success criterion. Unlike the repository helper, failure never collects raw logs.
This is a connectivity check, not proof of successful IAM account creation;
subsequent artifact-secret and default-editor creation remain required.

`python3 -m unittest discover -s /private/tmp/kfp-apf-overload-provision -p '*_test.py'`
passes 12 offline tests. Syntax compilation passes. Tests cover tar/config and
artifact ZIP binding, wrong producer head, exact source checkout evidence, redirect
refusal, generated frontend env preservation, all controller readiness calls,
IAM consecutive-success behavior, ownership and secret-output guards. No real
cluster, image loading or API upload has been tested.

Remaining live prerequisites: hosted amd64 execution and exact
SDK fixture generation/prewarming and output validation; pinned third-party
image/network availability; runtime APF classification evidence; workload and
observer timing integration. This candidate is still not live-ready until these
are resolved. Readiness permits up to 600 seconds per rendered controller; it is
bounded per resource, not one global deployment deadline.

## Actual discovered inputs (October 14 UTC)

`/private/tmp/apf-verified-images/image-inputs.json` contains seven exact image
IDs, ZIP/member/tar hashes and metadata/log paths from API run38489752122 attempt1.
The actual ZIPs were downloaded before execution moved to hosted amd64. They
occupy approximately458MiB; no tar extraction or Docker import was performed.
Both runs38489752122/38489752069 were audited via paginated job metadata.
`evidence-hashes.json` binds saved metadata/logs/ZIPs for copying and review.
Image verification can run directly against ZIP streams without extracting tar.
Deployment still requires extracting the already-verified tar member on the
hosted runner, adjusting input paths and obtaining fixture/node inputs. The local
host is aarch64 with limited free disk; do not deploy these amd64 images there.

The obsolete requirement for OCI revision labels was removed after examining
real successful fresh-build producer evidence. No labels or image bytes changed.

## Hosted downloader

```
python experiment/provision/download_images.py \
  --manifest experiment/inputs/discovery-manifest.json \
  --output "$RUNNER_TEMP/apf-images"
```

Use the same-repository `GH_TOKEN` environment with Actions read permission.
The destination must not exist; at least 3 GiB free is required for ZIP/tar
preparation (cluster/Docker capacity is an additional runner prerequisite).
Downloads are exact-ID, bounded, and not silently retried. The downloader fetches
original run-attempt/job/artifact metadata and producer logs, verifies source and
hash bindings, then extracts only the fixed verified tar member using streaming
copy. It produces `image-inputs.json` with absolute paths and
`evidence-hashes.json`. Logs are saved for provenance, not printed. No tokens are
passed in CLI arguments. Fifteen offline tests and syntax checks pass; this
script has not itself been executed against GitHub or Docker.

### Rerun-safe artifact correction

The first hosted smoke failed because API run38489752122 warm reruns replaced
normal-name attempt1 artifacts:11860864003 now returns404. This was stale
identity, not authorization failure. Replacement normal-name artifacts may be
cache consumers and must not be mislabeled fresh builds.

Use `/private/tmp/apf-retained-inputs/discovery-manifest.json` instead: it pins
seven retained `ci-image-artifact-v2-<component>-amd64-2709cff...` artifacts and
each actual fresh producer run/attempt/job (across same-merge workflows). The
verifier checks the retained key's full tree SHA against the authenticated merge
commit and still requires successful fresh build, checkout and artifact upload
log evidence. Both experiment arms use these exact images. They may differ in
bytes from independent API attempt1 builds; they are all actual builds of the
same pinned merge source. No claim of byte equality is made.

Downloader now supports per-image producer_run_id/producer_run_attempt and
producer_workflow_path. HTTP error status is shown without stderr bodies or
credentials. Sixteen offline tests pass, including retained exact-tree acceptance,
wrong-tree rejection, cache-consumer rejection and sanitized404 diagnostics.

### Deployment render repair verified

Hosted smoke38495217147 completed image verification and cluster creation, then
failed before creating KFP workloads. The candidate omitted the source helper's
`--load-restrictor LoadRestrictionsNone`; multiuser/default references sibling
patch `../pipeline-crd-rbac.yaml`, which kubectl Kustomize otherwise refuses.
The corrected exact-merge6bea snapshot renders109objects successfully (including
all10KFP Deployment/StatefulSet workloads). No cluster apply was performed locally.
Seventeen offline tests pass, including the render flag and diagnostics regression.
Command failures now print only executable, allowlisted operation, exit status
and allowlisted error category. Raw stderr, resource bodies and credentials are
still suppressed. This makes future smoke errors actionable without exposing
secrets. Source reference: `.github/resources/scripts/render-profiled-manifests.sh`.
