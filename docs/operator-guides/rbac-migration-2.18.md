# RBAC and client migration for 2.18

In multi-user deployments, review these permissions before upgrading to 2.18.
These checks apply to both V1 and V2 API callers. They do not currently have an audit/enforce
switch: update the caller or its namespace-scoped role instead of disabling
multi-user authorization. Existing experiment, run creation, and service-account
permissions are still required; the examples below are additions to custom roles,
not complete end-user roles.

| Operation | Required permission | Namespace checked | Migration |
| --- | --- | --- | --- |
| Upload a private pipeline | `create` on `pipelines.pipelines.kubeflow.org` | Explicit upload namespace | Pass `namespace=` to SDK uploads; choose **Private** in the pipeline upload UI. |
| Upload a shared pipeline | `create` on `pipelines.pipelines.kubeflow.org` | API server `POD_NAMESPACE` | Give a designated publisher this permission in the installation namespace. |
| Upload a pipeline version | `create` on `pipelines.pipelines.kubeflow.org` | Parent pipeline namespace; installation namespace for shared parents | Use the intended parent pipeline ID; changing client namespace does not relocate it. |
| Create a run or recurring run referencing a private pipeline/version | `get` on `pipelines.pipelines.kubeflow.org` | Parent pipeline namespace | Use an experiment in the same namespace as the private pipeline. |
| Read run logs | `readLog` on `runs.pipelines.kubeflow.org` | Run namespace | Add the custom verb to log-reader roles. |
| View an existing TensorBoard | `get` on `viewers.kubeflow.org` | Viewer namespace | Retain read access for viewers. |
| Create or delete TensorBoard | `create` or `delete` on `viewers.kubeflow.org` | Viewer namespace | Give these permissions only to users who should manage TensorBoard instances. |

## Make upload scope explicit

The SDK upload methods do **not** inherit `Client(namespace=...)` or
`set_user_namespace(...)`. An omitted upload namespace requests a shared pipeline,
which ordinary profile contributors may no longer be authorized to publish.

```python
import kfp

namespace = "team-a"
client = kfp.Client(namespace=namespace)  # Configure host/credentials as usual.
pipeline = client.upload_pipeline(
    pipeline_package_path="pipeline.yaml",
    pipeline_name="training",
    namespace=namespace,  # Required here even though Client has a default.
)
client.upload_pipeline_version(
    pipeline_package_path="pipeline.yaml",
    pipeline_version_name="revision-2",
    pipeline_id=pipeline.pipeline_id,  # Inherits team-a from the parent.
)
```

`upload_pipeline_from_pipeline_func(...)` likewise accepts an explicit
`namespace=`. In the pipeline upload UI, select the intended profile and choose
**Private**. Choosing **Shared** deliberately omits the namespace. Version uploads
retain the parent's scope. For REST multipart uploads, supply the `namespace`
query parameter on the pipeline upload endpoint; version uploads use `pipelineid`.

For intentional shared publication, leave the upload namespace unset and grant
publishing permission in the API server's actual `POD_NAMESPACE` (commonly
`kubeflow`). Supplying `namespace="kubeflow"` creates a namespace-scoped pipeline;
it does not request shared visibility. If `REQUIRE_NAMESPACE_FOR_PIPELINES` is
true, omitted namespaces are rejected before this authorization check.

## Keep private references in their owning namespace

A pipeline/version ID is not permission to use its contents. Creating a run or
recurring run from a stored private pipeline requires read access to the parent
pipeline and a destination in that pipeline's namespace. Read access to two
profiles does not, by itself, allow cross-profile references.

Select or create the experiment in the pipeline's namespace. If the destination
must differ, publish an authorized copy there, or deliberately publish a shared
pipeline through a designated publisher. Do not use `MULTIUSER_SHARED_READ` as a
migration shortcut: that existing setting deliberately permits broader reads and
cross-namespace references. It does not authorize log access or writes.

A version belongs to its parent pipeline; granting access to a different pipeline
or changing the supplied pipeline ID does not make the version accessible.
Inline pipeline specifications are not stored private references, but still need
the usual run/job and service-account permissions.

## Update custom roles selectively

The installed aggregated KFP view/edit roles already grant `runs` `readLog`.
The view role grants `viewers` `get`, while the edit role adds `create` and `delete`.
Check that the upgraded manifests and their aggregation labels are installed.
Custom roles copied from older releases need explicit updates.

Here is an example of **additional** permissions for a user who may upload/read
pipelines, read logs, and manage TensorBoard in `team-a`. Remove any rules or verbs
that the user does not need. Do not bind this role cluster-wide.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kfp-migration-permissions
  namespace: team-a
rules:
- apiGroups: [pipelines.kubeflow.org]
  resources: [pipelines]
  verbs: [create, get]
- apiGroups: [pipelines.kubeflow.org]
  resources: [runs]
  verbs: [readLog]
- apiGroups: [kubeflow.org]
  resources: [viewers]
  verbs: [get, create, delete]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kfp-migration-permissions
  namespace: team-a
subjects:
- kind: User
  name: user@example.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: kfp-migration-permissions
  apiGroup: rbac.authorization.k8s.io
```

Replace the subject with the authenticated user identity used by your installation,
or a `ServiceAccount` subject with its name and namespace for an SDK workload.
For a shared publisher, create a **separate** Role/RoleBinding in the installation
namespace granting only `create` on `pipelines.pipelines.kubeflow.org`; do not copy
the log or TensorBoard permissions there unnecessarily.

`readLog` is a custom **verb**, not a `runs/log` subresource. Granting `get` on runs,
`get` on pods, or `get` on `pods/log` does not satisfy KFP's check. TensorBoard uses
`viewers` in `kubeflow.org`, not `pipelines.kubeflow.org`. Upload checks have no
resource name, so adding `resourceNames` to the upload rule prevents it from
matching. The private-reference check uses the parent pipeline name, not its UUID.

## Verify before releasing the upgrade to users

An administrator with impersonation permission can check the effective grants:

```bash
kubectl auth can-i create pipelines.pipelines.kubeflow.org -n team-a --as=user@example.com
kubectl auth can-i create pipelines.pipelines.kubeflow.org -n kubeflow --as=user@example.com
kubectl auth can-i get pipelines.pipelines.kubeflow.org -n team-a --as=user@example.com
kubectl auth can-i readLog runs.pipelines.kubeflow.org -n team-a --as=user@example.com
kubectl auth can-i get viewers.kubeflow.org -n team-a --as=user@example.com
kubectl auth can-i create viewers.kubeflow.org -n team-a --as=user@example.com
kubectl auth can-i delete viewers.kubeflow.org -n team-a --as=user@example.com
```

Replace `kubeflow` with the actual installation namespace. Include the effective
impersonated groups when testing group-based bindings. A normal profile contributor
should get **no** for shared publication unless deliberately granted it. Kubernetes
may warn about KFP's custom resources/verbs; these are SubjectAccessReview
attributes and do not all represent Kubernetes objects.

Then test with real SDK/UI credentials:

1. Upload privately with an explicit namespace, upload a version, and create a run
   and recurring run in that namespace.
2. Confirm an ordinary user cannot publish shared pipelines, and the designated
   publisher can (unless namespace-required policy disables shared publication).
3. Confirm a private pipeline cannot be referenced from another profile under the
   default isolation policy, including when the user can read both profiles.
4. Read a running task's logs using a custom log-reader role; repeat without
   `readLog` and expect denial. Verify another profile's logs remain inaccessible.
5. Confirm a view-only user can inspect existing TensorBoard but cannot create or
   delete it; confirm the scoped manager can create and delete it.

The permission probes do not replace these API checks: private-reference namespace
confinement and resource ownership are enforced in addition to Kubernetes RBAC.
For upload failures, check the requested namespace, whether the parent is shared,
and the caller's `pipelines/create` grant. For log or TensorBoard denials, use the
resource, verb, and namespace in the response/server log to identify the missing
rule. Authorization service errors must be resolved rather than bypassed with
broader grants.
