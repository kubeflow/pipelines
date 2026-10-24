# Argo Exit POC Follow-Ups

Argo Workflows remains the default and supported runtime path for Kubeflow Pipelines today. The coordinator-managed
runtime remains an explicit optional path for focused development and validation. The items below are follow-up work
for that optional coordinator proof of concept rather than steps in a completed Argo replacement.

## Recovery durability

- Persist executor recovery handles on runtime tasks instead of only transient status metadata.
- For Kubernetes execution, store the executor pod UID and name durably enough to support restart-time reattachment.
- For local execution, persist whatever launcher or process handle is needed to distinguish recoverable work from work that must be restarted.
- Define explicit recover-vs-restart behavior per executor and test it against API server restarts.

## Multi-replica coordination

- Replace the current single-process in-memory work queue with a shared claiming model suitable for multiple API server replicas.
- Make run ownership, task dispatch, and cancellation idempotent across replicas.
- Ensure only one coordinator instance can actively own a run at a time.

## Pod watching and runtime observation

- Replace the current polling-style executor observation with a controller-runtime-based event loop.
- Watch only the coordinator-managed executor pods by using a narrow label selector.
- Cache pod state in-memory so run reconciliation does not need to re-list everything from the API server.
- Move any remaining workflow-reporter-style state observation to task- and pod-level reconciliation owned by the coordinator runtime.

## Lifecycle parity

- Add retry semantics for coordinator-managed runs.
- Finalize non-Argo log and log-archive behavior for task attempts.
- Revisit any remaining unsupported pipeline features once the coordinator/task model is stable enough to own them directly.

## Runtime selection and API shape

- Decide whether the temporary internal selector in `runtime_config.parameters` should remain the short-term opt-in
  contract or be replaced with a first-class API/model field before broader coordinator rollout.
- Keep coordinator-only behavior clearly isolated from the default Argo-backed create-run and recurring-run flows
  while that selector remains internal.
