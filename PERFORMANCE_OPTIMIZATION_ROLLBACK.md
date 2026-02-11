# Performance Optimization Rollback Guide

## Current State Before Optimization (2025-02-10)
- Frontend: `2.15.0-pod-status-S`
- API Server: `2.15.0-pod-status-I`

## Changes Being Made

### P0: MLMD refetchInterval increase
**File:** `frontend/src/pages/RunDetailsV2.tsx`
**Change:** Increase `refetchInterval` from 10000ms (10s) to 60000ms (60s)

### P0: Pod list limit
**File:** `frontend/server/k8s-helper.ts`
**Change:** Add `limit` parameter to `listNamespacedPod()` calls

### P1: Pod status refetch interval
**File:** `frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx`
**Change:** Increase pod status `refetchInterval` from 5000ms to 30000ms

## Rollback Instructions

To rollback these changes:

```bash
cd /Users/ovolpres/Documents/Projects/PEPSICO/KFP/pipelines

# Option 1: Revert specific files
git checkout HEAD -- frontend/src/pages/RunDetailsV2.tsx
git checkout HEAD -- frontend/server/k8s-helper.ts
git checkout HEAD -- frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx

# Option 2: If changes were committed, revert to previous commit
git revert HEAD

# Option 3: Rebuild previous version
docker build --platform linux/amd64 \
  -t kubeflownonprod01eus94136.azurecr.io/custom-kubeflow-pipelines/kfp-frontend:2.15.0-pod-status-S \
  -f frontend/Dockerfile .
```

## Version History
- Frontend: `2.15.0-pod-status-T` - Performance optimizations (refetch intervals + pod list limit)
- Frontend: `2.15.0-pod-status-U` - Added debug logging for cache retrieval diagnostics
- Frontend: `2.15.0-pod-status-V` - Background pod watcher for automatic caching (supports scheduled/cron runs)
- Frontend: `2.15.0-pod-status-V4` - **CURRENT** - Optimized pod watcher (only watches Running/Pending pods, reduced logging to prevent event loop blocking)
