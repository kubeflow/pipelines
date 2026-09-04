# Go version policy

This repository treats the effective compiler selected by the root `go.mod`
as the canonical Go version. The automation is deliberately a small policy
tool for known repository conventions, not a general Docker, YAML, shell, or
Git interpreter.

## Managed inventory

All tracked Go modules are managed. The current set is:

- `go.mod`
- `api/go.mod`
- `backend/api/tools/go.mod`
- `kubernetes_platform/go.mod`
- `test/tools/project-cleaner/go.mod`
- `third_party/ml-metadata/go.mod`

The managed Go builder images are the single builder `FROM` instructions in:

- `backend/Dockerfile`
- `backend/Dockerfile.cacheserver`
- `backend/Dockerfile.conformance`
- `backend/Dockerfile.driver`
- `backend/Dockerfile.launcher`
- `backend/Dockerfile.persistenceagent`
- `backend/Dockerfile.scheduledworkflow`
- `backend/Dockerfile.viewercontroller`
- `backend/api/Dockerfile`

Go setup is managed through these composite actions:

- `.github/actions/setup-go/action.yml`
- `.github/actions/test-and-report/action.yml`

## Canonical forms

- The root module contains one exact `go 1.X.Y` directive and may contain one
  exact `toolchain go1.X.Y` directive. The toolchain directive is authoritative
  when present; otherwise the `go` directive selects the compiler.
- Every other module uses the canonical compiler's major and minor version,
  does not require a newer patch, and either names the canonical toolchain or
  uses the exact canonical version in its `go` directive.
- Each managed builder is a literal, digest-pinned instruction of the form
  `FROM golang:1.X.Y[-flavor]@sha256:<64 lowercase hex> AS <stage>`.
- Each managed setup action has one literal `uses: actions/setup-go@...` step
  whose immediately following `with` block starts with
  `go-version-file: go.mod`. Workflows call those actions instead of invoking
  `actions/setup-go` directly.

## Guarantees

The consistency check verifies that all tracked modules and every declared
builder and setup action follow these forms and agree with the root compiler.
Builder images with the same flavor must use the same tag and digest.

The inventory guards are deliberately lexical. A literal Go source or setup-go
marker in a Dockerfile/YAML comment or heredoc is still reported so a maintainer
can remove or register it. The tool does not decide whether that text executes.

The updater accepts an exact stable `1.X.Y` target, computes and validates all
expected content before changing managed files, resolves one immutable digest
per image flavor, and is idempotent. Expected validation or resolution errors
occur before writes begin.

A real update requires managed files to be clean. An immediate no-op rerun may
leave the updater's previous diff uncommitted; digest resolution therefore
occurs before the tool decides whether a clean-path check is necessary.

## Explicit non-goals

The automation does not:

- interpret arbitrary Dockerfiles, custom Docker frontends, stage graphs,
  `ONBUILD` programs, YAML programs, or shell expansions;
- discover images or downloads assembled dynamically through variables,
  scripts, generated files, or external tools;
- prove runtime behavior or validate unrelated tool-installation provenance;
- implement a repository-wide Git transaction, lock manager, index snapshot,
  rollback engine, or recovery-bundle format; Git remains the recovery tool;
- preserve edits already present in managed files. Run it with those files
  clean and review the resulting diff normally.

## Extending the policy

- A new tracked `go.mod` is automatically managed and must satisfy the module
  rules.
- A new Go builder must use the canonical literal form and be added to the
  declared builder inventory and its focused tests.
- New CI users must route through a managed setup action. Add another action to
  the inventory only when routing through an existing one is unsuitable.
- If a use case needs dynamic Docker or shell behavior, handle that use case
  explicitly outside this updater. Do not widen this tool into an interpreter.
- Changes to supported forms must update this document and add a small test for
  the new form before changing the implementation.
