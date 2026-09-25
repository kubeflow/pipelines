# Debug Pause — moving to a launcher-owned design

**Status:** prototype built and verified end-to-end (before / after / on_error) on a
local standalone KFP cluster.
**Context:** follow-up on PR #13546 (`feat(frontend): fix artifact... set_debug_pause UI`).
**Audience:** the original PR author + reviewers, to align on direction before finalizing.

---

## TL;DR

The original PR surfaces debug pause by **inferring a `Paused` run state from the
presence of the `ARGO_DEBUG_PAUSE_*` env var**. Testing showed this can't be made
correct: the env var lives for the whole pod lifetime, so a normally-running task is
reported "paused" for most of its duration, and there is **no Argo-side runtime signal**
that distinguishes "executing" from "parked at the barrier."

We moved the pause **into the KFP launcher** instead of relying on Argo's emissary. The
launcher is already the main-container process, so it can park around the user code, stay
observable, and be resumed from the UI — with **no pod `exec`, no sidecar, and no new
RBAC**. Debug pause becomes a **separate overlay value**, never a run state, which also
keeps it clean under the MLMD removal (#12430).

---

## Background: what the original PR does

- `set_debug_pause()` injects `ARGO_DEBUG_PAUSE_BEFORE/AFTER/ON_ERROR=true` into the
  pod-spec-patch. Argo's emissary honors these and holds the pod at the barrier.
- The PR detects those env vars (backend `NodeStatuses`, frontend `WorkflowParser`,
  `DynamicFlow`) and, for a `Running` node with the env, reports the node state as
  **`Paused`**, rendering a pause badge in the graph.

This is a real, worthwhile goal — debug pause has no visual affordance today. The issue is
purely the **mechanism** used to detect and act on it.

## What we found (empirically, on a live cluster)

1. **"Configured to pause" ≠ "currently paused."** The `ARGO_DEBUG_PAUSE_*` env is present
   the entire pod lifetime. With the default `after=True`, we observed the task reported
   `Paused` for its **whole execution window** (`slow-task=PAUSED` while the container was
   actively running its work), not just at the barrier. This is the core issue the
   maintainer review also flagged.
2. **Argo exposes no runtime "parked" signal.** At every barrier (before, after, on_error)
   the pod is `2/2 Running` and the Argo node phase is `Running` — identical to normal
   execution. The only ground truth is the emissary's marker file *inside* the container,
   which is not observable via the k8s/workflow API.
3. **MLMD can't disambiguate `before` either.** The driver's `CreateExecution` sets the
   execution to `RUNNING` before the before-barrier, so parked-before and running-after-
   resume both read `RUNNING`. (`after` is distinguishable via MLMD — execution `COMPLETE`
   while the node is still `Running` — but that's MLMD-coupled and #12430 is removing MLMD.)

## The core constraint that drove the redesign

The resume marker (and the pause itself) lives **inside the pod**. The only things that can
observe or release it are (a) a process already in the pod, or (b) an `exec` into the pod.
There is **no external write/observe path** — an Argo emptyDir isn't reachable from the
apiserver, a controller, or a shared disk.

So a **UI-driven resume** of an emissary pause forces one of: apiserver `pods/exec` (broad,
cluster-wide privilege → a compromised apiserver becomes cluster-wide RCE), an injected
**sidecar** (per-pod overhead), or the **user's own `exec`** (CLI/kubectl only, no button).
None of these are attractive.

The way out: **make the in-pod agent be the KFP launcher**, which is already running as the
pod's main process.

---

## The new direction: launcher-owned debug pause

Instead of Argo's emissary pausing the container, the **KFP launcher** parks around the user
code at the requested barrier:

- Because the launcher is PID-1-ish in the main container, blocking there **keeps the pod
  alive for `kubectl exec`** — same interactive-debugging value as the emissary.
- Because it's the process that's already there, it is also the agent that **observes a
  resume request and continues by itself** — no `exec`, no sidecar.
- It publishes/observes the pause via a small, swappable transport, so the design is
  decoupled from MLMD (#12430).

Debug pause is modeled as a **separate overlay value**, never folded into `RuntimeState`.

### Flow

```
set_debug_pause()  ──►  KFP_DEBUG_PAUSE_{BEFORE,AFTER,ON_ERROR}=true   (SDK, container env)

launcher (in the task pod):
  prePublish → [pause BEFORE] → run user code → [pause ON_ERROR if it failed]
                                              → [pause AFTER] → publish COMPLETE
  while parked:  GET  /apis/v2beta1/runs/{run}/pods/{pod}:debug-pause-status
                 → {"resumeRequested": bool}   (the launcher's ONLY apiserver call)

UI "Resume" button ─► POST /apis/v2beta1/runs/{run}/tasks/{task}:resume
                       apiserver annotates the Argo Workflow (workflows:patch it already has)
                       launcher's next poll sees it → releases the barrier → continues
```

---

## Architecture / what changed across the stack

### SDK
- `set_debug_pause()` now emits `KFP_DEBUG_PAUSE_*` (read by the launcher) instead of
  `ARGO_DEBUG_PAUSE_*` (read by the emissary), so the launcher pauses and the emissary does
  not. Same `before/after/on_error` surface; docstring updated.

### Launcher (`backend/src/v2/component/`)
- **`debug_pause.go`** (new module — the substance):
  - `DebugPauseConfig` from env; a `PauseSignaler` interface (the transport seam); the
    `Pause()` loop; and a minimal `kfpApiPauseSignaler` (one GET poll).
  - **Robust failure handling** (unit-tested): publish failure → still parks; resume-poll
    errors → keep polling with escalating logs; a **max-duration safety valve** so a lost
    signal can't wedge a pod forever; context-cancel → best-effort clear.
- **`launcher_v2.go`**: thin hooks at the three barriers only.

### Apiserver — **no `pods/exec`, no new RBAC**
- **`resource/debug_pause.go`**: `ResumeDebugPauseBarrier` records the resume by
  **annotating the Argo Workflow** (uses the `workflows:patch` the apiserver already has);
  `IsDebugPauseResumeRequested` reads it.
- **`server/run_debug_pause_server.go`**: `…:resume` (user action, SAR-gated on the run's
  namespace, audited) and `…:debug-pause-status` (launcher poll).
- **`common/util/workflow.go`**: **reverts** the `Paused` `RuntimeState` synthesis — pause
  is no longer a state.

### Frontend — pause is an overlay, not a state
- **`DebugPause.ts`**: detects debug-pause config from `run.pipeline_spec` (static,
  independent of MLMD/state). The badge shows while a configured task's node is `RUNNING`;
  it is **not** an inferred "Paused" state.
- **`RunDetailsV2.tsx` / `DynamicFlow.ts` / `ExecutionNode.tsx`**: render the badge overlay;
  clear it on resume / completion / terminal run state; Resume button posts to the endpoint.
- **`DebugPauseResumeHelp.tsx`**: node-detail panel with the Resume button + copy commands.
- Reverts the original PR's V1-graph (`WorkflowParser`) state manipulation.

---

## Security

The launcher-owned design introduces **zero new privilege**:

- **No `pods/exec`** anywhere (the exec-based resume we prototyped would have needed
  cluster-wide `pods/exec` on the apiserver — a compromised apiserver → cluster-wide RCE;
  rejected).
- **No sidecar** (per-pod overhead; rejected).
- The apiserver only uses the **`workflows:patch`** it already has to record resume; the
  user-facing resume is **`SubjectAccessReview`-gated** on the run's namespace and audited.

**One honest caveat:** the launcher→apiserver **status poll** is a system GET with no user
identity, so it is currently unauthenticated. That launcher↔apiserver auth gap is exactly
what #12430's `apiclient/auth` formalizes — the endpoint is intentionally the minimal seam
that rebases onto it.

---

## MLMD removal (#12430) compatibility

This design is explicitly built to survive #12430 (Replace MLMD with KFP Server APIs):

- **Config** comes from `pipeline_spec` (authoring-time, static) — unaffected by MLMD removal.
- **No MLMD coupling** was added anywhere (an earlier MLMD-signal prototype was reverted).
- The launcher's **entire KFP-API footprint is one GET poll**, isolated behind
  `PauseSignaler`. When #12430 lands (it adds `backend/src/v2/apiclient/kfpapi`, the
  first-class launcher→apiserver client), the rebase is: **replace one method body** with the
  typed client. The `Pause()` loop and the launcher hooks don't change.

In other words, this PR introduces the *initial, minimal* launcher→apiserver hook that
#12430 was going to need anyway, in a form that's a small factor to address on rebase.

---

## Status & verification

- **Backend:** `go build` clean; 7 launcher unit tests (the `Pause` failure paths).
- **Frontend:** `tsc` 0 errors; 15 unit tests (8 config-detection, 7 graph).
- **End-to-end on a live cluster (launcher + apiserver + frontend + SDK rebuilt):**
  - `before`: launcher parks pre-execution → resume → runs.
  - `after`: runs → parks post-execution → resume → completes (green).
  - `on_error`: runs → **fails** → parks → resume → task correctly ends **FAILED** (the error
    is not masked).

---

## Open items / follow-ups

1. **CLI + panel resume commands are still the old emissary path.** `kfp run resume` and the
   panel's `kubectl … touch <marker>` / `kfp run resume` copy-commands `touch` the emissary
   marker, which the launcher no longer watches. Only the **UI button** resumes under
   launcher-owned. These should be reworked to **POST to the resume endpoint** (still
   user-credentialed) and the `kubectl touch` line dropped.
2. **`resume` could be a proper gRPC RPC** on `RunService` (alongside `TerminateRun` /
   `RetryRun`) rather than a plain-HTTP handler — cleaner and it removes the separate
   `RunDebugPauseServer`. Skipped here only to avoid proto codegen; worth doing for the real
   PR. The launcher poll would stay a lightweight HTTP endpoint.
3. **Launcher↔apiserver auth** on the status poll — defer to / coordinate with #12430.
4. **Precise "parked now" in the UI** is currently approximated ("debug-pause enabled while
   running"). The launcher *knows* precisely; we made `PublishPauseState` a no-op to keep the
   API footprint minimal. It can publish precise state on the same seam later.
5. **Safety-valve default** (max pause duration) is currently unlimited; decide on a sensible
   default / operator override.

---

## Trade-offs (stated plainly)

- **Cost:** we take on the pause/resume loop and its failure modes that Argo's emissary
  otherwise provides (crash-while-parked, resume races, timeout policy). It's more code on the
  launcher hot path (carefully gated off when unconfigured).
- **Benefit:** it's the **only** design that yields UI-driven resume + accurate, observable
  pause **without** `pods/exec` or a sidecar — because it reuses the process that's already in
  the pod. It's engine-agnostic, handles `before/after/on_error` uniformly, keeps the exec-in
  debugging experience, and fits KFP v2 / #12430's direction.

The original PR's instinct (surface debug pause in the UI, resume from the UI) is right — this
changes only the mechanism from "infer a state from Argo's emissary env" to "the launcher owns
the pause and exposes it as a separate, resumable overlay."
