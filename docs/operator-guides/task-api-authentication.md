# Task API authentication and upgrades

In multi-user deployments, task creation requires an authenticated caller with
`createTask` on `runs` in `pipelines.kubeflow.org` in the stored run namespace.
Task authorization and authoritative run-namespace binding must ship together;
authorization alone can leave task storage using a different experiment namespace.
Single-user operation is unchanged.

The shipped pipeline-runner and edit roles include the new permission. Custom
Kubernetes roles must grant this exact rule in the owning run's namespace:

```yaml
apiGroups: [pipelines.kubeflow.org]
resources: [runs]
verbs: [createTask]
```

Check effective permissions, not the service-account name. A custom service
account bound to an updated standard role may already inherit this permission.
Retain existing service-account `use` permissions. The existing TokenReview
authentication path passes the username, but not authenticated groups, to
SubjectAccessReview. A group-only grant is therefore insufficient on this path;
use a direct binding to the workload service account where necessary.

Use driver and launcher images built with the credential-aware cache client.
Set `V2_DRIVER_IMAGE` and `V2_LAUNCHER_IMAGE` to those tested image versions or
digests on every API replica. An image string alone cannot establish that an
older or custom binary sends credentials. The compiler projects a token for the
actual workload service account with audience `TOKEN_REVIEW_AUDIENCE` plus
`/task-write` (normally `pipelines.kubeflow.org/task-write`) and sets
`KFP_TASK_API_TOKEN_PATH=/kfp-task-api/token`. This internal projection is separate
from the SDK's `KF_PIPELINES_SA_TOKEN_PATH`; existing SDK token paths, projections
and user `/var` mounts are preserved. The directory mount allows kubelet rotation;
the client rereads the file through the existing cached token source. An identical
internal path declaration is accepted. Conflicting reserved env/volume/mount
fields are rejected in custom runtime or pod patches. These checks protect wiring
and retry compatibility; they do not isolate credentials from component code.
The launcher shares the user container, so user code can read the projected token.
Only task-write authorization accepts this audience; normal API credentials remain
accepted for task writes. Explicit projection still occurs when automatic Kubernetes
token mounting is disabled (`automountServiceAccountToken: false`).

The existing trusted identity-header authenticator remains enabled. Restrict direct
API access to trusted proxies/workloads with the deployment's network and Istio
policies, and strip caller-supplied identity headers at the trust boundary. A raw
`kubeflow-userid` header is not evidence of identity on an untrusted direct path.
The task-only audience does not secure a deployment that trusts forged headers.

Existing V2 schedules and uploaded V2 IR need no recreation or SDK recompilation.
Fresh ticks compile through the current API server. Default/custom service
accounts, version selection, replay and scheduling policies are preserved.

The final API build enforces authorization immediately in multi-user mode. It has
no preparation-mode flag. A staged upgrade requires a separately delivered
credential-preparation API/compiler build or release, with the runtime credential
wiring and RBAC changes but without task-write enforcement. Deliver and test that
artifact with the task-only audience above, alongside credential-aware driver and
launcher images, before deploying
the enforcing API build. Existing image tags in manifests do not prove that the
images contain the new client; pin the actual tested versions or digests.

After every API replica runs the preparation build and selects the tested runtime
images, let whole workflows created before that boundary finish, including pending
steps and automatic Argo retries. Recurring V2 schedules can continue creating
prepared workflows. Preparation alone retains the old task authorization behavior
and is not the completed security fix. Then deploy authorization and namespace
binding together. Running old pods cannot gain the new client or missing token
mounts from an API upgrade. Cache-write failures remain fatal; deploying enforcement
before preparation can fail an old launcher after its user container succeeds.

Supported explicit V2 retry preserves saved images, completed nodes and outputs. Prepared
workflows receive fresh tokens in recreated pods. Missing credentials can be
added only to recognized current credential-aware runtimes. An incompatible
historical snapshot is rejected before a retry claim or pod deletion, with a log
warning and an actionable error. The failed run remains intact. Create a new run
from V2 IR if needed; this uses current templates but may repeat completed work.
Inline custom runtime templates and retained external templates are outside this
bounded preparation path and require an explicit update. Saved patches must keep
the credential declarations and launcher shared volume/mounts intact.
Old snapshots do not self-upgrade, and no cutoff date is imposed. Precompiled
Argo/custom legacy runtime templates need their owner's explicit update.

Verify actual images, projected-token rotation, cache misses/hits, PVC tasks,
existing/new schedules and custom service accounts in an isolated environment
before enforcement. Offline tests do not establish cluster rollout compatibility.

## Required shipping release note

The release that first ships task-write enforcement must document the new
`createTask` permission on `runs` in `pipelines.kubeflow.org`, the owning-namespace
scope, and the custom-role/direct-binding requirements above. It must also name
the available preparation artifact and tested runtime images, explain immediate
enforcement and the whole-workflow drain boundary, and describe the unsupported
historical-retry limitation and explicit new-run alternative. Do not advertise a
seamless staged upgrade until those artifacts and deployment checks are available.

## Authorization load

Task-only credentials require one TokenReview and one SubjectAccessReview per
write. Ordinary API credentials can require two TokenReviews because the task-only
check precedes the existing trusted-header/ordinary-token authenticators. That
order, audience validation and denial behavior are unchanged.

TokenReview and SubjectAccessReview have separate client token buckets. Each uses
these settings, resolved independently per field:

| Setting | Environment variable | Precedence | Default |
| --- | --- | --- | --- |
| `AuthClientQPS` | `AUTHCLIENTQPS` | Auth setting, explicit `ClientQPS`, default | 40 |
| `AuthClientBurst` | `AUTHCLIENTBURST` | Auth setting, explicit `ClientBurst`, default | 80 |

These are API-server configuration keys; environment variables follow the existing
Viper convention. General workflow/Kubernetes clients retain their existing
`ClientQPS`/`ClientBurst` defaults of 5/10. Explicit positive legacy settings remain
effective for reviews unless the corresponding auth setting overrides them. For
example, an explicit `ClientQPS=5` still limits reviews to 5 QPS; set
`AuthClientQPS` explicitly to change only the authorization budget.

The bounded 40-QPS default budgets two TokenReviews for the measured workload of
20 ordinary-token writes per second; burst 80 permits two seconds of review tokens.
It is a starting point, not a production capacity guarantee. Each API replica has
its own buckets, shared by all of that replica's authenticated API operations.
At the defaults, the two review clients together can issue up to 80 review requests
per second per replica after burst tokens are exhausted. Size total replica count,
other API traffic and Kubernetes API capacity with the cluster administrator.

QPS must be positive, finite and representable as a positive float32; burst must be
a positive integer. Invalid authorization settings fail startup rather than disable
rate limiting. This also applies to inherited legacy values. If unrelated clients
use a nonpositive legacy rate, provide explicit positive auth settings. Single-user
and webhook-only operation do not initialize these review clients.

Rehearse representative parallel/cache-heavy pipelines and user traffic, including
ordinary credentials. Measure client wait time separately from Kubernetes HTTP
latency/errors and task persistence. No authentication or authorization decision
cache is introduced, and each request still rechecks identity and permission.

The shipping release notes must include these review-client defaults and override
precedence alongside the credential/RBAC/upgrade requirements above. Preparation
credentials and their audiences are unchanged by this capacity adjustment; the
preparation task endpoint still performs no authorization reviews. Image and
whole-workflow upgrade qualification remain required.
