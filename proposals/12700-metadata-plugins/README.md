# KEP-12xyz: Pipeline Metadata Plugins

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
- [Design Details](#design-details)
  - [Architecture Overview](#architecture-overview)
  - [Plugin Base Class](#plugin-base-class)
  - [Plugin Lifecycle Hooks](#plugin-lifecycle-hooks)
    - [Run Hooks (API Server)](#run-hooks-api-server)
    - [Task Hooks (Driver)](#task-hooks-driver)
    - [Executor Hooks (Launcher)](#executor-hooks-launcher)
  - [Plugin Input Fields](#plugin-input-fields)
  - [Plugin Server](#plugin-server)
    - [Deployment](#deployment)
    - [API Endpoints](#api-endpoints)
    - [Authorization](#authorization)
  - [API Server Configuration](#api-server-configuration)
  - [Out-of-Box Plugins](#out-of-box-plugins)
  - [API Changes](#api-changes)
- [Frontend Considerations](#frontend-considerations)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation Steps](#implementation-steps)
- [Test Plan](#test-plan)
- [Upgrade Considerations](#upgrade-considerations)

## Summary

This proposal introduces a plugin architecture for KFP that allows external systems to hook into pipeline run lifecycle events. Plugins can collect user inputs at run creation, validate those inputs, and respond to run/task/executor state changes.

> **Reference Implementation:** See the `src/` folder for example code including the plugin server, base class definitions, and sample plugins.

## Motivation

Pipeline runs often require integration with external systems for:
- Metadata tracking (metrics, inputs, outputs, lineage)
- Notifications (run state transitions, failures)
- Custom validation (environment-specific requirements)

Rather than implementing these integrations directly in KFP, this proposal provides a standardized plugin interface that allows developers to extend KFP without modifying core code.

## Design Details

### Architecture Overview

```
┌─────────────┐     ┌─────────────┐     ┌───────────────┐
│   Frontend  │────▶│  API Server │────▶│ Plugin Server │
└─────────────┘     └─────────────┘     └───────────────┘
                           │                    │
                           ▼                    │
                    ┌─────────────┐             │
                    │   Driver    │─────────────┤
                    └─────────────┘             │
                           │                    │
                           ▼                    │
                    ┌─────────────┐             │
                    │  Launcher   │─────────────┘
                    └─────────────┘
```

- **API Server**: Invokes run-level hooks (`on_run_start`, `on_run_end`)
- **Driver**: Invokes task-level hooks (`on_task_start`, `on_task_end`)
- **Launcher**: Invokes executor-level hooks (`on_executor_start`)

### Plugin Base Class

Plugins implement the `KfpLifecyclePlugin` abstract class, which defines hooks called during the pipeline lifecycle:

```python
class KfpLifecyclePlugin(ABC):
    @property
    def name(self) -> str:
        """Unique plugin name."""
        return self.__class__.__name__

    def get_input_fields(self) -> InputFieldsResponse | None:
        """Define custom input fields for the run creation UI."""
        return None

    def validate_inputs(self, inputs: Dict[str, Any]) -> ValidationResult:
        """Validate user-provided input values."""
        return ValidationResult(valid=True)

    def on_run_start(self, req: RunHookRequest) -> RunHookResponse | None:
        """Called when a pipeline run starts."""
        return None

    def on_run_end(self, req: RunHookRequest) -> RunHookResponse | None:
        """Called when a pipeline run completes."""
        return None

    def on_task_start(self, req: TaskHookRequest) -> TaskStartResponse | None:
        """Called when a task starts. Can return pod_spec_patch."""
        return None

    def on_task_end(self, req: TaskHookRequest) -> TaskEndResponse | None:
        """Called when a task completes."""
        return None

    def on_executor_start(self, req: ExecutorHookRequest) -> ExecutorStartResponse | None:
        """Called when executor starts. Can return pre/post execution code."""
        return None
```

### Plugin Lifecycle Hooks

#### Run Hooks (API Server)

The API server invokes run-level hooks at run creation and completion.

| Hook           | When Called                        | Response                  |
|----------------|------------------------------------|---------------------------|
| `on_run_start` | Run is created                     | Metadata to attach to run |
| `on_run_end`   | Run completes (success or failure) | Metadata to attach to run |

**Implementation** (`backend/src/apiserver/`):
- Call `/v1/hooks/on_run_start` after creating the run in the database
- Call `/v1/hooks/on_run_end` when run state transitions to a terminal state

#### Task Hooks (Driver)

The driver invokes task-level hooks before and after task execution.

| Hook            | When Called                  | Response                   |
|-----------------|------------------------------|----------------------------|
| `on_task_start` | Before creating executor pod | `pod_spec_patch`, metadata |
| `on_task_end`   | After task completes         | Metadata to attach to task |

**Implementation** (`backend/src/v2/driver/driver.go`):
- Call `/v1/hooks/on_task_start` with run and task context before creating the executor pod
- Strategically merge returned `pod_spec_patch` into the executor pod spec

**Plugin Author Responsibility**: Patches are merged in plugin registration order. Authors must ensure their patches are additive and don't conflict with other plugins (e.g., append to lists rather than replace).

#### Executor Hooks (Launcher)

The launcher invokes executor-level hooks to enable code wrapping around user function execution.

| Hook                | When Called                 | Response                                    |
|---------------------|-----------------------------|---------------------------------------------|
| `on_executor_start` | Before running user command | `pre_execution_code`, `post_execution_code` |

This enables use cases like:
- Setting up experiment tracking context
- Wrapping user code with try/catch for custom error handling
- Collecting metrics or artifacts after execution

**ExecutorStartResponse Fields:**

```python
class ExecutorStartResponse(BaseModel):
    pod_spec_patch: V1PodSpec | None = None
    pre_execution_code: str | None = None   # Runs BEFORE user function
    post_execution_code: str | None = None  # Runs AFTER user function
```

**Implementation:**

**Launcher** (`backend/src/v2/component/launcher_v2.go`):
- Before `exec.Command()`, call plugin server's `/v1/hooks/on_executor_start`
- Set environment variables `KFP_PRE_EXEC_CODE` and `KFP_POST_EXEC_CODE` with response values

**SDK Executor** (`sdk/python/kfp/dsl/executor.py`):
- In `execute()` method, read env vars and wrap user function call:

```python
pre_code = os.environ.get('KFP_PRE_EXEC_CODE')
post_code = os.environ.get('KFP_POST_EXEC_CODE')

if pre_code:
    exec(pre_code, {'func_kwargs': func_kwargs, 'executor_input': self.executor_input})

result = self.func(**func_kwargs)

if post_code:
    exec(post_code, {'result': result, 'func_kwargs': func_kwargs})
```

### Plugin Input Fields

Plugins can define custom input fields that appear on the run creation page.

#### Flow

1. Frontend calls `GET /v1/hooks/input_fields` to retrieve field definitions
2. User fills out form and submits
3. Frontend calls `POST /v1/hooks/validate_inputs` with field values
4. If validation fails, run creation is blocked with error messages
5. On success, validated inputs are passed to `on_run_start` via `RunHookRequest.plugin_inputs`

#### Defining Input Fields

```python
def get_input_fields(self) -> InputFieldsResponse:
    return InputFieldsResponse(
        group_label="My Plugin Settings",
        order=10,  # Lower values appear first
        fields=[
            InputFieldDefinition(
                field_id="experiment_name",
                label="Experiment Name",
                field_type="text",
                required=True,
                description="Name of the experiment",
            ),
            InputFieldDefinition(
                field_id="log_level",
                label="Log Level",
                field_type="select",
                options=["DEBUG", "INFO", "WARNING", "ERROR"],
                default_value="INFO",
            ),
        ]
    )
```

#### Validating Inputs

```python
def validate_inputs(self, inputs: Dict[str, Any]) -> ValidationResult:
    errors = []
    if not inputs.get("experiment_name"):
        errors.append(ValidationError(
            field_id="experiment_name",
            message="Experiment name is required"
        ))
    return ValidationResult(valid=len(errors) == 0, errors=errors)
```

#### Using Inputs in Hooks

```python
def on_run_start(self, req: RunHookRequest) -> RunHookResponse:
    exp_name = req.plugin_inputs.get("experiment_name")
    log_level = req.plugin_inputs.get("log_level", "INFO")
    # Use the inputs...
```

### Plugin Server

The plugin server is a standalone deployment that hosts one or more plugins. KFP components communicate with the plugin server via HTTP to invoke plugin hooks.

Plugin authors package their plugins into a container image that includes:
- The KFP SDK
- The plugin implementation(s)
- Any plugin-specific dependencies

#### Deployment

1. **Build plugin server image** - Package plugins into a container image with the KFP SDK
2. **Deploy plugin server** - Create a Deployment and Service in the same namespace as the API server
3. **Configure API server** - Set `PluginConfig.Enabled=true` and `PluginConfig.Endpoint` to the plugin server Service URL
4. **Restart API server** - Apply the configuration change

#### API Endpoints

**GET /v1/hooks/input_fields**

Returns field definitions from all plugins:

```json
{
  "plugins": {
    "example-plugin": {
      "group_label": "Example Plugin Settings",
      "order": 10,
      "fields": [
        {"field_id": "experiment_name", "label": "Experiment Name", "...": "..."}
      ]
    }
  }
}
```

**POST /v1/hooks/validate_inputs**

Request:
```json
{
  "inputs": {
    "example-plugin": {
      "experiment_name": "my-experiment",
      "log_level": "INFO"
    }
  }
}
```

Response:
```json
{
  "valid": false,
  "results": {
    "example-plugin": {
      "valid": false,
      "errors": [
        {"field_id": "experiment_name", "message": "Experiment name is required"}
      ]
    }
  }
}
```

**Lifecycle Hook Endpoints:**

- `POST /v1/hooks/on_run_start`
- `POST /v1/hooks/on_run_end`
- `POST /v1/hooks/on_task_start`
- `POST /v1/hooks/on_task_end`
- `POST /v1/hooks/on_executor_start`

Each endpoint accepts the appropriate request type and returns aggregated results from all plugins.

#### Authorization

The plugin server validates incoming requests using Kubernetes RBAC. Callers must present a token with the following permission:

| Resource | Group                    | Verb     |
|----------|--------------------------|----------|
| `runs`   | `pipelines.kubeflow.org` | `update` |

The plugin server performs a `SubjectAccessReview` to verify the caller's token has this permission before processing requests. The API server ServiceAccount already has this permission, ensuring only authorized components can invoke plugin hooks.

### API Server Configuration

Administrators can configure multiple plugin servers. Each server is invoked in order, and results are aggregated across all servers.

```json
{
  "PluginServers": [
    {
      "Name": "mlflow-plugin",
      "Endpoint": "http://mlflow-plugin-server:8080",
      "Timeout": "30s",
      "TLS": {
        "Enabled": false,
        "CABundlePath": "",
        "SkipVerify": false
      }
    },
    {
      "Name": "notifications-plugin",
      "Endpoint": "http://notifications-plugin-server:8080",
      "Timeout": "10s"
    }
  ]
}
```

| Field              | Type   | Description                      | Default  |
|--------------------|--------|----------------------------------|----------|
| `Name`             | string | Identifier for the plugin server | Required |
| `Endpoint`         | string | Plugin server URL                | Required |
| `Timeout`          | string | HTTP request timeout             | `"30s"`  |
| `TLS.Enabled`      | bool   | Use TLS for connection           | `false`  |
| `TLS.CABundlePath` | string | Path to CA bundle file           | `""`     |
| `TLS.SkipVerify`   | bool   | Skip TLS verification            | `false`  |

**Hook Invocation**: When a hook is triggered, the API server calls each configured plugin server in order. Responses are aggregated:
- **Metadata**: Keys are automatically prefixed with the plugin name (e.g., `mlflow.experiment_id`) to prevent conflicts
- **Pod patches**: Applied sequentially in server order
- **Unavailable servers**: Skipped with a warning logged

**API Server Proxy Endpoints:**

The frontend communicates with plugins through the API server, which aggregates responses from all configured plugin servers:

- `GET /apis/v2beta1/plugins/input_fields` → Aggregates `GET /v1/hooks/input_fields` from all servers
- `POST /apis/v2beta1/plugins/validate_inputs` → Aggregates `POST /v1/hooks/validate_inputs` from all servers
- `GET /apis/v2beta1/plugins/status` → Aggregates `GET /v1/plugins` from all servers

### Out-of-Box Plugins

KFP ships with optional plugins that implement common integrations. These are disabled by default and loaded conditionally based on configuration.

**Entry Point Groups:**

| Group                 | Purpose                    | Loading                     |
|-----------------------|----------------------------|-----------------------------|
| `kfp.sdk.plugins`     | Custom/third-party plugins | Always loaded               |
| `kfp.sdk.plugins.oob` | KFP-shipped plugins        | Loaded if enabled in config |

**Enable OOB plugins via environment variable:**

```bash
# Enable specific plugins
OOB_PLUGINS=mlflow,slack-notification

# Enable all OOB plugins
OOB_PLUGINS=all

# Disable all OOB plugins (default)
OOB_PLUGINS=none
```

**Available OOB Plugins:**

| Plugin               | Description                            |
|----------------------|----------------------------------------|
| `mlflow`             | MLflow experiment tracking integration |
| `slack-notification` | Slack notifications for run events     |

OOB plugins use the same `KfpLifecyclePlugin` interface as custom plugins. They are discovered via the `kfp.sdk.plugins.oob` entry point group:

```python
# In sdk/python/setup.py
setuptools.setup(
    entry_points={
        'kfp.sdk.plugins.oob': [
            'mlflow = kfp.plugins.oob.mlflow:create_plugin',
            'slack-notification = kfp.plugins.oob.slack_notification:create_plugin',
        ],
    }
)
```

### API Changes

The `Run` and `PipelineTaskDetail` messages gain a `metadata` field for plugin-provided metadata:

```protobuf
message MetadataValue {
  enum Type {
    TEXT = 0;
    URL = 1;
  }
  string value = 1;
  Type type = 2;  // Hint for UI rendering
}

message Metadata {
  map<string, MetadataValue> entries = 1;
}
```

## Frontend Considerations

### Run Creation Page

- Fetch plugin input fields via `GET /apis/v2beta1/plugins/input_fields` when the page loads
- Group fields by plugin using `group_label`, ordered by plugin `order` then field `order`
- Render appropriate input controls based on `field_type` (text, number, select, checkbox, textarea)
- On submit, call `POST /apis/v2beta1/plugins/validate_inputs` before creating the run
- Display validation errors inline next to the corresponding fields

### Run/Task Details

- Display plugin metadata in a dedicated section on run and task detail pages
- Render values based on `type`: URL values as hyperlinks, TEXT as plain text
- Use the metadata key as the label (e.g., `example.tracking_url` → "Tracking URL")

### Error Handling

- If plugin server is unavailable, run creation should still work (plugins are optional)
- Show a warning if plugin fields cannot be loaded

## Drawbacks

- **Latency**: Driver and Launcher call plugin servers in the critical path of task execution. Latency grows with the number of configured servers.
- **Code Injection Security**: Using `exec()` for pre/post execution code requires trusting the plugin server. A compromised server could execute arbitrary code in user pipelines. Future work may explore sandboxing alternatives.

## Alternatives

### Bake Extensions into KFP Core

Continue implementing integrations (metadata tracking, notifications, etc.) directly in KFP.

**Pros:**
- No additional deployment complexity
- Tighter integration with core APIs
- Single codebase to maintain

**Cons:**
- Every integration requires changes to KFP core
- Increases maintenance burden on KFP maintainers
- Slower iteration cycle (tied to KFP release schedule)
- Difficult for organizations to add custom integrations without forking

The plugin approach trades deployment complexity for flexibility and separation of concerns.

## Implementation Steps

1. Add plugin SDK to KFP (`kfp/plugins/` with api, loader, server modules)
2. Add OOB plugins package and register entry points in `setup.py`
3. Add `PluginConfig` to API server configuration
4. Implement API server proxy endpoints (`/apis/v2beta1/plugins/*`)
5. Add plugin client in API server to call plugin server hooks
6. Invoke run hooks at run create/complete in API server
7. Invoke task hooks at task start/complete in driver
8. Invoke executor hooks at executor start in launcher
9. Update frontend run creation page to fetch and render plugin input fields

## Test Plan

**Unit Tests:**
- Plugin loader: entry point discovery, OOB_PLUGINS parsing, plugin filtering
- Plugin server: HTTP endpoint handlers, hook aggregation across plugins
- API server plugin client: request/response handling, timeout and error cases

**Integration Tests:**
- API server with plugin server: proxy endpoints, run lifecycle hook invocation
- Driver with plugin server: task hooks and pod spec patching
- Launcher with plugin server: executor hooks and code injection

**E2E Tests:**
- Full pipeline run with plugins enabled: input fields render, validation works, hooks fire at each lifecycle stage, metadata appears in UI

## Upgrade Considerations

This is an additive change with no breaking changes to existing APIs. No upgrade strategy is required.

- Plugin server must be deployed alongside the API server to use plugins
- Pipelines compiled with older SDK versions will still work with plugins enabled
- Plugins are optional; if the plugin server is unavailable, KFP continues to function normally
