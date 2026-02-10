# Pod Status Feature - Change Tracking

This file tracks all changes made for the Pod Status feature to enable easy rollback.

## Image Version History

### API Server Image Versions

| Version | Tag Suffix | Status | Description |
|---------|------------|--------|-------------|
| A | `2.15.0-pod-status-A` | Working | Initial Pod Status feature (launcher) |
| B | `2.15.0-pod-status-B` | Working | Added `task_name` label to pods |
| C | `2.15.0-pod-status-C` | **BROKEN** | Attempted to add `iteration_index` label - caused MLMD connection errors |
| D | `2.15.0-pod-status-D` | Rollback | Rollback to B functionality |
| E | `2.15.0-pod-status-E` | **BROKEN** | Added `iteration_index` label to wrong templates - same failure as C |
| F | `2.15.0-pod-status-F` | Superseded | Fixed: labels only on driver templates (not Container Impl) |
| G | `2.15.0-pod-status-G` | Superseded | Task Path Enhancement - full path in label (63-char limit issue) |
| H | `2.15.0-pod-status-H` | **CURRENT** | Label + Annotation hybrid - label for simple name, annotation for full path |

### Frontend Image Versions

| Version | Tag Suffix | Status | Description |
|---------|------------|--------|-------------|
| K | `2.15.0-pod-status-K` | Working | Previous production frontend version |
| L | `2.15.0-pod-status-L` | Working | Previous frontend version |
| M | `2.15.0-pod-status-M` | Working | Added for-loop iteration index support in `getPodStatusAndEvents()` |
| N | `2.15.0-pod-status-N` | Superseded | Task Path Enhancement - path-based pod matching (before annotation) |
| O | `2.15.0-pod-status-O` | Working | Error pod filtering fix |
| P | `2.15.0-pod-status-P` | Bug | Error pods from other tasks incorrectly included |
| Q | `2.15.0-pod-status-Q` | Superseded | Fixed: error pods only included if pod name matches task |
| R | `2.15.0-pod-status-R` | **CURRENT** | Fixed: include unlabeled error impl pods when task has matching driver |

---

## Version R Changes (CURRENT - Unlabeled Error Impl Pod Fix)

### Problem
Version Q fixed error pods from other tasks showing on the wrong component, but introduced a new bug: impl pods with errors were now completely hidden from the UI. The UI showed "all success" even when there was a pod in error state.

### Root Cause
Impl pods in KFP v2 have names like `<workflow>-system-container-impl-<hash>` which do NOT contain the task name. The Version Q fix required `podName.includes(taskName)` to be true, which always fails for impl pods.

Additionally, impl pods don't have the `pipelines.kubeflow.org/task_name` label or `pipelines.kubeflow.org/task_path` annotation (these are only set on driver pods).

### Solution
Added detection for impl pods and associate them with tasks that have matching driver pods.

### Files Modified

#### frontend/server/k8s-helper.ts
Added `isImplPod()` helper and updated filter logic:

```typescript
// Helper to detect impl pods by name pattern
const isImplPod = (pod: V1Pod): boolean => {
  const name = pod.metadata?.name || '';
  return (name.includes('-impl-') || name.includes('system-container-impl')) && !name.includes('driver');
};

// Check if any driver pod (non-impl) matches this task
const hasDriverForTask = uniquePods.some(pod => !isImplPod(pod) && podMatchesTask(pod));

// For error impl pods WITHOUT task labels:
// Include them if this task has a driver pod.
if (isImplPod(pod) && hasContainerError(pod) && !podTaskLabel && !podTaskPath && hasDriverForTask) {
  return true;
}
```

### Logic
1. If a pod directly matches the task (by label, annotation, or name) → include it
2. If a pod is an impl pod with errors, has no task labels, AND the task has a matching driver pod → include it

This handles the case where impl pods weren't compiled with task labels but belong to a task that has a driver.

---

## Version H Changes (CURRENT API Server - Label + Annotation Hybrid)

### Problem
Kubernetes labels have a 63-character limit. With deeply nested DAGs, task paths like `root.for-loop-1.sub-dag-2.another-loop.process-item` could exceed this limit.

### Solution
Use both label (for backward compatibility) and annotation (for full path):

| What | Where | Value | Limit |
|------|-------|-------|-------|
| Label | `pipelines.kubeflow.org/task_name` | Simple task name: `process-item` | 63 chars |
| Annotation | `pipelines.kubeflow.org/task_path` | Full path: `root.for-loop-1.process-item` | 256KB |

### Files Modified

#### container.go - `addContainerDriverTemplate()`
```go
Metadata: wfapi.Metadata{
    Labels: map[string]string{
        // Simple task name for backward compatibility (max 63 chars)
        "pipelines.kubeflow.org/task_name": inputParameter(paramTaskName),
    },
    Annotations: map[string]string{
        // Full task path for sub-DAG differentiation (no length limit)
        "pipelines.kubeflow.org/task_path": inputParameter(paramTaskPath),
    },
},
```

#### dag.go - `addDAGDriverTemplate()`
Same pattern - label for simple name, annotation for full path.

#### dag.go - Task path construction
```go
// Construct full task path for pod labeling.
// Root-level tasks keep simple names; sub-DAG tasks get full path.
taskPath := name
if inputs.parentDagName != "" && inputs.parentDagName != "root" {
    taskPath = "root." + inputs.parentDagName + "." + name
}
```

---

## Version Q Changes (Fixed Error Pod Filtering)

### Problem
Version P had a bug where error pods from OTHER tasks were being included in the current task's pod list.

### Solution
Changed the logic to only include error pods if their pod name also contains the task name:

```typescript
// Before (Version P - broken):
if (hasContainerError(pod)) {
  return true;  // Included ALL error pods!
}

// After (Version Q - fixed):
const podNameMatchesTask = podName.includes(taskName);
if (hasContainerError(pod) && podNameMatchesTask) {
  return true;  // Only include error pods that match the task
}
```

---

## Version O Changes (Error Pod Filtering)

### Problem
After Task Path Enhancement (Version N), UI showed successful driver pods instead of error impl pods (e.g., ImagePullBackOff). The root cause: impl/executor pods don't have the `task_name` label.

### Solution
Added `hasContainerError()` helper function that detects container errors:
- ErrImagePull, ImagePullBackOff, CrashLoopBackOff
- CreateContainerConfigError, InvalidImageName, CreateContainerError
- RunContainerError, OOMKilled, Error
- Non-zero exit codes

Modified the filter to include pods with container errors.

---

## Version B Changes (WORKING - task_name label only)

### Files Modified in API Server (`backend/src/v2/compiler/argocompiler/`)

#### container.go
- Added `Metadata.Labels` to `addContainerDriverTemplate()` with `pipelines.kubeflow.org/task_name` label
- Label value: `inputParameter(paramTaskName)` which resolves to `{{inputs.parameters.task-name}}`

#### dag.go
- Added `Metadata.Labels` to `addDAGDriverTemplate()` with `pipelines.kubeflow.org/task_name` label
- Label value: `inputParameter(paramTaskName)` which resolves to `{{inputs.parameters.task-name}}`

---

## Rollback Instructions

### Rollback API Server to Version B (Working State)

If labels/annotations cause issues, simplify to just task_name label:

**container.go** - `addContainerDriverTemplate()`:
```go
Metadata: wfapi.Metadata{
    Labels: map[string]string{
        "pipelines.kubeflow.org/task_name": inputParameter(paramTaskName),
    },
},
```

**dag.go** - `addDAGDriverTemplate()`:
```go
Metadata: wfapi.Metadata{
    Labels: map[string]string{
        "pipelines.kubeflow.org/task_name": inputParameter(paramTaskName),
    },
},
```

### Rollback Frontend to Version Q

If Version R causes issues with error pods showing on multiple tasks, revert k8s-helper.ts:

```typescript
// Remove isImplPod helper and hasDriverForTask check
// Keep only: if (hasContainerError(pod) && podNameMatchesTask) { return true; }
```

---

## Frontend Changes (all versions)

These changes are in the frontend and are backwards compatible:

### Files:
- `frontend/server/k8s-helper.ts` - Pod listing with task/iteration filtering
- `frontend/server/pod-events-cache.ts` - File-based caching for pod events/status
- `frontend/server/handlers/pod-info.ts` - HTTP handlers for pod info/events
- `frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx` - Pod Status tab UI
- `frontend/src/lib/Apis.ts` - API client methods

The frontend changes are safe - they check for labels/annotations if they exist, with multiple fallback methods.

---

## Which Pods Have Labels

With the current implementation:
- **Driver pods** have `pipelines.kubeflow.org/task_name` label and `pipelines.kubeflow.org/task_path` annotation
- **Impl/Executor pods** do NOT have these labels/annotations

The frontend handles this by:
1. Matching by label/annotation (for driver pods)
2. Matching by pod name pattern (fallback)
3. Including unlabeled error impl pods when a matching driver exists (Version R)
