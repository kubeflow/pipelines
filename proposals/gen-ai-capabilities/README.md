# KEP-XXXXX: Gen-AI Capabilities for Kubeflow Pipelines

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Natural Language Pipeline Authoring](#story-1-natural-language-pipeline-authoring)
    - [Story 2: Intelligent Pipeline Debugging](#story-2-intelligent-pipeline-debugging)
    - [Story 3: Pipeline Optimization Suggestions](#story-3-pipeline-optimization-suggestions)
    - [Story 4: Auto-Generated Documentation](#story-4-auto-generated-documentation)
    - [Story 5: Conversational Pipeline Exploration](#story-5-conversational-pipeline-exploration)
    - [Story 6: Agent-Driven Multi-Step Operations](#story-6-agent-driven-multi-step-operations)
    - [Story 7: External Tool Integration via MCP](#story-7-external-tool-integration-via-mcp)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Architecture Overview](#architecture-overview)
  - [Agentic Tool System](#agentic-tool-system)
    - [Chat Modes](#chat-modes)
    - [Prompt Rules Engine](#prompt-rules-engine)
    - [MCP Integration](#mcp-integration)
  - [LLM Integration Layer](#llm-integration-layer)
    - [Provider Abstraction](#provider-abstraction)
    - [Multi-Model Routing](#multi-model-routing)
    - [Configuration](#configuration)
  - [Backend Changes](#backend-changes)
    - [API Server Extensions](#api-server-extensions)
    - [Proto Definitions](#proto-definitions)
  - [Frontend Changes](#frontend-changes)
    - [AI Assistant Panel](#ai-assistant-panel)
    - [Inline AI Features](#inline-ai-features)
  - [SDK Extensions](#sdk-extensions)
  - [Security and Privacy](#security-and-privacy)
  - [Observability, Metrics, and Audit Logging](#observability-metrics-and-audit-logging)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [E2E Tests](#e2e-tests)
    - [Security Tests](#security-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Migration Strategy](#migration-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [External Plugin Architecture](#external-plugin-architecture)
  - [SDK-Only Approach](#sdk-only-approach)
  - [Fine-Tuned Domain Model](#fine-tuned-domain-model)
<!-- /toc -->

## Summary

This proposal introduces Gen-AI capabilities into the Kubeflow Pipelines UI and backend, enabling
users to interact with their ML pipelines through natural language. The feature set includes natural
language pipeline authoring, intelligent debugging and log analysis, pipeline optimization
suggestions, auto-generated documentation, and conversational pipeline exploration. The
implementation adds an LLM integration layer to the API server with a pluggable provider abstraction
(supporting OpenAI-compatible APIs, Google Vertex AI, Amazon Bedrock, and self-hosted models), and
a new AI assistant panel in the frontend. The feature is entirely opt-in, disabled by default, and
requires explicit administrator configuration to enable.

## Motivation

Kubeflow Pipelines is a powerful platform for building and deploying ML workflows, but it has a
steep learning curve. Users must understand the KFP SDK, pipeline DSL, Kubernetes concepts, and
Argo Workflows to effectively build, debug, and optimize pipelines. This creates friction for:

- **Data scientists** who are proficient in ML but not infrastructure or pipeline DSLs.
- **New adopters** who struggle to translate their ML workflows into KFP pipelines.
- **Operations teams** who need to quickly diagnose pipeline failures across hundreds of runs.
- **Organizations** that want to improve developer productivity and reduce time-to-production for
  ML models.

Gen-AI capabilities can significantly reduce this friction by allowing users to describe intent in
natural language and receive actionable pipeline code, debugging insights, and optimization
recommendations. Similar capabilities have proven transformative in adjacent tools (e.g., GitHub
Copilot for code, Jupyter AI for notebooks).

### Goals

1. Enable users to describe pipeline logic in natural language and receive valid KFP Python SDK
   code that can be directly compiled and run.
2. Provide AI-powered analysis of pipeline run failures, including log summarization, root cause
   identification, and suggested fixes.
3. Offer pipeline optimization suggestions based on resource utilization, execution patterns, and
   best practices.
4. Auto-generate human-readable documentation for pipelines, components, and their parameters.
5. Support conversational exploration of pipeline history, run metadata, and experiment results.
6. Maintain a pluggable LLM provider abstraction so organizations can use their preferred model
   provider (cloud APIs or self-hosted).
7. Ensure the feature is fully opt-in and disabled by default, with no impact on existing
   deployments.
8. Respect data privacy by allowing administrators to configure what context (logs, parameters,
   artifacts) is sent to the LLM provider.

### Non-Goals

1. Training or fine-tuning a Kubeflow-specific LLM. This proposal uses general-purpose LLMs with
   prompt engineering and retrieval-augmented generation (RAG).
2. Autonomous pipeline execution. The AI assists and suggests; the user always confirms and
   triggers execution.
3. Replacing the existing SDK or UI authoring experience. Gen-AI capabilities augment the existing
   workflows.
4. Supporting real-time streaming inference pipelines. This proposal focuses on batch/training
   pipeline workflows.
5. Building a general-purpose chatbot. The AI assistant is scoped to Kubeflow Pipelines operations.

## Proposal

Add a Gen-AI integration layer to the Kubeflow Pipelines backend and a corresponding AI assistant
interface in the frontend. The integration layer is a new optional service within the API server
that communicates with configurable LLM providers through a provider abstraction. The frontend
gains an AI assistant panel that users can invoke for natural language interactions.

The feature is configured through the existing `pipeline-install-config` ConfigMap and is disabled
by default. When enabled, the API server exposes new gRPC/REST endpoints for AI-assisted
operations that the frontend consumes.

### User Stories

#### Story 1: Natural Language Pipeline Authoring

As a data scientist, I want to describe my ML workflow in plain English (e.g., "Create a pipeline
that loads data from BigQuery, preprocesses it with pandas, trains an XGBoost model, evaluates it,
and deploys to a KServe endpoint if accuracy > 0.95") and receive valid KFP Python SDK code that I
can review, edit, and run.

#### Story 2: Intelligent Pipeline Debugging

As an ML engineer, I want to click on a failed pipeline run and ask "Why did this fail?" to receive
a summary of the error, the likely root cause (e.g., OOM on the training step, missing input
artifact, permission denied on the storage bucket), and suggested fixes (e.g., increase memory
request to 8Gi, check IAM bindings).

#### Story 3: Pipeline Optimization Suggestions

As a platform engineer, I want to select a pipeline and ask "How can I optimize this?" to receive
recommendations such as: reducing over-provisioned resource requests based on historical usage,
adding caching to deterministic steps, parallelizing independent components, or switching to a
more cost-effective accelerator type.

#### Story 4: Auto-Generated Documentation

As a team lead, I want to select a pipeline version and click "Generate documentation" to produce
a human-readable description of what the pipeline does, its inputs/outputs, component
dependencies, and expected resource requirements, formatted as Markdown that can be stored
alongside the pipeline code.

#### Story 5: Conversational Pipeline Exploration

As a data scientist, I want to ask questions like "Show me all runs from last week that used the
v2 preprocessing component and had accuracy above 0.9" and receive a filtered view of matching
runs without needing to manually construct filters in the UI.

#### Story 6: Agent-Driven Multi-Step Operations

As an ML engineer, I want to ask "Rerun the failed preprocessing step with 16Gi memory and notify
me when it completes" and have the AI break this into sequential tool calls (fetch run → identify
failed step → create new run with modified resources → set up notification), presenting each
mutating action for my confirmation before executing it.

#### Story 7: External Tool Integration via MCP

As a platform engineer, I want the AI assistant to query our internal model registry and data
catalog when answering questions about pipelines, so that it can provide context-aware
recommendations like "The v3 model in the registry outperforms v2 by 5% — consider updating this
pipeline's model reference."

### Notes/Constraints/Caveats

- **LLM availability**: The feature requires network access to an LLM endpoint (cloud API or
  self-hosted). Air-gapped environments must deploy a self-hosted model.
- **Cost**: Cloud LLM API calls incur per-token costs. The implementation should support token
  budgets and rate limiting to prevent unexpected charges.
- **Latency**: LLM responses may take several seconds. The UI must handle this gracefully with
  streaming responses and progress indicators.
- **Accuracy**: LLM-generated pipeline code may contain errors. The proposal emphasizes a
  human-in-the-loop approach where all generated code is presented for review, not auto-executed.
- **Context window**: Pipeline specs, logs, and metadata may exceed LLM context windows. The
  implementation must use intelligent context selection and summarization.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM generates incorrect pipeline code | Users run broken pipelines | Generated code is always presented for review; never auto-executed. Include a validation step that compiles the pipeline before presenting it. |
| Sensitive data sent to external LLM provider | Data leakage, compliance violations | Admin-configurable data filtering controls. Support self-hosted models. Redact secrets and credentials from context by default. |
| High cloud API costs from LLM usage | Unexpected billing | Configurable token budgets, rate limiting per user/namespace, and cost estimation displayed before expensive operations. |
| LLM provider outage | Feature unavailable | Graceful degradation — all existing UI/SDK functionality works without the AI features. Clear error messaging when AI is unavailable. |
| Prompt injection via pipeline names or parameters | Unauthorized LLM behavior | Sanitize all user-provided content before including in prompts. Use structured prompts with clear system/user boundaries. |
| Model hallucination of non-existent KFP APIs | User confusion | Include KFP SDK documentation as RAG context. Validate generated code against the installed SDK version. |

## Design Details

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubeflow Pipelines UI                    │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ AI Assistant  │  │ Inline AI    │  │ Existing UI       │  │
│  │ Panel         │  │ Features     │  │ (unchanged)       │  │
│  └──────┬───────┘  └──────┬───────┘  └───────────────────┘  │
└─────────┼──────────────────┼────────────────────────────────┘
          │                  │
          ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                   KFP API Server                            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              AI Service (new, optional)               │   │
│  │  ┌─────────────┐  ┌──────────┐  ┌────────────────┐  │   │
│  │  │ Prompt       │  │ Context  │  │ Response       │  │   │
│  │  │ Templates    │  │ Builder  │  │ Processor      │  │   │
│  │  └─────────────┘  └──────────┘  └────────────────┘  │   │
│  │  ┌─────────────────────────────────────────────────┐ │   │
│  │  │         LLM Provider Abstraction                │ │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌─────────┐  │ │   │
│  │  │  │OpenAI  │ │Vertex  │ │Bedrock │ │Self-    │  │ │   │
│  │  │  │compat. │ │AI      │ │        │ │hosted   │  │ │   │
│  │  │  └────────┘ └────────┘ └────────┘ └─────────┘  │ │   │
│  │  └─────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         Existing API Server (unchanged)               │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Agentic Tool System

Rather than implementing fixed RPCs for every predefined use case, the design centers on an
**agentic chat endpoint** where the LLM accesses a toolkit of KFP operations and composes them
dynamically. This approach enables flexible tool composition without requiring new backend
endpoints for each new capability.

The agent maintains a registry of tools:

- **Built-in KFP tools**: List runs, fetch logs, get pipeline specs, compile pipelines, submit
  runs, create experiments, etc.
- **MCP external tools**: Tools exposed by external MCP servers (model registries, data catalogs,
  monitoring systems).
- **Custom extension tools**: Platform-team-registered tools that the agent can invoke, wrapped
  with confirmation requirements.

Each tool declares whether it is **read-only** (auto-approved) or **mutating** (requires explicit
user confirmation before execution). The agent plans a sequence of tool calls, executes read-only
tools automatically, and pauses for user approval before any state-changing operation.

#### Chat Modes

The assistant supports two interaction modes:

- **Ask mode** (default): Read-only exploration. The agent can invoke read-only tools (list runs,
  fetch logs, get pipeline specs) but cannot perform any mutating operations. Safe for discovering
  pipeline information and getting recommendations.
- **Agent mode**: Full tool access with confirmation gates. The agent can plan and execute
  multi-step operations that include state-changing actions (submit runs, create experiments,
  modify pipeline parameters), but every mutating tool call requires explicit user confirmation
  before execution.

Users select the mode via a toggle in the AI assistant panel. Administrators can restrict which
modes are available per namespace via the prompt rules engine.

#### Prompt Rules Engine

Administrators inject organization-specific guidelines into system prompts without model
fine-tuning. Rules are defined as Markdown files with YAML frontmatter and support:

- **Scope matching**: Rules can be scoped to specific namespaces, chat modes, or pipeline name
  patterns.
- **Priority ordering**: Rules are applied in priority order, allowing organization-wide defaults
  with namespace-specific overrides.
- **Dynamic injection**: Rules are injected into the system prompt at request time, ensuring they
  are always up-to-date without requiring model redeployment.

Example rule file:

```markdown
---
scope:
  namespaces: ["production", "staging"]
  modes: ["agent"]
priority: 100
---
# Production Safety Rules

- NEVER suggest deleting or modifying runs in the production namespace without explicit mention of
  a rollback plan.
- Always recommend canary deployments for model updates.
- Flag any pipeline that does not include a model validation step.
```

Rules are stored in a ConfigMap or mounted volume and referenced via the `aiRulesPath`
configuration key.

#### MCP Integration

KFP integrates with the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) in two
directions:

- **KFP as MCP server**: Exposes KFP operations (list pipelines, get run status, fetch logs,
  compile pipelines) as MCP tools, enabling external AI clients (e.g., Claude Code, Cursor,
  custom agents) to interact with KFP programmatically.
- **KFP as MCP client**: Connects to external MCP servers (model registries, data catalogs,
  monitoring systems) whose tools become available to the AI agent during conversations. This
  enables cross-system queries like "Which model in the registry has the best F1 score for this
  dataset?" without building custom integrations.

MCP server endpoints are configured in the `pipeline-install-config` ConfigMap:

```yaml
aiMCPServers: |
  - name: model-registry
    url: http://model-registry.ml-platform:8080/mcp
  - name: data-catalog
    url: http://data-catalog.ml-platform:8080/mcp
```

### LLM Integration Layer

#### Provider Abstraction

A pluggable provider interface allows organizations to use their preferred LLM:

```go
// LLMProvider defines the interface for LLM backends.
type LLMProvider interface {
    // ChatCompletion sends a prompt and returns a response.
    ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    // StreamChatCompletion sends a prompt and streams the response.
    StreamChatCompletion(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
    // Name returns the provider name for logging and metrics.
    Name() string
}

type ChatRequest struct {
    SystemPrompt string
    Messages     []Message
    MaxTokens    int
    Temperature  float32
}

type Message struct {
    Role    string // "user", "assistant", "system"
    Content string
}

// ChatChunk represents a single chunk in a streaming response.
type ChatChunk struct {
    Content string // Full content accumulated so far
    Delta   string // Incremental text in this chunk
    Done    bool   // Whether this is the final chunk
}
```

**Streaming error handling**: When errors occur during streaming, the provider sends a final
`ChatChunk` with `Done: true` and returns the error from the channel. The caller should select on
both the chunk channel and a context cancellation to handle timeouts and client disconnects
gracefully. Provider implementations must close the channel when the stream ends (whether
successfully or due to error) to prevent goroutine leaks.

Built-in provider implementations:

- **OpenAI-compatible**: Works with OpenAI, Azure OpenAI, vLLM, Ollama, and any OpenAI-compatible
  API endpoint.
- **Google Vertex AI**: Native integration with Gemini models via the Vertex AI API.
- **Amazon Bedrock**: Native integration with models available through Bedrock.
- **Self-hosted**: Generic HTTP/gRPC interface for self-hosted model servers.

#### Multi-Model Routing

The design separates **providers** (service endpoints) from **models** (specific capabilities),
allowing organizations to route different tasks to different models for cost optimization:

- **Chat models**: Used for complex reasoning, debugging, code generation, and conversational
  interactions. Typically a larger, more capable model (e.g., GPT-4o, Claude Sonnet, Gemini Pro).
- **Embedding models**: Used for vector-based pipeline similarity search, enabling features like
  "find pipelines similar to this one" or semantic search over pipeline descriptions and
  documentation. Typically a smaller, specialized embedding model (e.g., text-embedding-3-small,
  Vertex AI text-embedding).

Embedding models are optional. When configured, they enable similarity-based features; when
omitted, the system falls back to keyword-based search. Embedding models integrate with chat
models by providing relevant pipeline context retrieved via vector similarity before the chat
model generates a response (retrieval-augmented generation).

Model assignments are configured separately so organizations can use an expensive model for
chat (where quality matters most) and a cheaper model for embeddings (where throughput matters):

```yaml
aiModel: "gpt-4o"                    # Chat model
aiEmbeddingModel: "text-embedding-3-small"  # Embedding model (optional)
```

#### Configuration

Configuration via `pipeline-install-config` ConfigMap:

```yaml
## Gen-AI configuration. Feature is disabled when aiProvider is empty (default).
aiProvider: ""              # "openai", "vertexai", "bedrock", "custom"
aiEndpoint: ""              # LLM API endpoint URL
aiModel: ""                 # Chat model identifier (e.g., "gpt-4o", "gemini-2.0-flash")
aiEmbeddingModel: ""        # Embedding model identifier (optional, e.g., "text-embedding-3-small")
## Integer values stored as strings per ConfigMap convention. Parsed as int32 by the API server.
## Invalid values (negative, non-numeric) cause the AI service to fail initialization with a
## descriptive error logged. Omitted keys use the documented defaults.
aiMaxTokensPerRequest: "4096"   # Range: 1–128000. Default: 4096.
aiRateLimitPerMinute: "30"      # Range: 1–1000. Default: 30. Per-user rate limit.
## Controls what pipeline context is sent to the LLM. Comma-separated.
## Options: "logs", "parameters", "metrics", "artifacts", "pipeline-spec"
aiAllowedContext: "logs,parameters,metrics,pipeline-spec"
## Path to prompt rules directory (optional). Rules are Markdown files with YAML frontmatter.
aiRulesPath: ""             # e.g., "/etc/kfp/ai-rules/"
## MCP server configuration (optional). YAML list of external MCP server endpoints.
aiMCPServers: ""
```

The API key or credentials are stored in a Kubernetes Secret, not the ConfigMap:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kfp-ai-credentials
type: Opaque
data:
  api-key: <base64-encoded-key>
```

### Backend Changes

#### API Server Extensions

New gRPC service registered in the API server when the AI feature is enabled:

```go
// AIService handles Gen-AI operations.
type AIService struct {
    provider   LLMProvider
    config     AIConfig
    runStore   storage.RunStoreInterface
    pipeStore  storage.PipelineStoreInterface
}
```

#### Proto Definitions

```protobuf
syntax = "proto3";

package kubeflow.pipelines.backend.api.v2beta1;

option go_package = "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client";

import "google/api/annotations.proto";
import "google/protobuf/duration.proto";

service AIService {
  // Generate pipeline code from a natural language description.
  rpc GeneratePipeline(GeneratePipelineRequest) returns (GeneratePipelineResponse) {
    option (google.api.http) = {
      post: "/apis/v2beta1/ai/pipelines/generate"
      body: "*"
    };
  }

  // Analyze a failed run and suggest fixes.
  rpc AnalyzeRun(AnalyzeRunRequest) returns (AnalyzeRunResponse) {
    option (google.api.http) = {
      post: "/apis/v2beta1/ai/runs/{run_id}/analyze"
    };
  }

  // Suggest optimizations for a pipeline.
  rpc SuggestOptimizations(SuggestOptimizationsRequest) returns (SuggestOptimizationsResponse) {
    option (google.api.http) = {
      post: "/apis/v2beta1/ai/pipelines/{pipeline_id}/optimize"
      body: "*"
    };
  }

  // Generate documentation for a pipeline.
  rpc GenerateDocumentation(GenerateDocumentationRequest) returns (GenerateDocumentationResponse) {
    option (google.api.http) = {
      post: "/apis/v2beta1/ai/pipelines/{pipeline_id}/document"
      body: "*"
    };
  }

  // Conversational query over pipeline data.
  rpc Chat(ChatRequest) returns (stream ChatResponse) {
    option (google.api.http) = {
      post: "/apis/v2beta1/ai/chat"
      body: "*"
    };
  }
}

// Service for AI feature discovery.
service AIFeatureService {
  // Returns whether AI features are enabled for the current deployment.
  rpc GetAIFeatureStatus(GetAIFeatureStatusRequest) returns (GetAIFeatureStatusResponse) {
    option (google.api.http) = {
      get: "/apis/v2beta1/ai/enabled"
    };
  }
}

message GeneratePipelineRequest {
  string description = 1;         // Natural language description
  string sdk_version = 2;         // Target KFP SDK version
  repeated string components = 3; // Optional: preferred component names
}

message GeneratePipelineResponse {
  string python_code = 1;         // Generated KFP Python SDK code
  string explanation = 2;         // Explanation of the generated pipeline
  repeated string warnings = 3;   // Validation warnings
  bool compiles = 4;              // Whether the code passes compilation check
}

message AnalyzeRunRequest {
  string run_id = 1;
}

message AnalyzeRunResponse {
  string summary = 1;                 // Human-readable failure summary
  string root_cause = 2;              // Identified root cause
  repeated string suggested_fixes = 3; // Actionable fix suggestions
  repeated string relevant_logs = 4;   // Key log excerpts
}

message SuggestOptimizationsRequest {
  string pipeline_id = 1;
  string pipeline_version_id = 2;
  int32 recent_runs_to_analyze = 3;  // Number of recent runs to consider
}

message SuggestOptimizationsResponse {
  repeated Optimization optimizations = 1;
}

message Optimization {
  string category = 1;     // "resources", "caching", "parallelism", "cost"
  string description = 2;
  string impact = 3;       // "high", "medium", "low"
  string suggested_change = 4;
}

message GenerateDocumentationRequest {
  string pipeline_id = 1;
  string pipeline_version_id = 2;
  string format = 3;        // "markdown", "plain"
}

message GenerateDocumentationResponse {
  string documentation = 1;
}

message ChatRequest {
  string message = 1;
  string conversation_id = 2;  // For multi-turn context
  string namespace = 3;        // Scope queries to a namespace
}

message ChatResponse {
  string content = 1;
  bool is_final = 2;
}

// Feature status messages.

message GetAIFeatureStatusRequest {}

message GetAIFeatureStatusResponse {
  bool enabled = 1;                     // Whether AI features are enabled
  repeated string available_modes = 2;  // Available chat modes ("ask", "agent")
}

// Error modeling.
//
// gRPC status codes (e.g., UNAVAILABLE, RESOURCE_EXHAUSTED, PERMISSION_DENIED)
// are used for coarse-grained error semantics. For richer, client-visible error
// handling (e.g., rate-limit countdown timers, specific UI messages), services
// attach the following error detail in google.rpc.Status.details.

enum GenAIErrorCode {
  GENAI_ERROR_CODE_UNSPECIFIED = 0;
  GENAI_ERROR_CODE_RATE_LIMIT_EXCEEDED = 1;      // Paired with RESOURCE_EXHAUSTED
  GENAI_ERROR_CODE_LLM_PROVIDER_UNAVAILABLE = 2;  // Paired with UNAVAILABLE
  GENAI_ERROR_CODE_INVALID_CONFIGURATION = 3;      // Paired with FAILED_PRECONDITION
  GENAI_ERROR_CODE_INSUFFICIENT_PERMISSIONS = 4;   // Paired with PERMISSION_DENIED
  GENAI_ERROR_CODE_CONTEXT_TOO_LARGE = 5;          // Paired with INVALID_ARGUMENT
}

message GenAIErrorDetail {
  GenAIErrorCode code = 1;              // Machine-readable error category
  string message = 2;                   // Human-readable error message for UI display
  google.protobuf.Duration retry_after = 3;  // When to retry (for rate limiting)
  string resource = 4;                  // Associated resource (pipeline ID, config field, etc.)
}
```

### Frontend Changes

#### AI Assistant Panel

A collapsible side panel accessible from a toolbar button throughout the UI. The panel provides:

- A chat interface for conversational interactions.
- Context-aware prompts based on the current page (e.g., on a failed run page, the panel
  pre-populates "Analyze this run failure").
- Code blocks with syntax highlighting and a "Copy to clipboard" button for generated code.
- Streaming response display with a typing indicator.

The panel is only rendered when the AI feature is enabled (detected via a feature flag endpoint).

#### Inline AI Features

Context-specific AI actions integrated into existing UI surfaces:

- **Run details page**: "Analyze failure" button on failed runs.
- **Pipeline details page**: "Generate docs" and "Suggest optimizations" buttons.
- **Pipeline list**: "Create with AI" button alongside the existing upload flow.

### SDK Extensions

An optional `kfp[ai]` extra dependency provides local AI-assisted authoring:

```python
from kfp.ai import generate_pipeline

# Generate a pipeline from a description
code = generate_pipeline(
    description="Load CSV from GCS, train a scikit-learn random forest, evaluate, and deploy",
    sdk_version="2.15.0",
)
print(code)
```

This is a convenience wrapper that calls the API server's `GeneratePipeline` endpoint. It is not
required — the UI provides the same functionality.

**Authentication and configuration**:

- The `kfp.ai` helpers reuse the same configuration and authentication mechanisms as the standard
  `kfp.Client` (kubeconfig, in-cluster configuration, environment variables, or explicit
  host/token settings).
- No additional initialization beyond installing the `kfp[ai]` extra is required; it relies on
  the underlying client configuration to locate and authenticate with the API server.

**Feature availability**:

- If the connected API server does not have AI features enabled, calls to `kfp.ai` functions
  raise a `kfp.client.ApiException` with an appropriate HTTP status (404 or 501). Callers should
  treat this as a signal that AI-assisted authoring is not available in the current environment.

**Error handling and exceptions**:

- Network and connectivity issues surface as the same connection-related exceptions raised by
  `kfp.Client`.
- Authentication and authorization problems (invalid token, insufficient RBAC permissions) result
  in `ApiException` with HTTP 401/403 status codes.
- LLM provider errors (quota exceeded, provider unavailability, invalid model configuration) are
  propagated as `ApiException` with the `GenAIErrorDetail` from the server response, allowing
  callers to inspect the structured error code and retry-after duration.
- Callers should wrap invocations in standard try/except blocks and handle these cases according
  to their application needs (retry, degrade gracefully, or display an informative error).

### Security and Privacy

- **Opt-in only**: The entire feature is disabled by default. No LLM calls are made unless an
  administrator explicitly configures the provider.
- **Credential isolation**: API keys are stored in Kubernetes Secrets, never in ConfigMaps or
  environment variables logged to stdout.
- **Context filtering**: Administrators configure which categories of pipeline data
  (logs, parameters, metrics, artifacts, pipeline specs) may be sent to the LLM provider.
- **Secret redaction**: Environment variable values, secret references, and credential-like strings
  are automatically redacted before inclusion in LLM prompts.
- **Prompt injection defense**: All user-provided content (pipeline names, parameters, log lines)
  is sanitized and placed in structured prompt sections with clear system/user boundaries.
- **Audit logging**: All AI service calls are logged with the requesting user, namespace, operation
  type, and token count (but not the prompt or response content).
- **RBAC**: AI endpoints respect existing KFP RBAC. In multi-user mode, users can only query
  pipelines and runs within their authorized namespaces.

### Observability, Metrics, and Audit Logging

To support safe, cost-effective operation in production, the AI integration exposes observability
signals and audit trails:

**Prometheus metrics** (per provider/model where applicable):

- `kfp_ai_llm_requests_total{provider,model,operation,result}` — Request counters.
- `kfp_ai_llm_request_duration_seconds_bucket{provider,model,operation}` — Request latency
  histograms.
- `kfp_ai_llm_tokens_total{provider,model,operation,kind}` — Token usage counters (kind is
  `prompt` or `completion`).
- `kfp_ai_llm_errors_total{provider,model,operation,code}` — Error counters by type, including
  timeouts, rate limits, and provider errors.
- `kfp_ai_llm_rate_limit_hits_total{provider,model,operation}` — Rate limiting event counters.
- `kfp_ai_llm_estimated_cost_usd_total{provider,model,operation}` — Optional cost estimates when
  token pricing is configured.

**Administrator monitoring**:

- Operators can create Grafana dashboards over these metrics to track usage by provider, model,
  and operation.
- Documentation will include example dashboard panels for request volume/latency, token usage and
  estimated cost, and error rates.

**Audit log contents** (in addition to existing cluster audit logs):

- Timestamp and unique request/correlation ID.
- Caller identity (user/service account) and originating namespace.
- Operation type (e.g., `GeneratePipeline`, `AnalyzeRun`, `Chat`).
- Provider and model identifier.
- Token counts for prompt and completion.
- Result status (success/failure) and error category on failure.
- Estimated cost when available.
- Prompt and response content are **not** recorded to avoid leaking sensitive data.

**Alerting recommendations**:

- Sustained elevated error rates for a provider/model/operation.
- Frequent rate-limit hits approaching provider quotas.
- Sudden spikes in token usage or estimated cost beyond configured thresholds.

### Test Plan

#### Prerequisite testing updates

- [ ] I/we understand the owners of the involved components may require updates to existing tests
  to make this code solid enough prior to review.

#### Unit Tests

| Package | Coverage Target | Key Tests |
|---------|----------------|-----------|
| `backend/src/apiserver/ai` | 80%+ | Provider abstraction, prompt template rendering, context building, secret redaction, rate limiting |
| `backend/src/apiserver/ai/providers` | 80%+ | Each provider implementation (mock HTTP responses), error handling, retry logic |
| `frontend/src/components/AIAssistant` | 70%+ | Panel rendering, streaming display, feature flag gating, error states |

#### Integration Tests

- API server with a mock LLM provider verifying end-to-end request flow.
- GeneratePipeline → compile validation → response flow.
- AnalyzeRun with seeded failed run data.
- RBAC enforcement on AI endpoints in multi-user mode.
- Rate limiting behavior under concurrent requests.

#### E2E Tests

- Enable AI feature via ConfigMap, verify UI panel appears.
- Disable AI feature, verify UI panel is hidden and endpoints return 404.
- Generate a pipeline from natural language, compile it, and verify it produces a valid workflow.
- Analyze a deliberately failed run and verify the response contains relevant log context.

#### Security Tests

- **Prompt injection resilience**: Craft adversarial prompts (including attempts to exfiltrate
  secrets, bypass RBAC, or request raw logs) and verify the system refuses unsafe actions and does
  not leak sensitive data.
- **Secret redaction edge cases**: Verify that secrets in environment variables, log messages,
  base64-encoded credentials, and other obfuscated forms are redacted before being sent to the
  LLM or displayed in responses.
- **Context filtering validation**: Ensure disallowed context types are never included in LLM
  request payloads, even under error conditions or unusual query patterns.
- **Multi-tenant isolation**: In multi-user/multi-namespace setups, attempt to query or reference
  resources from other namespaces and verify that the AI assistant respects RBAC boundaries and
  cannot access or describe cross-tenant data.
- **Session fixation**: Verify that chat session IDs cannot be reused across users or hijacked
  by concurrent requests.

### Graduation Criteria

#### Alpha

- Feature is behind a feature flag (ConfigMap `aiProvider` must be explicitly set).
- OpenAI-compatible provider implemented and tested.
- UI AI assistant panel with chat interface.
- GeneratePipeline and AnalyzeRun endpoints functional.
- Basic secret redaction in prompts.
- Documentation for administrator setup.

#### Beta

- All four provider implementations (OpenAI, Vertex AI, Bedrock, self-hosted).
- Agent mode with confirmation gates for mutating operations.
- Prompt rules engine with namespace-scoped and mode-scoped rules.
- SuggestOptimizations and GenerateDocumentation endpoints.
- Inline AI features on run/pipeline detail pages.
- Rate limiting and token budget enforcement.
- Audit logging for all AI operations.
- Prometheus metrics for request volume, latency, token usage, and errors.
- Comprehensive context filtering controls.
- RBAC enforcement on all AI endpoints in multi-user mode.
- SDK `kfp[ai]` extension.

#### Stable

- Proven in production deployments with feedback incorporated.
- Performance benchmarks for prompt latency and token usage.
- Hardened prompt injection defenses with adversarial testing.
- MCP server exposing KFP operations for external AI clients.
- MCP client consuming external tool servers (model registries, data catalogs).
- Extension point for platform teams to register custom agent tools.
- Embedding model support for vector-based pipeline similarity search.
- Localization support for non-English interactions.
- Full E2E test coverage in CI.

## Frontend Considerations

This proposal has significant frontend impact:

- **New AI assistant panel**: A new collapsible side panel component accessible from the main
  toolbar. This is the largest frontend addition.
- **Inline action buttons**: New buttons on run details, pipeline details, and pipeline list pages.
- **Feature flag gating**: All AI UI elements must be conditionally rendered based on the
  `AIFeatureService.GetAIFeatureStatus` endpoint (`GET /apis/v2beta1/ai/enabled`). When the
  feature is disabled, there must be zero visual or functional changes to the existing UI. The
  response also indicates available chat modes, allowing the UI to show/hide the mode selector.
- **Streaming response handling**: The chat interface must handle server-sent events or gRPC
  streaming for real-time LLM response display.
- **Code rendering**: Generated pipeline code must be displayed with Python syntax highlighting
  and a copy-to-clipboard action.
- **Loading states**: LLM responses may take 5-30 seconds. The UI must provide appropriate
  progress indicators and allow cancellation.
- **Error handling**: Graceful handling of LLM provider errors, rate limit exceeded, and timeout
  scenarios.

No changes to existing frontend components are required. The AI features are purely additive.

## KFP Local Considerations

The Gen-AI features are backend-dependent and require the API server with LLM provider
configuration. They are **not available in KFP local mode** (local compilation and execution
without a remote backend).

However, the SDK `kfp[ai]` extension (Beta milestone) could support a local mode where the user
provides their own LLM API key via environment variable, enabling local pipeline generation without
a deployed KFP backend. This local SDK feature would call the LLM provider directly, bypassing the
API server.

## Migration Strategy

No migration is required. This is a purely additive feature with no changes to existing APIs,
data models, or workflows. Existing deployments are unaffected when the feature is not enabled.

When upgrading to a version that includes this feature:

- The new ConfigMap keys default to empty (feature disabled).
- No new environment variables are required unless the feature is enabled.
- No database schema changes are required.
- No changes to existing pipeline specs or runs.

## Implementation History

- 2026-02-19: Initial proposal

## Drawbacks

- **Increased API server complexity**: Adding an LLM integration layer increases the surface area
  of the API server. This is mitigated by the modular design — the AI service is a separate
  package that is only initialized when enabled.
- **External dependency on LLM providers**: The feature depends on external LLM APIs which may
  have availability, latency, or cost issues. This is mitigated by supporting self-hosted models
  and ensuring graceful degradation.
- **Maintenance burden for prompt templates**: As the KFP SDK evolves, prompt templates that
  reference SDK APIs must be updated. This requires ongoing maintenance.
- **User trust in generated code**: Users may over-trust AI-generated pipeline code without
  adequate review. The UI should include clear disclaimers and encourage review.
- **Potential for scope creep**: Gen-AI features can expand indefinitely. The graduation criteria
  define a bounded scope for each phase.

## Alternatives

### External Plugin Architecture

Instead of building AI capabilities into the KFP API server, expose a plugin API that allows
third-party AI assistants to integrate with KFP.

**Pros**: Lower maintenance burden for the core team; allows ecosystem innovation.
**Cons**: Fragmented user experience; no out-of-the-box solution; plugin API design is a
significant effort itself. Organizations would need to build or source their own plugin.

**Decision**: Rejected for the initial implementation. A plugin API could be extracted from the
built-in implementation in a future phase if there is demand.

### SDK-Only Approach

Provide AI capabilities exclusively through the Python SDK (e.g., `kfp.ai.generate_pipeline()`)
without any UI or backend changes.

**Pros**: Simpler implementation; no frontend or API server changes.
**Cons**: Excludes UI-only users; cannot leverage server-side context (run logs, metrics, pipeline
history) for analysis and optimization; each user must configure their own LLM credentials.

**Decision**: Rejected as the sole approach. The SDK extension is included as a complementary
feature (Beta milestone), but the primary experience is through the UI and backend where
server-side context is available.

### Fine-Tuned Domain Model

Train or fine-tune a model specifically on KFP documentation, SDK code, and pipeline examples
rather than using general-purpose LLMs with prompt engineering.

**Pros**: Potentially higher accuracy for KFP-specific tasks; no dependency on external providers.
**Cons**: Significant training infrastructure and data curation effort; ongoing retraining as the
SDK evolves; large model hosting requirements; limited by training data rather than adaptable via
prompts.

**Decision**: Rejected for the initial implementation. RAG with general-purpose models provides
a better cost/quality tradeoff. A fine-tuned model could be explored as a future enhancement if
prompt engineering proves insufficient.
