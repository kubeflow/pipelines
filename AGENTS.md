# Kubeflow Pipelines

Kubeflow Pipelines is a monorepo for compiling, running, and managing machine-learning pipelines on Kubernetes.

## Bug fixes and PR evidence

- Start each bug fix with a minimal, repeatable reproducer. Record the steps or command, required inputs, and expected versus actual behavior in the PR description.
- Add an automated regression test that exercises the reported failure. Verify that it fails for the expected reason before the fix and passes after the fix; keep the test in the PR.
- Show the evidence in the PR description: identify the regression test, include the exact command used to run it, and report the before-fix failure and after-fix passing result. Link to the passing CI run when available. If verification is blocked, state what remains unverified instead of claiming the fix is proven.
- Include a picture, diagram, or animation in the PR description that explains the problem and the fix. Clearly label the before and after behavior; use screenshots or a recording for visible changes, or a diagram (such as Mermaid) for logic, data flow, and backend changes.

## Task-specific guides

Read the guide relevant to the files or task you are changing:

- [Conventions](docs/agents/conventions.md)
- [Architecture and runtime](docs/agents/architecture.md)
- [Development and deployment](docs/agents/development.md)
- [Testing and formatting](docs/agents/testing.md)
- [Nonblocking database changes](docs/agents/database-migrations.md) before changing
  models, indexes, migrations, startup SQL or backfills
- [Generated code and APIs](docs/agents/generation.md)
- [Frontend](docs/agents/frontend.md)
- [CI and workflows](docs/agents/ci.md)
- [Dependency updates](docs/agents/dependabot.md)
- [Go version policy](docs/agents/go-version-policy.md) when changing compiler,
  module, setup-action, or builder-image pins
- [Domain documentation](docs/agents/domain.md) when exploring domain behavior or architecture decisions
- [GitHub issue tracking](docs/agents/issue-tracker.md) for issue or pull-request work
- [Triage labels](docs/agents/triage-labels.md) for issue triage

## Pull request descriptions

When applicable, start PR descriptions with a `## Problem` section. Explain the
underlying problem and why it matters, assuming the reader has no prior context,
so they can understand the proposed solution. Omit this section for trivial,
self-explanatory changes, such as fixing a hyperlink.
