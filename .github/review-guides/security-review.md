# Security Review Guide

Load when reviewing PRs touching manifests, RBAC, security contexts, Dockerfiles, or credentials.

---

## Principles

- **Enforcement > defaults**: `runAsNonRoot: true` means nothing without Pod Security Admission enforcing it
- **Never configurable**: `allowPrivilegeEscalation`, `privileged` must be hardcoded `false` -- no env var overrides
- **System containers**: driver, launcher, API server must ALWAYS run non-root. User workloads may have documented exceptions.
- **Reject "helpful flexibility"**: Don't expose dangerous K8s APIs 1:1. Expose only the safe subset; hardcode secure defaults.
- **Defense in depth**: Backend must enforce policy independently of SDK. SDK is convenience; server is enforcement.
- **Threat model required**: PRs adding security-configurable fields should answer: "What could a malicious pipeline author do?"
- **Verify third-party assumptions**: "Argo runs as root" must be fact-checked against official docs.
- **Dockerfile USER**: All KFP Dockerfiles specify non-root USER (UID 65532) matching manifest `runAsUser`.

## Container SecurityContext checklist

**NEVER user-configurable:** `privileged`, `add_capabilities`, `allowPrivilegeEscalation`
**ALWAYS hardcoded:** `drop: ["ALL"]`, `seccompProfile: RuntimeDefault`
**MAY be user-configurable:** `runAsUser` (warn on UID 0), `runAsGroup`, `readOnlyRootFilesystem`

- Proto schema is the security boundary -- remove dangerous fields from proto, don't just hide in SDK
- Use `reserved` numbers for removed fields: `reserved 3, 4, 5; // forbidden by policy`
- Container-level `securityContext` must not conflict with pod-level `PodSecurityContext`
- Admin-set contexts take precedence over user-specified values
- Tests must not normalize violations (no golden files with `privileged: true`)

## RBAC manifest review

- `resourceNames` restrictions on `update`/`patch`/`delete` where possible
- `create` can't be scoped by `resourceNames` -- document why needed, validate in code
- Standalone role and multi-user cluster role updated in sync
- ClusterRoleBindings use Kustomize variables (`$(kfp-cluster-scoped-namespace)`)
- New ClusterRoles follow `ml-pipeline-*` naming; added to `kustomization.yaml`

## General

- No hardcoded credentials; credential chains for object store access
- Multi-user RBAC checks on new API endpoints
- Parameterized SQL queries only (squirrel builder)
- Input validation on pipeline spec sizes and uploads
- ConfigMap keys must match `[-._a-zA-Z0-9]+`; validate inputs used as keys
- ConfigMap/Secret cleanup on resource deletion (no orphans)
- CI workflows: no secrets in logs, `set -euo pipefail` in scripts, pinned action versions
