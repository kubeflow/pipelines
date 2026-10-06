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

The API creates generic ScheduledWorkflows without embedded execution templates in
multi-user mode, even when plugins are disabled. Startup reconciliation also
preserves generic routing for schedules it processes; it does not convert every
existing tenant CR. This routes unmodified API-created schedules through
the API on older controllers. It is defense in depth, not a substitute for the
upgraded multi-user controller: editable CR content must never select a direct
execution path. The multi-user controller role no longer grants `workflows/create`;
retain this restriction in custom manifests.

The API also owns durable scheduling state, initialized when a recurring run is
created through the API. It computes the actual scheduled time from the stored
trigger and catch-up policy, ignoring the timestamp supplied by the controller,
and enforces the stored maximum concurrency. Editing the CR's trigger, concurrency,
or scheduling counters may cause submissions to be rejected; it cannot authorize
earlier or additional executions. The API records a pending tick's index and time
before submitting it. A retry of that tick retains its index, scheduled time,
selected pipeline version, and run identity, with a deterministic workflow name
to prevent duplicate executions. Saving the run also completes its pending tick
in the same database transaction.

If that run is deleted before the controller records the successful submission,
retrying the same tick returns its retained identity and timestamps with an
unspecified runtime state. This acknowledgement advances the controller without
recreating the deleted run or workflow, and still requires the normal permissions.
Metadata-only acknowledgements and retained static workflows do not require the
original pipeline version to remain available. Retained workflows that need the
V2 compiler-patch exception must still establish their authoritative V2 source,
as described below. A schedule without a pipeline version ID resolves the
pipeline's default version again for its next tick. SQL storage selects the newest
ready version. Kubernetes-native storage selects the version configured in the
Pipeline's `spec.defaultVersionName`; only when that field is unset does it select
the newest owned version. An invalid configured default fails resolution rather
than falling back to the newest version. Already-claimed ticks retain their
selected version even if the default changes or a newer version is uploaded.
Recurring-run request keys (the run display names) must contain at most 255
characters so the run and its scheduling state can both be stored.

Single-user controller behavior remains unchanged. In a custom multi-user
installation, explicitly set `--multiUser=true`; authentication headers or bearer
tokens alone do not enable this execution mode.
Set the same `CRON_SCHEDULE_TIMEZONE` on the API server and controller; the supplied
manifests use `pipeline-install-config.cronScheduleTimezone` for both. Both binaries
embed Go timezone data, so non-UTC schedules do not depend on an OS tzdata package.

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

Before upgrading, block recurring-run creation and recreation for all clients.
Keep this block until every old API server replica has stopped and its in-flight
persistence-agent reports have drained. During a rolling upgrade, an old API can
still overwrite a new schedule's stored inputs from its editable Kubernetes CR,
even when the schedule is disabled and already has API-owned scheduling state.

Upgrade the API server and the ScheduledWorkflow controller together and apply
the complete release manifests, including the controller's pipeline-read grants.
Also wait for old controller pods to stop before adding custom-account grants or
enabling replacement schedules.

Review existing schedules before granting controller access to an account.
Schedules created before service-account authorization, or whose inputs were
previously changed through Kubernetes, must be reviewed and recreated through the
API to establish authorized inputs. Multi-user schedules created only as Kubernetes
CRs, without a corresponding API job, must also be recreated through the API.
**Every pre-existing multi-user schedule without API-owned scheduling state must
be reviewed and recreated before its next execution.** It will otherwise stop
submitting runs, including when the UI still shows it enabled. Audit mode does not
bypass this requirement. Disabled schedules also require recreation before use.
The API does not initialize trusted counters from an existing CR's status.

An old `jobs` row is not sufficient evidence of authorized inputs: earlier
controller reports could overwrite its parameters, trigger, catch-up policy and
concurrency settings from the CR. Automatically seeding an index and timestamp
would trust those settings without reviewing them. Preserve the original pipeline
and parameters from an independently reviewed source when recreating a schedule;
do not blindly copy its current CR or database row.
Unresolvable namespaces or missing/replaced Kubernetes objects fail closed.

### Inventory before completing the upgrade

Each multi-user API replica logs `recurring_run_migration action=recreate` at
startup for every job missing trusted scheduling state, including disabled jobs.
The messages contain its ID, namespace, ScheduledWorkflow name and enabled state;
they omit execution inputs. A final `affected_jobs` count summarizes the inventory.
An `inventory_failed` error means the inventory is incomplete, not that no jobs
need migration. Rerun the read-only inventory below after database recovery.

Inspect the API startup logs and Kubernetes schedule identities:

```bash
kubectl -n kubeflow logs deployment/ml-pipeline --all-pods=true --all-containers=true --prefix | grep recurring_run_migration
kubectl get scheduledworkflows.kubeflow.org --all-namespaces \
  -o custom-columns='NAMESPACE:.metadata.namespace,NAME:.metadata.name,UID:.metadata.uid,ENABLED:.spec.enabled'
```

Use a read-only database connection after the upgraded API has created the
`recurring_run_states` table. For MySQL:

```sql
SELECT j.`UUID`, j.`Namespace`, j.`Name`, j.`Enabled`
FROM `jobs` AS j
LEFT JOIN `recurring_run_states` AS s ON s.`JobUUID` = j.`UUID`
WHERE s.`JobUUID` IS NULL
ORDER BY j.`UUID`;
```

For PostgreSQL:

```sql
SELECT j."UUID", j."Namespace", j."Name", j."Enabled"
FROM jobs AS j
LEFT JOIN recurring_run_states AS s ON s."JobUUID" = j."UUID"
WHERE s."JobUUID" IS NULL
ORDER BY j."UUID";
```

Match database IDs to Kubernetes UIDs. A CR without a matching API job also
requires recreation. The SQL result alone cannot inventory CR-only schedules.
Before the first upgraded API startup, all existing jobs require review; the new
state table does not exist yet.

Also review and recreate every schedule created or recreated while old and new
API replicas overlapped, including disabled schedules. Use the rollout window and
creation records to identify them. The startup logs and SQL inventory above omit
those that already have scheduling-state rows.

After all old API replicas and their in-flight reports have drained, resume
creation. Disable the old schedule through the API, review its intended inputs
and account, and recreate it through the API. KFP 2.18 continues to support V1 workflows
where the namespace policy permits them, as well as V2 PipelineSpec IR. Keep the replacement
disabled until the old controller pods have stopped and existing executions have
been accounted for. Disabling a schedule does not terminate its runs. Do not insert
scheduling-state rows or reset counters directly in the database to bypass review.
Repeat the inventory after recreation and remove obsolete disabled schedules
through the API when their history is no longer needed.

A schedule without a pipeline version ID can execute future default versions of
its referenced pipeline. Trust publishers of that pipeline to supply code running under the
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

## Recovery limits and follow-up validation

The concurrency limit counts non-terminal API run records. A missing Workflow or
an interrupted persistence report can leave a record consuming a slot. Archiving
a run only changes its storage visibility; it does not prove that execution has
stopped. Old or archived records therefore still count. Investigate the execution
and persistence-agent health, and reconcile its lifecycle through supported run
operations before restarting a blocked schedule. Never free capacity solely from
an age cutoff or by hiding records. Automatic stale-record reconciliation remains
a follow-up.

A pending tick retains its original request key until its run is persisted. If
that key changes before completion, the API rejects a new tick. The persistence
agent can recover a Workflow that was already created, but interruption before
creation can require operator intervention. Disable the schedule, establish
whether the pending execution exists, and recover persistence or recreate the
reviewed schedule after accounting for existing runs. Do not delete a pending
claim or edit its counters: that can lose the identity needed for deduplication.
Automatic resumption of the trusted pending claim is a follow-up.

The Workflow identity checks support retry and response-loss recovery; they do
not authenticate a creator against someone who can directly create or edit
Workflows. A concurrent API request does not persist an existing Workflow as a
new run; it waits for the creator or persistence agent. Editable creator or
plugin-parent annotations would not establish trusted provenance. Restrict direct
Workflow/Pod permissions and apply admission policy where that boundary is
required. Stronger execution provenance is a separate follow-up.

The backend presubmit runs dedicated MySQL 8.0 and PostgreSQL 16 transaction
regressions with twelve independent connections. They cover identical and competing
request keys, duplicate persistence, atomic rollback and reporter recovery, and
completed-request tombstones. Database-observed lock waits verify that a blocked
claim sees a newly committed active run or schedule disable before proceeding.
The harness verifies MySQL REPEATABLE READ and PostgreSQL READ COMMITTED isolation.

To repeat this coverage against disposable local databases, set
`KFP_RECURRING_MYSQL_TEST_DSN` and `KFP_RECURRING_POSTGRES_TEST_DSN`, then run:

```bash
go test ./backend/src/apiserver/storage -run '^TestRecurringRunProductionDatabases$' -count=5
```

The tests create and remove only their uniquely named databases/schemas. The test
roles need database/schema and trigger DDL privileges, plus access to MySQL
`performance_schema.data_lock_waits`/`threads` or PostgreSQL session lock metadata.
Without a DSN, that database's cases are skipped. These storage tests do not replace
live schedule firing, coordinated rollout, or populated-cluster upgrade acceptance.

## Scheduling regression test

The live integration suite includes an opt-in test for both default-following and pinned
custom-account schedules. It modifies the backing CR before activation and checks
that the run succeeds with the original API-stored account and parameters.
Backend regression tests also cover rejected scheduling-state tampering and
retries that preserve the pending tick's execution identity.

On a disposable multi-user test installation, provision a custom runner using the
scoped grants above for the integration-test caller and controller. Set
`KFP_SCHEDULE_TEST_SERVICE_ACCOUNT` to that account, and run the existing v2
integration suite with the installation's usual endpoint/authentication flags and
`-run '^TestRecurringRunApi$' -testify.m '^TestRecurringRunCustomServiceAccount$'`.
The test requires permission to list and update ScheduledWorkflows and cleans up
pipeline resources in the test namespace; do not run it against production.

## Temporary audit mode for migration

Service-account authorization is enforced by default in 2.18.0, including on
upgrades. Administrators can temporarily set the API server environment variable
`KFP_SECURITY_SERVICE_ACCOUNT_MODE=audit` while configuring the allowlist and scoped
RBAC grants described above. The only supported modes are `enforce` and `audit`;
unset or empty configuration means `enforce`, and invalid values are rejected.

**Audit mode restores the security exposure addressed by service-account
authorization.** It evaluates the allowlist and, in multi-user mode, the caller's
`serviceaccounts/use` permission, but allows policy denials. It logs a startup
warning and structured events with `security_audit control=service_account mode=audit`,
the failed check, namespace, and requested service account. These
events describe a bypassed check, not confirmation that the overall request
succeeded. They do not include pipeline inputs, tokens, or the configured
allowlist. Authentication failures and authorization-service errors still reject
the request.

This is an administrator deployment setting, not a request or ScheduledWorkflow
field. Apply the same configuration to every API server replica and complete the
rollout before relying on a mode change. For example, with the standard deployment:

```bash
kubectl -n kubeflow set env deployment/ml-pipeline KFP_SECURITY_SERVICE_ACCOUNT_MODE=audit
kubectl -n kubeflow rollout status deployment/ml-pipeline
```

Persist the setting in your deployment configuration if using GitOps. No separate
controller audit setting is needed: the upgraded multi-user controller submits
scheduled runs through the API server, which applies the same mode on each tick
and replay. Keep the controller in multi-user mode. Audit does not restore
CR-only schedules, trust CR edits to execution inputs, or bypass namespace
permissions, schedule identity checks, or disabled-schedule checks. Single-user
mode still evaluates the allowlist; its existing lack of user RBAC checks is
unchanged.

Use audit events to configure the required allowlist and narrowly scoped user
and controller grants. Exercise both immediate runs and each recurring run's
execution path, then restore enforcement:

```bash
kubectl -n kubeflow set env deployment/ml-pipeline KFP_SECURITY_SERVICE_ACCOUNT_MODE=enforce
kubectl -n kubeflow rollout status deployment/ml-pipeline
```

Before treating the mode change as complete, also verify that every old API Pod
has terminated and all remaining API Pods use the requested mode. Deployment
rollout readiness alone does not establish that terminating Pods have stopped
serving existing connections. During this overlap, a request can still reach a
process with the previous mode. For a strict cutover, pause new submissions and
disable recurring runs until this check completes, then resume them. Runs
already admitted under audit are not retroactively rejected.

A previously denied schedule may retain up to 360 seconds of controller retry
backoff after permissions or the policy mode change. Disabling and re-enabling
the schedule does not reset that delay. Allow time for the retry and workload
completion when validating the change; an immediate absence of audit logs does
not establish that the schedule was evaluated.

Verify both run types succeed under enforcement before completing migration.
An absence of audit warnings alone does not prove that unexercised schedules are
ready. Removing the opt-out does not stop runs already executing.

We plan to remove audit mode in **3.0.0**, tracked in
[#14367](https://github.com/kubeflow/pipelines/issues/14367). Deployments using audit
mode remain exposed even though version-based vulnerability scanners may identify
2.18.0 as patched.

### Combining main-account and workflow-identity modes

The API server has two independent temporary controls:

| `KFP_SECURITY_SERVICE_ACCOUNT_MODE` | `KFP_SECURITY_WORKFLOW_IDENTITY_MODE` | Behavior |
| --- | --- | --- |
| `enforce` | `enforce` | Enforce the main account and expanded workflow identities. This is the default. |
| `audit` | `enforce` | Audit main-account policy denials; enforce additional identities and complete inspection. |
| `enforce` | `audit` | Enforce the main account; audit additional-account policy denials and incomplete local inspection. |
| `audit` | `audit` | Audit both policy scopes; authentication, authorization-service failures, namespace access, and scheduling trust checks still block. |

The service-account setting does not relax additional-account enforcement, and the
workflow-identity setting does not relax the main account. A templated main account
remains invalid. Invalid modes fail startup and are rejected if observed during
request handling. All SubjectAccessReview evaluation errors, including a response
that also says `allowed`, remain blocking. Neither control is a request parameter,
and neither disables fresh-workflow validation or permits trusting editable CR
execution inputs. The old unreleased configuration names are not aliases.

For recurring runs, grant the submitting user and the controller only the named
accounts actually required by the workflow, including helper/template accounts.
Extend the `resourceNames` lists in the scoped Role examples above deliberately;
do not grant unrestricted `serviceaccounts/use` merely to silence findings. Test
both the schedule-creation caller and the controller identity that submits ticks.

Audit findings are not a complete inventory when inspection stops at a dynamic or
malformed patch. A warning records a permitted policy violation or incomplete
inspection, not success of the overall request. Exercise immediate runs, pinned and
default-following schedules, plugin changes, retries, and re-enabling schedules. Return
each control to enforcement independently and verify the next scheduled execution;
changing a setting does not retroactively stop existing workloads.

Both audit controls are planned for removal in 3.0 under
[#14367](https://github.com/kubeflow/pipelines/issues/14367).

Retained recurring-run acknowledgements inspect the stored runtime identities when
the manifest is available, including persistence-agent-recovered runs. A stored main
account does not hide additional identities in that manifest. Metadata-only
acknowledgements, including a consumed tick whose run was deleted, still authorize
the main account but do not recreate an execution or fetch a deleted pipeline
version just to reconstruct an identity inventory. Malformed retained execution
records require recovery; audit does not make invalid execution metadata valid.

The V2 compiler-patch exception uses the same authoritative source checks as
retry and re-enablement. For referenced pipelines, only the exact saved version
can establish V2 provenance. A deleted version, missing saved source, unrelated
manifest, or newer V2 version cannot supply that exception. Under workflow-identity
enforcement, a retained dynamic patch without that provenance blocks
acknowledgement; static workflows can still pass ordinary identity inspection.
Workflow-identity audit can temporarily permit incomplete inspection, but
main-account enforcement remains independent.

Additional identities are checked before a new tick is claimed and again after
plugin mutation. A post-plugin denial can leave that tick pending, as with other
plugin failures; correct the policy or plugin output and retry the pending tick.
Expired-retry recovery may acknowledge an already-running execution without
starting another execution; changing policy does not stop that running workload.
