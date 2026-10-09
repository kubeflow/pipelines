# Recurring-run prerequisite for 3.0

Multi-user installations with existing recurring runs must complete adoption into trusted
SQL scheduling state on a 2.18 build containing the legacy schedule recovery fix
before upgrading to 3.0. This includes disabled schedules. Merely starting a 2.18
API server is not evidence that SQL adoption has finished. Outstanding Kubernetes
synchronization can recover automatically during the 3.0 rollout.

The 2.18 implementation is tracked in
[PR #14741](https://github.com/kubeflow/pipelines/pull/14741). Select a build that
contains that change; this guide does not identify a released patch containing it.

This is a **scheduling prerequisite**, not confirmation that the complete 2.18 to
3.0 upgrade is ready. MLMD migration and removal of V1 execution support are
separate concerns. Adoption preserves scheduling state for supported V2 pipelines;
it does not make a V1-only pipeline executable on 3.0. The master upgrade workflow
remains paused pending the migration work tracked in
[#14029](https://github.com/kubeflow/pipelines/issues/14029).

## Automatic startup check

Before database migrations, the multi-user API checks existing schedule state.
It rejects startup when a recurring run lacks trusted API-owned scheduling state,
including a disabled recurring run, or when an existing adoption receipt is
invalid. Earlier 2.18 native schedules can have trusted SQL state without an
adoption receipt; those schedules satisfy this prerequisite. Receipts are checked
when present. A valid receipt still awaiting Kubernetes synchronization does not
block startup. Missing SQL state always blocks startup, as do missing required
scheduling-state tables with existing schedules. A fresh
database, or a database with no recurring runs, passes this scheduling check.
Other upgrade checks may still apply.

A rejection leaves database migrations unstarted and reports the unmet
prerequisite. Continue running the fixed 2.18 installation with its complete
manifests so its normal background reconciliation can finish. If 2.18 reports a
record-specific error, resolve that error there, then retry the 3.0 rollout. There
is no separate adoption flag, migration Job, manual SQL update, or requirement to
recreate healthy schedules. Do not remove receipts or manufacture state rows to
bypass the check.

The check happens during each API startup. Completing the prerequisite on one
schedule does not excuse missing state on another schedule, and disabling a
schedule does not bypass it.

## Rolling handoff and continued recovery

Apply the complete manifests for the API server and ScheduledWorkflow controller,
including their managed-writer permissions. Fixed 2.18 and 3.0 processes use the
same writer protocol to recognize a completed managed rollout. This coordination
retries automatically; operators do not need to shut down all schedule writers
or perform a coordinated cutover.

During an incomplete handoff, schedule creation and enable/disable operations
can return retryable `Unavailable` errors before changing
persistent state. Retry after the rollout completes. Schedule reads and ordinary
run operations remain available, subject to their normal authorization checks.
The writer check covers the standard managed API and controller Deployments;
custom deployment layouts and independent writers require separate assessment.

Master retains the state and receipts produced by 2.18 and reconciles outstanding
Kubernetes synchronization from trusted SQL state after the handoff. Each affected
schedule remains blocked from claiming a new execution until its synchronization
finishes. API startup must allow valid pending receipts: requiring synchronization
first would prevent the rollout from completing while the reconciler itself waits
for that rollout. It does not contain a second
legacy conversion path. Scheduling counters, pending execution identities, and
concurrency limits remain authoritative across retries; recovery must not reseed
them from editable Kubernetes status.

See [scheduled service accounts](scheduled-service-accounts.md) for authorization,
revocation, and execution recovery limits.
