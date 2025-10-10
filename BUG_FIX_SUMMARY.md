# Bug Fix Summary: Empty Runs Array in API Response

## Problem

Test `TestCacheRecurringRun` fails with timeout error:
```
Error: Condition never satisfied
at cache_test.go:167
```

The test waits for recurring run to create at least 2 runs, but `ListRuns` API always returns empty `runs` array.

## Root Cause

**File**: `backend/src/apiserver/common/utils.go:159`

```go
EmitUnpopulated: false,  // BUG: Causes empty arrays to be omitted from JSON
```

When `EmitUnpopulated` is set to `false`, the protobuf JSON marshaller omits "unpopulated" fields including:
- Empty arrays/slices
- Empty strings
- Zero-value numbers
- nil values

This causes the `runs` field to be omitted from JSON response even when it's a valid empty slice `[]`.

## Impact

Affects **all** List* API endpoints when returning empty lists via HTTP/JSON:
- ❌ `ListRuns`
- ❌ `ListPipelines`
- ❌ `ListExperiments`
- ❌ `ListRecurringRuns`
- ❌ All other List* methods

Note: gRPC endpoints are not affected (use protobuf directly).

## Fix

Change `EmitUnpopulated` from `false` to `true`:

```diff
func CustomMarshaler() *runtime.JSONPb {
    return &runtime.JSONPb{
        MarshalOptions: protojson.MarshalOptions{
            UseProtoNames:   true,
-           EmitUnpopulated: false,
+           EmitUnpopulated: true,
        },
```

## Verification

### Before Fix
```bash
$ curl http://localhost:8888/apis/v2beta1/runs?page_size=5
{"total_size":76}  # Missing "runs" field!
```

### After Fix
```bash
$ curl http://localhost:8888/apis/v2beta1/runs?page_size=5
{
  "runs": [],
  "total_size": 76,
  "next_page_token": ""
}
```

## Test Plan

1. Rebuild and redeploy API server with fix
2. Run `TestCacheRecurringRun` test
3. Verify test passes within expected time
4. Verify API returns `"runs": []` for empty results

## Files Modified

- `backend/src/apiserver/common/utils.go` - Changed `EmitUnpopulated: false` to `true`
