#   

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
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Architecture Overview](#architecture-overview)
  - [LLM Integration Layer](#llm-integration-layer)
    - [Provider Abstraction](#provider-abstraction)
    - [Configuration](#configuration)
  - [Backend Changes](#backend-changes)
    - [API Server Extensions](#api-server-extensions)
    - [Proto Definitions](#proto-definitions)
  - [Frontend Changes](#frontend-changes)
    - [AI Assistant Panel](#ai-assistant-panel)
    - [Inline AI Features](#inline-ai-features)
  - [SDK Extensions](#sdk-extensions)
  - [Security and Privacy](#security-and-privacy)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [E2E Tests](#e2e-tests)
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
```

Built-in provider implementations:

- **OpenAI-compatible**: Works with OpenAI, Azure OpenAI, vLLM, Ollama, and any OpenAI-compatible
  API endpoint.
- **Google Vertex AI**: Native integration with Gemini models via the Vertex AI API.
- **Amazon Bedrock**: Native integration with models available through Bedrock.
- **Self-hosted**: Generic HTTP/gRPC interface for self-hosted model servers.

#### Configuration

Configuration via `pipeline-install-config` ConfigMap:

```yaml
## Gen-AI configuration. Feature is disabled when aiProvider is empty (default).
aiProvider: ""              # "openai", "vertexai", "bedrock", "custom"
aiEndpoint: ""              # LLM API endpoint URL
aiModel: ""                 # Model identifier (e.g., "gpt-4o", "gemini-2.0-flash")
aiMaxTokensPerRequest: "4096"
aiRateLimitPerMinute: "30"
## Controls what pipeline context is sent to the LLM. Comma-separated.
## Options: "logs", "parameters", "metrics", "artifacts", "pipeline-spec"
aiAllowedContext: "logs,parameters,metrics,pipeline-spec"
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
service AIService {
  // Generate pipeline code from a natural language description.
  rpc GeneratePipeline(GeneratePipelineRequest) returns (GeneratePipelineResponse);

  // Analyze a failed run and suggest fixes.
  rpc AnalyzeRun(AnalyzeRunRequest) returns (AnalyzeRunResponse);

  // Suggest optimizations for a pipeline.
  rpc SuggestOptimizations(SuggestOptimizationsRequest) returns (SuggestOptimizationsResponse);

  // Generate documentation for a pipeline.
  rpc GenerateDocumentation(GenerateDocumentationRequest) returns (GenerateDocumentationResponse);

  // Conversational query over pipeline data.
  rpc Chat(ChatRequest) returns (stream ChatResponse);
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
- SuggestOptimizations and GenerateDocumentation endpoints.
- Inline AI features on run/pipeline detail pages.
- Rate limiting and token budget enforcement.
- Audit logging for all AI operations.
- Comprehensive context filtering controls.
- SDK `kfp[ai]` extension.

#### Stable

- Proven in production deployments with feedback incorporated.
- Performance benchmarks for prompt latency and token usage.
- Hardened prompt injection defenses with adversarial testing.
- Localization support for non-English interactions.
- Full E2E test coverage in CI.

## Frontend Considerations

This proposal has significant frontend impact:

- **New AI assistant panel**: A new collapsible side panel component accessible from the main
  toolbar. This is the largest frontend addition.
- **Inline action buttons**: New buttons on run details, pipeline details, and pipeline list pages.
- **Feature flag gating**: All AI UI elements must be conditionally rendered based on a feature
  flag endpoint (`/apis/v2beta1/ai/enabled`). When the feature is disabled, there must be zero
  visual or functional changes to the existing UI.
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
