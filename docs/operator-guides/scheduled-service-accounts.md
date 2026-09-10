# Service accounts for recurring runs

A recurring run is a standing authorization to execute a stored pipeline with
its stored parameters, pipeline root, plugin inputs, and service account.
Creating it does not grant its creator new service-account permissions.

For a non-default service account, the API server requires its name in the
comma-separated `ALLOWEDSERVICEACCOUNTS` configuration. In multi-user mode it
also checks the caller's `use` permission on that exact `serviceaccounts` resource
in the execution namespace. The configured default pipeline runner remains
exempt from the additional service-account check.

## Multi-user execution

The ScheduledWorkflow controller submits each execution as its own identity.
Consequently, both the creating user and the controller need permission to use
the selected custom account. The controller does not impersonate the creator,
and supplying a recurring-run ID does not bypass the caller's authorization.

The multi-user manifests set `MULTIUSER=true` on the controller, enabling its
`--multiUser=true` mode. Every schedule, including a pinned or inline workflow,
then goes through the API. The API resolves execution inputs from the stored job,
checks its namespace and backing Kubernetes UID, and checks that it is enabled.
Changes to execution fields in the Kubernetes `ScheduledWorkflow` cannot change
which pipeline, parameters, account, or plugin inputs the API executes. Controller
reports update status only in multi-user mode. To change a recurring run's
specification, recreate it through the API; use the API to enable or disable it.

Single-user controller behavior remains unchanged. In a custom multi-user
installation, explicitly set `--multiUser=true`; authentication headers or bearer
tokens alone do not enable this execution mode.

## Grant access to an approved custom account

Do not grant the controller cluster-wide `use` access to all service accounts.
For each approved namespace/account pair, create a namespaced Role and bind it
to the authorized user and the actual controller service account. For example,
replace the names below with the installation's identities:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: use-approved-pipeline-runner
  namespace: team-a
rules:
  - apiGroups: [""]
    resources: [serviceaccounts]
    resourceNames: [approved-pipeline-runner]
    verbs: [use]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: approved-pipeline-runner-user
  namespace: team-a
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: use-approved-pipeline-runner
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: user@example.com
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: approved-pipeline-runner-controller
  namespace: team-a
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: use-approved-pipeline-runner
subjects:
  - kind: ServiceAccount
    name: ml-pipeline-scheduledworkflow
    namespace: kubeflow
```

Add `approved-pipeline-runner` to the API server's `ALLOWEDSERVICEACCOUNTS` value
and retain any other deliberately approved names. This name-based allowlist does
not grant namespace access: the namespaced Role above supplies that restriction.
The runner separately needs its normal pipeline execution permissions.

## Upgrade and revocation

Upgrade the API server and the ScheduledWorkflow controller together, apply the
release RBAC, and wait for old controller pods to stop before adding custom-account
grants. The release controller also needs the documented pipeline-read grants;
apply the complete release manifests rather than only changing images.

Review existing schedules before granting controller access to an account.
Schedules created before service-account authorization, or whose inputs were
previously changed through Kubernetes, must be reviewed and recreated through the
API to establish authorized inputs. Multi-user schedules created only as Kubernetes
CRs, without a corresponding API job, must also be recreated through the API.
Unresolvable namespaces or missing/replaced Kubernetes objects fail closed.

A follow-latest schedule intentionally executes future versions of its referenced
pipeline. Trust publishers of that pipeline to supply code running under the
schedule's approved account, or pin a reviewed version instead. Users who can
create arbitrary Kubernetes Workflows or Pods may already be able to execute as
other accounts; this API control does not replace Kubernetes admission policies
for that separate access path.

Removing the creator's `use` permission does not revoke an existing standing
schedule. Disable or delete the recurring run through the API to stop future
executions. Removing the controller's account-specific `use` grant, or removing
the account from the allowlist, also prevents subsequent API submissions under
that custom account. Already-created runs continue; terminate them separately if
required. Retain the normal permission checks when troubleshooting a denied tick.

## Scheduling regression test

The live integration suite includes an opt-in test for both latest and pinned
custom-account schedules. It modifies the backing CR before activation and checks
that the run succeeds with the original API-stored account and parameters.

On a disposable multi-user test installation, provision a custom runner using the
scoped grants above for the integration-test caller and controller. Set
`KFP_SCHEDULE_TEST_SERVICE_ACCOUNT` to that account, and run the existing v2
integration suite with the installation's usual endpoint/authentication flags and
`-run '^TestRecurringRunApi$' -testify.m '^TestRecurringRunCustomServiceAccount$'`.
The test requires permission to list and update ScheduledWorkflows and cleans up
pipeline resources in the test namespace; do not run it against production.
