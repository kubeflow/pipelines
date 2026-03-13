# Architecture Context for Reviews

Load when reviewing PRs that touch multiple subsystems, modify controllers, or change class hierarchies.

**See also:** [impact-analysis.md](impact-analysis.md), [language-checklists.md](language-checklists.md)

---

## High-level architecture

```
SDK (Python) -> compile -> IR YAML (PipelineSpec proto)
  -> API Server (Go) -> compile -> Argo Workflow YAML
    -> Driver (Go init container) -> resolve inputs, check cache
      -> Launcher (Go) -> download artifacts -> invoke user container
        -> Executor (Python) -> component function -> output artifacts
          -> Launcher -> upload artifacts -> record in MLMD
```

## Key subsystem boundaries

| Subsystem | Language | Root path |
|-----------|----------|-----------|
| SDK DSL/Compiler/Client/Local | Python | `sdk/python/kfp/` |
| API Server | Go | `backend/src/apiserver/` |
| Backend Compiler | Go | `backend/src/v2/compiler/argocompiler/` |
| Driver | Go | `backend/src/v2/driver/` |
| Launcher | Go | `backend/src/v2/component/` |
| Cache Server | Go | `backend/src/cache/` |
| Object Store | Go | `backend/src/v2/objectstore/` |
| MLMD Client | Go | `backend/src/v2/metadata/` |
| CRD Controllers | Go | `backend/src/crd/controller/` |
| K8s Platform | Python+Proto | `kubernetes_platform/` |
| Pipeline Spec | Proto | `api/v2alpha1/` |
| Frontend | TypeScript | `frontend/` |

## Critical dependency directions

Flag violations of these rules:

```
sdk/python/kfp/compiler  -> sdk/python/kfp/dsl (never reverse)
sdk/python/kfp/client    -> sdk/python/kfp/dsl (never reverse)
backend/src/v2/*         -> never imports backend/src/apiserver/
backend/src/crd/controller/* -> backend/src/crd/pkg only, never apiserver/
```

Full dependency map: see [code-tree.md](../../docs/agents/code-tree.md#key-module-dependencies).

## Controller patterns

- **No synchronous API callbacks**: Controllers must not call back to the API server. Pass data via CRD spec/annotations/labels.
- **Idempotent reconciliation**: Each loop iteration must be safe to retry.
- **Informer-based reads**: Read from caches, not direct API calls.
- **New gRPC dependencies**: Require architectural justification.

## Cross-language struct alignment

When Go structs mirror Python dataclasses or Proto messages (e.g., `TaskKubernetesConfig`):

- Add cross-reference comments in both languages
- Proto is the source of truth
- Flag changes to one side without corresponding changes to the other

## Backward compatibility

For API/proto/data-format changes:

- What happens when an old backend receives new SDK output? New fields must have safe defaults.
- What happens when a new backend reads old persisted data? Migration path must be explicit.
- Key format changes (maps, ConfigMaps, databases) must handle old-format keys during transition.
- Cached executions may be invalidated -- document the upgrade path.

## Cardinality changes

When a value changes from single to list (or vice versa):

1. Use `--callers` and `--rdeps` to find all consumers
2. Check for `[0]` indexing, `.length == 1` assumptions, nil checks
3. Verify downstream systems (MLMD, cache keys, Argo specs)

## Inheritance hierarchies

Changes to base classes affect all subclasses. Verify LSP compliance. Key hierarchies in [sdk-guide.md](../../docs/agents/sdk-guide.md#artifact-types) and [code-tree.md](../../docs/agents/code-tree.md#key-inheritance-hierarchies).
