# Runtime interface findings from implementation

These findings compare the [RFC](./README.md) with the task-scoped Kubernetes
runtime implementation and its failure-injection tests. They identify contracts
to specify more precisely, not a need to replace the overall architecture.

The highest-priority revisions are execution cleanup, the computation/publication
handoff, and attempt-scoped reporting. None requires introducing new multi-tenant
authorization.

## 1. Separate cancellation from execution cleanup

**Priority: high.** The proposed executor interface has `Cancel`, but no operation
for releasing retained execution resources. Workspace cleanup is a different
lifecycle.

Specify that:

- Canceling observation does not cancel execution.
- `Cancel` either confirms workload termination or returns an explicit
  pending/unknown result. Merely accepting a termination request is not proof
  that an execution-capacity slot can be released.
- Termination preserves the output and log sources required for publication.
- Execution cleanup is idempotent and occurs only after required results and logs
  are durably checkpointed or published.
- Cleanup failure retains the execution reference so cleanup can be retried,
  including after publication retries are exhausted.

The implementation added an `ExecutionCleaner` interface. Docker cancellation
originally removed the container before its logs could be captured; cancellation
now stops the container, and cleanup removes it later.

Evidence:
[cancellation/log regression](../../backend/src/v2/runtime/poc/cancellation_logs_test.go)
and [publication-exhaustion regression](../../backend/src/v2/runtime/poc/publication_failure_test.go).

## 2. Specify the computation-to-publication handoff

**Priority: high.** The RFC correctly gives computation and publication separate
retry budgets. A terminal outcome and unspecified durable references are not
enough to describe the handoff between a workload, launcher, and coordinator.

Define a versioned internal checkpoint/report contract containing:

- Attempt identity.
- Whether computation has completed.
- Durable output values or locations from which publication can resume.
- Publication phase and completion.
- Failure classification and publication-attempt accounting.

Clarify who stages bytes, performs uploads, reports artifact metadata, and
finalizes public task state when publication runs inside a remote launcher.
The framework must remain the authority for public completion regardless of
where the upload operation runs.

A Kubernetes workload can finish computation successfully and then exit
unsuccessfully because output publication failed. That must not authorize another
computation attempt, even when the computation retry policy would otherwise
permit it.

The implementation introduced a
[`RuntimeLauncherReport`](../../backend/src/v2/component/runtime_launcher.go).
The RFC should specify an internal reporting channel rather than making its
current task-custom-metadata representation the permanent public API.

Evidence:
[checkpoint/attempt matching and missing-Pod recovery](../../backend/src/v2/runtime/poc/launcher_checkpoint_test.go).

## 3. Make attempt-scoped reporting a concrete API contract

**Priority: high.** Define the contents and permitted operations of the RFC's
"scoped reporting context."

Require:

- Run, task-instance, and attempt identity on launcher reports.
- Atomic checks that the attempt is current and that cancellation/finalization
  still permits the mutation.
- Fencing of task updates, artifacts, metrics, artifact links, and bulk/ancestor
  mutations, not just the final completion callback.
- Defined rejection semantics for stale or canceled attempts.
- Idempotent replay identities for operations that create metadata.
- A distinction between launcher reports and trusted coordinator transitions.

Guarding only the terminal task update left other launcher mutation paths able
to modify a newer attempt's records. The implementation now routes those writes
through [transactionally fenced reporting](../../backend/src/apiserver/resource/runtime_reports.go).

This is attempt-lifecycle correctness. It does not replace authorization or
require the RFC's future multi-tenant features.

## 4. Define an error/result classification table

**Priority: medium.** Specify which failures return a method error and which
produce a terminal task result.

| Condition | Required interpretation |
| --- | --- |
| Backend observation/connectivity failure | Execution outcome remains unknown; reconcile rather than infer failure or resubmit. |
| Failed user computation | Terminal computation failure; apply the computation retry policy. |
| Invalid declared output after confirmed termination | Terminal output-contract error, not indefinite observation failure. |
| Failed publication after successful computation | Apply publication recovery/retries without replaying computation. |
| Failed execution cleanup | Retain recovery identity and retry cleanup, not computation. |
| Rejected stale-attempt report | Stop that mutation; do not reinterpret it as a new workload attempt. |

Use typed failure classifications and phase-specific retry semantics rather than
depending on loosely specified strings or a single retryability flag.

Invalid Docker output initially looked like an unresolved observation and could
remain unresolved indefinitely. The current
[`TaskFailure` and result types](../../backend/src/v2/runtime/poc/types.go)
provide a starting point for this contract.

## 5. Include a log-source contract

**Priority: medium.** The RFC promises backend-independent log retrieval, but
the proposed executor methods do not expose a log source.

Specify live-log access or a durable source reference, including:

- Task and attempt identity.
- Retention across worker loss and confirmed cancellation.
- Availability until required log publication and execution cleanup complete.
- Public API behavior for current versus historical attempts.
- Streaming behavior after partial delivery: a fallback must not silently replay
  already delivered content.

The implementation added an `ExecutionLogReader` interface. Task UUIDs, rather
than Pod names, are already the RFC's intended API identity and should remain so.

Evidence:
[executor log adapters](../../backend/src/v2/runtime/poc/plugin_logs.go) and
[task log retrieval](../../backend/src/v2/runtime/poc/task_logs.go).

## 6. Separate installation preflight from task compatibility validation

**Priority: medium.** `CheckRequirements(ctx)` can check backend access and
installation requirements, but it has no individual workload to validate.

Add a side-effect-free task-requirements validation contract, or equivalent
structured capability checks. It must not receive the full run, job, or DAG.

- Reject statically knowable unsupported requirements before starting execution.
- Validate input-dependent requirements after their inputs are resolved.
- Keep UI feature availability consistent with the admission checks.

Supporting a container workload does not imply supporting every container
requirement. For example, Docker rejects Kubernetes-style scheduling reservations,
accelerators, and lifecycle hooks in
[resource validation](../../backend/src/v2/runtime/poc/docker_container.go).

## Preserve the existing architectural boundaries

Some implementation defects were violations of invariants the RFC already states,
not missing design requirements. In particular:

- Unknown recovery outcomes must not authorize duplicate execution.
- Workspace storage and artifact storage have separate lifecycles.
- Durable workspace references and execution-target attachments are distinct
  concepts; sharing a handle type in the implementation is a refinement to make
  there, not a reason to weaken the RFC.
- Database-backed multi-replica leases are already described in the RFC. Their
  implementation is separate from the interface clarifications above.

Add interface-conformance tests for these lifecycle guarantees, especially the
first three findings, before treating the plugin contracts as stable.
