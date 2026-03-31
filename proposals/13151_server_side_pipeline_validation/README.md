# KEP-13151: Server-Side Pipeline Validation for Kubeflow Pipelines

**Authors:** Ugo Giordano (@ugiordan)

**Status:** Provisional

**Created:** 2026-03-27

**Last Updated:** 2026-03-31

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Overview](#overview)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration](#configuration)
  - [Validation Rules](#validation-rules)
  - [Credential Detection Patterns](#credential-detection-patterns)
  - [Validation Result](#validation-result)
  - [Integration Point](#integration-point)
  - [Panic Recovery](#panic-recovery)
- [Test Plan](#test-plan)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
  - [E2E Tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Migration Strategy](#migration-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Add a configurable, server-side validation rules engine to the Kubeflow Pipelines API server that inspects pipeline specs at admission time — before pipelines run. Platform administrators define policies (allowed registries, mutable tag blocking, credential detection, task limits) via a configuration resource, and the existing `PipelineVersion` validating webhook enforces them. Three enforce modes (`enforce`, `audit`, `off`) enable gradual rollout.

## Motivation

Data scientists build ML pipelines using the KFP SDK, compile them to YAML (the KFP Intermediate Representation), and submit them for execution. When something goes wrong — a `:latest` tag, a leaked API key, an unauthorized container registry — the failure happens **after** the pipeline is already running. Each failure wastes 10–30 minutes in submit-wait-fail cycles, and some failures (credential leaks) have security implications that runtime detection cannot prevent.

Today, Kubeflow Pipelines validates pipeline specs for structural correctness (valid IR, valid DAG) but does **not** validate them against organizational policies. Platform administrators have no built-in mechanism to enforce:

- Which container registries are allowed
- Whether mutable image tags (`:latest`) are permitted
- Whether hardcoded credentials appear in pipeline specs
- Pipeline complexity limits (max tasks)

These policies differ per deployment, and administrators need per-namespace control.

### Goals

- Provide a **server-side validation rules engine** that runs in the existing KFP API server process — zero new infrastructure
- Support **configurable policies** that administrators control via a Kubernetes resource (e.g., ConfigMap, CR field, or API server flags)
- Implement **three enforce modes** (`enforce`, `audit`, `off`) for safe, gradual rollout
- Hook into the **existing `PipelineVersion` validating webhook** path — no new webhook configurations
- Deliver **sub-millisecond overhead** with panic recovery so validation never crashes the API server
- Provide a **clear, actionable validation report** (denials and warnings with rule names, descriptions, and affected resources)

### Non-Goals

- Client-side / SDK validation (out of scope for this KEP — complementary but independent)
- OPA/Gatekeeper integration (future work; this KEP provides a built-in, zero-dependency solution)
- Runtime enforcement (this KEP covers admission-time only)
- Image signature verification (requires external infrastructure like Sigstore/cosign)
- Network policy enforcement

## Proposal

### Overview

Extend the existing `PipelineVersion` validating webhook in the KFP API server with a pluggable validation rules engine. When a pipeline is submitted:

1. The webhook deserializes the KFP IR (protobuf) from `spec.pipelineSpec`
2. It extracts the `DeploymentSpec` and iterates over each executor's container spec
3. Each enabled validation rule inspects the container spec and produces findings (denials or warnings)
4. Based on the enforce mode, denials either block submission (`enforce`), are logged as warnings (`audit`), or are skipped entirely (`off`)

```
PipelineVersion CR ──► Validating Webhook ──► Existing Checks
     (create)                                      │
                                                   ▼
                                          PipeClear Validation
                                                   │
                                          ┌────────┴────────┐
                                          │  Rules Engine    │
                                          │  ─────────────   │
                                          │  Registry Check  │
                                          │  Tag Check       │
                                          │  Credential Scan │
                                          │  Task Limit      │
                                          │  Env Var Check   │
                                          └────────┬────────┘
                                                   │
                                          ┌────────┴────────┐
                                          │  Enforce Mode    │
                                          │  enforce → deny  │
                                          │  audit → warn    │
                                          │  off → skip      │
                                          └─────────────────┘
```

### User Stories

**Story 1: Platform Admin — Registry Lockdown**

As a platform administrator, I want to restrict pipeline container images to approved registries (`registry.redhat.io`, `quay.io/myorg`) so that data scientists cannot accidentally or intentionally pull images from untrusted sources.

**Story 2: Security Team — Credential Detection**

As a security engineer, I want the system to automatically detect and block pipelines that contain hardcoded API keys, tokens, or PEM keys in environment variables or command arguments, so that credentials are never stored in pipeline definitions.

**Story 3: Platform Admin — Gradual Policy Rollout**

As a platform administrator deploying validation for the first time, I want to run in `audit` mode to see what would be blocked without actually disrupting existing workflows, then switch to `enforce` once policies are tuned.

**Story 4: Data Scientist — Actionable Feedback**

As a data scientist, when my pipeline submission is blocked, I want to see exactly which rule was violated, which executor/image triggered it, and what I need to fix — not a generic "validation failed" error.

### Notes/Constraints/Caveats

- Validation runs in-process within the API server — no external calls, no additional latency beyond microseconds
- All rules are opt-in or have sensible defaults that avoid breaking existing pipelines
- The panic recovery wrapper (`SafeValidatePipelineSpec`) ensures that a bug in validation rules never crashes the API server
- Configuration changes take effect on the next pipeline submission (no API server restart required if using ConfigMap/CR watch)

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Validation bug crashes the API server | `SafeValidatePipelineSpec` wraps all validation with `recover()` — panics are caught and logged, pipeline proceeds |
| False positives block legitimate pipelines | `audit` mode allows tuning before enforcing; credential regex patterns are conservative |
| Performance overhead | Validation is pure in-memory string matching — benchmarked at <1ms for pipelines with 100+ tasks |
| Configuration drift across namespaces | Configuration is per-DSPA CR (or per-namespace ConfigMap), giving admins explicit control |

## Design Details

### Configuration

The validation configuration can be provided through multiple mechanisms (implementation-specific):

```yaml
# Option A: ConfigMap (simplest)
apiVersion: v1
kind: ConfigMap
metadata:
  name: pipeclear-config
  namespace: kubeflow
data:
  config.yaml: |
    mode: enforce                    # enforce | audit | off
    allowedRegistries:
      - registry.redhat.io
      - quay.io/myorg
    blockMutableTags: true           # Block :latest or missing tags (warning in enforce, denial if combined with registry rules)
    blockInlineCredentials: true     # Detect hardcoded secrets
    maxTasks: 100                    # 0 = unlimited
    deniedEnvVarPatterns:
      - _PASSWORD
      - _SECRET
      - _TOKEN
      - _API_KEY
    warnDigestPinning: false         # Opt-in: warn on missing @sha256:
    warnSemverTags: false            # Opt-in: warn on non-semver tags
    warnResourceLimits: false        # Opt-in: warn on missing CPU/mem limits
    warnDuplicateTasks: true         # Warn on identical executor configs
```

```yaml
# Option B: DSPA CR field (for OpenDataHub/RHOAI deployments)
apiVersion: datasciencepipelinesapplications.opendatahub.io/v1alpha1
kind: DataSciencePipelinesApplication
metadata:
  name: dspa
spec:
  # ... existing fields ...
  pipeClear:
    mode: enforce
    allowedRegistries:
      - registry.redhat.io
      - quay.io/myorg
    blockMutableTags: true
    blockInlineCredentials: true
    maxTasks: 100
```

### Validation Rules

| Rule | Inspects | Finding Type | Default |
|------|----------|-------------|---------|
| **Registry Allowlist** | `container.image` hostname | Denial | Disabled (no allowlist = all allowed) |
| **Mutable Tags** | `container.image` tag (`:latest` or missing) | Warning (logged, does not block) | Enabled |
| **Credential Detection** | `container.env[].value`, `container.args[]`, `container.command[]` | Denial | Enabled |
| **Denied Env Vars** | `container.env[].name` matching patterns | Denial | Enabled (common patterns) |
| **Max Tasks** | DAG task count | Denial | 100 |
| **Digest Pinning** | `container.image` missing `@sha256:` | Warning | Disabled |
| **Semver Tags** | `container.image` tag not matching semver | Warning | Disabled |
| **Resource Limits** | `container.resources` missing CPU/memory limits | Warning | Disabled |
| **Duplicate Tasks** | Identical executor configurations | Warning | Enabled |

### Credential Detection Patterns

The credential detector scans environment variable values, command arguments, and container args for patterns indicating hardcoded secrets:

- **GitHub PATs:** `ghp_[A-Za-z0-9]{36}`
- **AWS Access Keys:** `AKIA[0-9A-Z]{16}`
- **Generic API Keys:** `(?i)(api[_-]?key|api[_-]?secret|access[_-]?token)[\s]*[=:]\s*['"]?[A-Za-z0-9+/=]{20,}`
- **PEM Private Keys:** `-----BEGIN (RSA |EC |DSA )?PRIVATE KEY-----`
- **Bearer Tokens:** `(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`
- **Base64 Encoded Secrets:** High-entropy strings in env vars named `*_SECRET`, `*_TOKEN`, etc.

### Validation Result

```go
type ValidationResult struct {
    Denials  []Finding
    Warnings []Finding
}

type Finding struct {
    Rule        string // e.g., "registry-allowlist"
    Severity    string // "denial" or "warning"
    Message     string // Human-readable description
    Executor    string // Which executor triggered it
    Image       string // The container image (if applicable)
    EnvVar      string // The env var name (if applicable)
}
```

Example denial message returned to the user:

```
Pipeline validation failed (PipeClear):

DENIED:
  [registry-allowlist] executor "train": image "docker.io/library/python:3.11"
    is from registry "docker.io" which is not in the allowed list:
    [registry.redhat.io, quay.io/myorg]

  [credential-detection] executor "deploy": env var "API_KEY" contains
    what appears to be a hardcoded API key

WARNINGS:
  [mutable-tag] executor "preprocess": image "quay.io/myorg/etl:latest"
    uses mutable tag ":latest" — consider pinning to a specific version
```

### Integration Point

The validation hooks into the existing `PipelineVersion` validating webhook. In pseudocode:

```go
func (w *PipelineVersionWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
    pv := obj.(*v2beta1.PipelineVersion)

    // Existing structural validation
    if err := w.validatePipelineSpec(pv); err != nil {
        return err
    }

    // NEW: PipeClear policy validation
    config := w.loadPipeClearConfig(ctx)
    if config.Mode != "off" {
        result, err := SafeValidatePipelineSpec(pv.Spec.PipelineSpec, config)
        if err != nil {
            // Validation itself errored — log and allow (fail-open)
            log.Error(err, "PipeClear validation error")
            return nil
        }
        if len(result.Warnings) > 0 {
            log.Info("PipeClear warnings", "warnings", result.FormatWarnings())
        }
        if config.Mode == "enforce" && len(result.Denials) > 0 {
            return fmt.Errorf("pipeline validation failed:\n%s\n%s", result.FormatDenials(), result.FormatWarnings())
        }
        if config.Mode == "audit" && len(result.Denials) > 0 {
            log.Info("PipeClear audit", "denials", result.FormatDenials(), "warnings", result.FormatWarnings())
        }
    }

    return nil
}
```

### Panic Recovery

```go
func SafeValidatePipelineSpec(spec interface{}, config Config) (result ValidationResult, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("validation panicked: %v", r)
            // Log stack trace for debugging
        }
    }()
    return ValidatePipelineSpec(spec, config)
}
```

## Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

### Unit Tests

- **Validation rules:** Each rule has dedicated test functions covering positive matches, negative matches, and edge cases
- **Configuration parsing:** Tests for default values, partial configs, invalid configs
- **Panic recovery:** Tests that intentionally panic inside validation and verify recovery
- **Result formatting:** Tests for human-readable output formatting
- **Package:** `backend/src/apiserver/webhook/` — target >90% coverage for new code

A reference implementation with 31 test functions is available at [ugiordan/data-science-pipelines@feat/pipeclear-validation](https://github.com/ugiordan/data-science-pipelines/tree/feat/pipeclear-validation).

### Integration Tests

- Submit a pipeline with a disallowed registry image in `enforce` mode → verify rejection
- Submit a pipeline with a disallowed registry image in `audit` mode → verify acceptance with logged warnings
- Submit a pipeline in `off` mode → verify no validation occurs
- Change configuration and verify next submission uses new config
- Submit a valid pipeline → verify it passes through unchanged

### E2E Tests

- Deploy KFP with PipeClear enabled, submit pipelines via the SDK, and verify enforcement behavior end-to-end
- Verify that existing pipelines (without policy violations) continue to work with PipeClear enabled

## Graduation Criteria

### Alpha

- Validation rules engine implemented and tested
- Configuration via ConfigMap
- `enforce`, `audit`, `off` modes working
- Documentation for administrators

### Beta

- Configuration via CR field (for operator-managed deployments)
- UI integration: show validation warnings/denials in the pipeline submission response
- Metrics: Prometheus counters for denials, warnings, and errors per rule
- At least two production deployments in `audit` mode

### Stable

- Stable configuration API
- Comprehensive documentation with examples
- Performance benchmarks published

## Frontend Considerations

- Pipeline submission errors should display the structured validation report (rule name, executor, image, description) rather than a raw error string
- The UI could optionally show an "audit" badge on pipelines that passed with warnings
- No frontend changes required for the alpha implementation — validation errors flow through the existing error handling path

## KFP Local Considerations

- `kfp local` runs pipelines locally without the API server, so server-side validation does not apply
- Users who want local validation can use a complementary SDK-side solution (out of scope for this KEP)

## Migration Strategy

- **No breaking changes:** Validation is disabled by default when no configuration is provided
- **Gradual rollout path:** Deploy with `mode: off` → switch to `audit` to observe → switch to `enforce` to block
- **Existing pipelines:** Already-deployed pipelines are not affected; validation only runs on new `PipelineVersion` creation

## Implementation History

- 2026-03-27: KEP created (provisional)
- 2026-03: Reference implementation completed with 31 test functions ([ugiordan/data-science-pipelines@feat/pipeclear-validation](https://github.com/ugiordan/data-science-pipelines/tree/feat/pipeclear-validation))
- 2026-03-31: Go validation library extracted to standalone package ([ugiordan/pipeclear](https://github.com/ugiordan/pipeclear)) with 22 library tests. Clarified `blockMutableTags` semantics and pseudocode warning handling per review feedback.

## Drawbacks

- Adds complexity to the API server webhook path (mitigated by panic recovery and fail-open design)
- Credential detection via regex can have false positives (mitigated by `audit` mode for tuning)
- Does not replace external policy engines like OPA/Gatekeeper for organizations that already use them (but provides a simpler, zero-dependency alternative)

## Alternatives

### External Validating Webhook

Deploy a separate webhook service that intercepts `PipelineVersion` creation.

**Pros:** Decoupled from the API server; can be developed/deployed independently.
**Cons:** Requires additional infrastructure (Deployment, Service, TLS certificates); adds network latency; another component to operate and monitor.

**Why not chosen:** The KFP API server already runs a validating webhook — adding validation rules in-process is simpler, faster, and requires zero new infrastructure.

### OPA/Gatekeeper Integration

Use Open Policy Agent with Rego policies to validate pipeline specs.

**Pros:** Industry-standard policy engine; highly flexible; supports complex policy logic.
**Cons:** Requires OPA/Gatekeeper installed on the cluster; Rego has a learning curve; pipeline specs are deeply nested protobuf structures that are awkward to express in Rego.

**Why not chosen:** This KEP provides a built-in solution that works out of the box. OPA integration could be added as a future extension where the validation rules engine delegates to OPA for custom policies.

### SDK-Only Validation

Validate pipelines only in the KFP SDK at compile time.

**Pros:** Fastest feedback loop (catches issues in the notebook).
**Cons:** Cannot enforce policies — users can bypass the SDK and submit raw YAML; policies differ per deployment and the SDK doesn't know about server-side configuration.

**Why not chosen:** SDK validation is complementary but insufficient. Server-side validation is the only place where policies can be enforced.
