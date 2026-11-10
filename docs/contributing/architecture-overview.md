# Architecture Overview

This document provides a set of progressive, high-level architecture diagrams for
new contributors and anyone looking for a quick mental model of Kubeflow Pipelines
(KFP). Each diagram adds one layer of detail so you can stop at the level that is
useful for your current task.

For the full, detailed cluster-wide diagram see
[`images/kfp-cluster-wide-architecture.png`](../images/kfp-cluster-wide-architecture.png).
For a per-pipeline anatomy breakdown see [Pipeline Anatomy](anatomy.md).

---

## Diagram 1 — Top-level system sections

KFP is divided into four conceptual sections. Each runs in a distinct Kubernetes
namespace (or logical boundary) in a typical installation.

```mermaid
graph TB
    subgraph user-ns["User / Client"]
        SDK["KFP Python SDK\n(compile · submit · query)"]
        UI["KFP Frontend UI\n(browser)"]
    end

    subgraph kubeflow-ns["kubeflow namespace\n(KFP Control Plane)"]
        API["API Server\n(REST / gRPC)"]
        DB[("Database\n(MySQL / PostgreSQL)")]
        OS[("Object Store\n(MinIO / GCS / S3)")]
        API --> DB
        API --> OS
    end

    subgraph argo-ns["kubeflow namespace\n(Kubernetes Execution)"]
        WC["Argo Workflow Controller"]
        MLMD[("MLMD\n(ML Metadata)")]
        WC --> MLMD
    end

    subgraph task-ns["Pipeline run namespace\n(Task Execution)"]
        DRV["Driver Pod\n(resolve inputs · patch pod spec)"]
        EXE["Executor Pod\n(user container + launcher)"]
        DRV --> EXE
    end

    SDK -- "compile to IR YAML\nthen submit" --> API
    UI -- "REST" --> API
    API -- "create Argo Workflow CR" --> WC
    WC -- "schedule pods" --> DRV
    EXE -- "read / write artifacts" --> OS
    EXE -- "record metadata" --> MLMD
```

**Key take-aways**

| Section | Responsibility |
|---|---|
| **User / Client** | Authors pipelines in Python; compiles to IR YAML; submits runs and monitors them |
| **KFP Control Plane** | Stores pipeline definitions, run records, and artifact metadata; owns the REST/gRPC surface |
| **Kubernetes Execution** | Argo Workflow Controller translates pipeline graphs into Kubernetes pod schedules |
| **Task Execution** | One driver + one executor pod pair per component invocation; does the real ML work |

---

## Diagram 2 — Single task request lifecycle

This diagram traces one pipeline run from browser click to recorded completion.

```mermaid
sequenceDiagram
    actor User
    participant FE as Frontend UI
    participant API as API Server
    participant DB as Database
    participant AWC as Argo Workflow<br/>Controller
    participant DRV as Driver Pod
    participant EXE as Executor Pod<br/>(launcher + user code)
    participant Store as Object Store
    participant MLMD as ML Metadata

    User->>FE: Trigger run
    FE->>API: POST /runs
    API->>DB: Persist run record (QUEUED)
    API->>AWC: Create Argo Workflow CR
    DB-->>FE: Run ID

    AWC->>DRV: Schedule driver pod
    DRV->>API: Resolve pipeline inputs / cache check
    DRV->>MLMD: Create execution record
    DRV->>AWC: Return podSpecPatch (volumes, node selectors, …)

    AWC->>EXE: Schedule executor pod (with patch applied)
    EXE->>Store: Download input artifacts
    EXE->>EXE: Run user component code
    EXE->>Store: Upload output artifacts
    EXE->>MLMD: Record outputs + execution status (COMPLETE)
    EXE->>API: Notify task completion

    API->>DB: Update run state (SUCCEEDED)
    DB-->>FE: Run status update
    FE-->>User: ✅ Run complete
```

**Key take-aways**

- The **Driver** is the only pod that talks to the API server before work begins.
  It is responsible for caching, input resolution, and dynamic pod configuration.
- The **Executor** (launcher + user container) never modifies the pipeline run
  record directly; it writes artifacts to the object store and metadata to MLMD.
- Each component in a pipeline graph produces **one Driver + one Executor pair**.

---

## Diagram 3 — Generic controller pattern

Several subsystems inside KFP (the Workflow Controller, the Scheduled Workflow
controller, the cache server) follow the same Kubernetes controller pattern. This
diagram explains it once so the pattern is recognisable whenever you encounter it.

```mermaid
graph LR
    subgraph api["Kubernetes API Server"]
        CR["Custom Resource\n(e.g. Workflow CR)"]
    end

    subgraph controller["Controller (control loop)"]
        direction TB
        INF["Informer\n(watch + local cache)"]
        QUEUE["Work Queue\n(rate-limited)"]
        REC["Reconciler\n(desired state → actual state)"]
        INF -- "enqueue key on\nAdd / Update / Delete" --> QUEUE
        QUEUE -- "dequeue key" --> REC
    end

    subgraph cluster["Cluster resources"]
        PODS["Pods / Jobs\n/ other objects"]
    end

    CR -- "watch events" --> INF
    REC -- "read current state" --> CR
    REC -- "create / update / delete" --> PODS
    REC -- "update status" --> CR
    PODS -- "status changes\ntrigger re-queue" --> INF
```

**Key take-aways**

| Component | Role |
|---|---|
| **Informer** | Subscribes to API server watch events; maintains a local in-memory cache to reduce API load |
| **Work Queue** | Decouples event receipt from processing; provides rate limiting and deduplication |
| **Reconciler** | Compares *desired* state (the CR spec) with *actual* state (live cluster objects) and drives them to match |

This loop runs continuously. If the reconciler crashes mid-reconciliation the item
is re-queued and retried, making the controller self-healing by design.

---

## Further reading

- [Pipeline Anatomy](anatomy.md) — detailed breakdown of the Argo Workflow tasks
  generated for a single-component pipeline
- [`images/kfp-cluster-wide-architecture.png`](../images/kfp-cluster-wide-architecture.png) — full cluster diagram with every component labelled
- [`docs/agents/architecture.md`](https://github.com/kubeflow/pipelines/blob/master/docs/agents/architecture.md) — key source paths and local runner notes (contributor reference, not part of the published docs site)
