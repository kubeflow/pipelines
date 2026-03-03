# Kubeflow Pipelines Roadmap (2026+)

This roadmap focuses on improving reliability, developer experience, and platform flexibility across the KFP monorepo (`backend/`, `sdk/`, `frontend/`, and `kubernetes_platform/`).

## Roadmap goals

- Improve day-1 developer onboarding and day-2 maintainability.
- Make local and CI workflows more deterministic and easier to debug.
- Increase confidence in releases through stronger test signal and compatibility coverage.
- Continue platform-agnostic execution and reduce coupling to specific workflow engines.
- Keep user-facing documentation accurate and aligned with the current code paths.

## 2026 H1 priorities (Current)

### 1) Developer productivity and repo ergonomics

- Keep root-level contributor docs synchronized (`README.md`, `AGENTS.md`, `developer_guide.md`).
- Consolidate and clarify the most-used local dev commands for SDK, backend, and frontend.
- Reduce setup friction for Python, Go, and Node environments.
- Improve troubleshooting guidance for common local failures.

### 2) Test reliability and confidence

- Improve presubmit signal quality by prioritizing deterministic suites and reducing flaky paths.
- Keep clear separation between unit, integration, and end-to-end test scopes.
- Expand and maintain critical-path test coverage for SDK compilation and backend API behavior.
- Strengthen coverage in areas with historical regressions (compiler output compatibility, runtime behavior, and UI critical flows).

### 3) API/spec/runtime consistency

- Keep protobuf- and swagger-generated assets in sync with source definitions.
- Improve validation around spec evolution and backward compatibility expectations.
- Continue aligning SDK compiler output with backend runtime expectations.
- Strengthen guardrails for generated code update workflows.

### 4) Frontend quality and maintainability

- Maintain React 17 compatibility while reducing migration blockers.
- Improve type safety and lint cleanliness in key UI surfaces.
- Keep API-client generation and usage patterns consistent.
- Prioritize UX reliability in run details, DAG views, and recurring workflow management.

## 2026 H2 priorities (Planned)

### 1) Execution and platform flexibility

- Continue reducing engine-specific assumptions in APIs and internal abstractions.
- Improve interoperability for Kubernetes-native and external execution backends.
- Expand compatibility verification across supported Kubernetes versions and pipeline stores.

### 2) Performance and scalability

- Optimize critical backend paths for run creation, scheduling, and metadata access.
- Improve resource efficiency for high-scale run and artifact scenarios.
- Add targeted performance benchmarks to prevent regressions.

### 3) Release and upgrade quality

- Improve upgrade-path confidence with stronger compatibility and migration test coverage.
- Increase observability for failures in CI and local reproduction loops.
- Simplify release readiness checks and artifact verification.

## Ongoing quality tracks

### Security and supply chain

- Keep dependency hygiene and vulnerability scanning active across languages.
- Maintain secure defaults in manifests and runtime integrations.
- Improve visibility and triage workflow for security-related findings.

### Documentation quality

- Ensure SDK docstrings remain accurate and user-facing docs stay aligned.
- Refresh stale architecture and workflow documentation as implementation evolves.
- Keep generated-artifact guidance explicit so contributors avoid manual edits to generated files.

### Contributor experience

- Keep issue/PR contribution paths clear and actionable.
- Favor small, well-scoped changes with strong tests.
- Maintain predictable coding, lint, and formatting workflows.

## Success indicators

- Reduced average time for new contributors to complete first successful local test run.
- Fewer flaky CI failures and faster rerun recovery.
- Higher confidence in release branches due to stable compatibility checks.
- Lower frequency of documentation drift reports and setup-related support questions.

## Not in scope for this roadmap

- Large-scale architectural rewrites without clear migration paths.
- Broad feature additions that reduce reliability or testability.
- Changes that introduce unnecessary coupling across monorepo domains.

## Change policy for this file

When major workflows, CI matrices, or architectural boundaries change, update this roadmap and related contributor docs in the same PR.
