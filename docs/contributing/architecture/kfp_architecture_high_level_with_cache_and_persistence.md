# KFP Architecture — High Level, With Cache Server + Persistence Agent

Same four groupings and namespace framing as `kfp_architecture_high_level.md` (Frontend, KFP Control Plane, Kubernetes Execution, Task Execution), but with **Cache Server** and **Persistence Agent** included in the KFP Control Plane box. Kept as a separate doc for anyone who wants the slightly fuller picture without jumping straight to the full-detail capstone.

```mermaid
flowchart TD
    subgraph FRONTEND["FRONTEND (user environment)"]
        FE["User's own container<br/>— submits via KFP SDK / kfp.Client()"]
    end

    subgraph KFP_CP["KFP CONTROL PLANE (namespace: kubeflow)"]
        API["KFP API Server<br/>(pipeline, run, task, artifact APIs)"]
        CACHE["Cache Server"]
        PA["Persistence Agent"]
        ARGO["Argo Workflow Controller"]
    end

    subgraph STORAGE["STORAGE (namespace: kubeflow)"]
        DB[("Metadata DB<br/>runs, tasks, artifacts")]
        OBJ[("Object Store<br/>artifact contents")]
    end

    subgraph K8S_EXEC["KUBERNETES EXECUTION (namespace: kube-system)<br/>schedules pods, starts containers —<br/>kept abstract here, see further-detail docs"]
        K8S["Kubernetes API Server<br/>+ Scheduler + kubelet"]
    end

    subgraph TASK_EXEC["TASK EXECUTION (namespace: example-user-namespace)<br/>one pod per task, created fresh each time"]
        TASK["Runs this task:<br/>resolves inputs, runs the user's code,<br/>reports results back"]
    end

    FE -->|"1: CreateRun"| API
    API -->|"2: write Run record"| DB
    API -->|"3: submit Workflow"| ARGO

    ARGO -->|"4: create task pod"| K8S_EXEC
    K8S_EXEC -->|"5: starts it"| TASK_EXEC

    TASK -->|"6: register + report state"| API
    TASK -->|"7: artifact bytes"| OBJ

    CACHE -->|"cache-hit check"| DB
    PA -->|"watches Workflow,<br/>syncs status"| ARGO
    PA --> API
```

## What this adds versus `kfp_architecture_high_level.md`

- **Cache Server** — checks/writes cache entries against the Metadata DB, so a task with a matching fingerprint (same image, command, inputs) can skip real execution and reuse a prior Artifact.
- **Persistence Agent** — watches the Argo Workflow (via the Workflow CR) and syncs Run/Task status into the Metadata DB independently of whatever Driver/Launcher directly report, so KFP's records stay correct even if something else fails to report in.

Both are real, standing `kubeflow`-namespace services — not part of the primary request path, but worth seeing one step earlier than the full-detail capstone for anyone who wants it.

## Where this fits

Optional step between `kfp_architecture_high_level.md` (#3) and `kfp_architecture_full_detail.md` (the capstone) — not currently part of the core numbered sequence. Covers the same ground as `kfp_architecture_master.md`'s original three-grouping diagram, but with the namespace split (`kubeflow` Control Plane vs. `example-user-namespace` Task Execution) that diagram was missing.
