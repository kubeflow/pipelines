# Troubleshooting Guide

This guide covers common issues and solutions when using the Kubeflow Pipelines SDK.

## Table of Contents

- [Cloud Logging Issues](#cloud-logging-issues)
  - [403 Permission Denied with Resource Limits on Vertex AI](#403-permission-denied-with-resource-limits-on-vertex-ai)

---

## Cloud Logging Issues

### 403 Permission Denied with Resource Limits on Vertex AI

#### Problem

When using resource limit methods like `set_memory_limit()`, `set_cpu_limit()`, `set_cpu_request()`, or `set_memory_request()` with components that use the `google-cloud-logging` Python client, you may encounter a permission error:

```
Failed to submit 1 logs. 
google. api_core.exceptions. PermissionDenied: 403 Permission 'logging.logEntries.create' 
denied on resource (or it may not exist).
[reason: "IAM_PERMISSION_DENIED"
domain: "iam.googleapis.com"
metadata {
  key: "permission"
  value: "logging. logEntries.create"
}
```

This is a known issue when running pipelines on Vertex AI Pipelines.  See [#12554](https://github.com/kubeflow/pipelines/issues/12554) and [#11302](https://github.com/kubeflow/pipelines/issues/11302) for details.

#### Root Cause

When resource specifications are present in a component, Vertex AI may create pods with a restricted execution context that lacks permissions for the Cloud Logging API.  This appears to be a platform-level behavior in Vertex AI rather than a Kubeflow Pipelines issue.

#### Solution

Use standard Python logging instead of the `google-cloud-logging` client. Vertex AI automatically captures container stdout and stderr and forwards them to Cloud Logging, so direct API access is not necessary.

**Before (fails with resource limits):**

```python
from kfp import dsl

@dsl.component(
    base_image="python:3.11",
    packages_to_install=["google-cloud-logging"]  # Not needed
)
def my_component():
    import logging
    from google.cloud import logging as cloud_logging
    
    # This requires logging.logEntries.create permission
    client = cloud_logging.Client()
    client.setup_logging()
    
    logging.info("This may fail with 403 when using set_memory_limit()")

@dsl.pipeline(name="my-pipeline")
def my_pipeline():
    task = my_component()
    task.set_memory_limit("8G")  # Triggers 403 error
```

**After (works with resource limits):**

```python
from kfp import dsl

@dsl.component(base_image="python:3.11")  # No extra packages needed
def my_component():
    import logging
    
    # Standard logging writes to stdout, which Vertex AI captures automatically
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    logging.info("This works correctly with set_memory_limit()")
    logging.warning("All log levels work")
    logging.error("Errors are captured too")

@dsl.pipeline(name="my-pipeline")
def my_pipeline():
    task = my_component()
    task.set_memory_limit("8G")  # Works correctly
```

#### Why This Works

Vertex AI Pipelines automatically configures log collection for pipeline containers: 

1. Your component writes logs to stdout/stderr using standard Python logging
2. The Kubernetes pod captures stdout/stderr streams
3. Vertex AI forwards these logs to Cloud Logging automatically
4. Logs appear in Cloud Logging under your pipeline's execution

This approach bypasses the need for direct Cloud Logging API calls, avoiding the permission issue entirely.

#### Viewing Logs

Your logs will appear in Cloud Logging: 

1. Go to [Cloud Logging](https://console.cloud.google.com/logs)
2. Filter by:
   - **Resource type:** Cloud Composer Environment (for Vertex AI Pipelines)
   - **Log name:** `stdout` or `stderr`
   - **Labels:** Pipeline name, component name

You can also view logs directly in the Vertex AI Pipelines UI by clicking on individual pipeline steps.

#### Trade-offs

| Feature | `google-cloud-logging` Client | Standard Python Logging |
|---------|------------------------------|------------------------|
| Works with resource limits | ❌ No (403 error) | ✅ Yes |
| Setup complexity | High (client setup, auth) | Low (built-in) |
| Dependencies | Requires `google-cloud-logging` | None (stdlib) |
| Structured logging | Advanced features | Basic (but sufficient) |
| Custom log names | ✅ Yes | ❌ No (uses stdout) |
| Performance | Slower (API calls) | Faster (local writes) |

For most pipeline use cases, standard Python logging provides sufficient functionality while ensuring compatibility with resource limits on Vertex AI.

#### Structured Logging Alternative

If you need structured logging, you can use JSON formatting with standard logging:

```python
import logging
import json

@dsl.component(base_image="python:3.11")
def my_component():
    logging.basicConfig(level=logging.INFO)
    
    # Log structured data as JSON
    log_data = {
        "event":  "model_training_complete",
        "accuracy": 0.95,
        "epochs": 100,
        "model_size_mb": 250
    }
    
    logging.info(json.dumps(log_data))
```

In Cloud Logging, you can parse these JSON logs using log queries and create custom metrics or alerts.

#### Additional Notes

- This issue is specific to **Vertex AI Pipelines**.  Other platforms (e.g., Kubeflow on GKE) may not be affected.
- The issue occurs regardless of the IAM roles assigned to your service account.
- Both `set_memory_limit()` and other resource methods (`set_cpu_limit()`, `set_memory_request()`, `set_cpu_request()`) trigger this behavior.

#### Related Issues

- [#12554](https://github.com/kubeflow/pipelines/issues/12554) - Current issue report and investigation
- [#11302](https://github.com/kubeflow/pipelines/issues/11302) - Previous related report

---

## Other Issues

*More troubleshooting sections can be added here as needed.*