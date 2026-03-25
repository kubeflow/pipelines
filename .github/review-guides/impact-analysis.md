# Impact Analysis for Reviews

Load when assessing blast radius, proto changes, or reviewing XXL PRs.

**See also:** [architecture-context.md](architecture-context.md) (backward compat, cardinality), [test-quality.md](test-quality.md) (coverage adequacy)

---

## PR size and scope

- Flag PRs > 500 additions crossing 3+ subsystems as "needs decomposition discussion"
- XXL PRs (>1000 additions): each component needs dedicated tests, not just E2E
- Suggest splitting when a single PR includes: proto + backend + SDK + E2E + CI + manifests

## Hub files

Changes to these files need extra scrutiny. Full table with 9 entries in [code-tree.md](../../docs/agents/code-tree.md#hub-files-most-connected).

| File | Imported by |
|------|-------------|
| `backend/src/common/util/time.go` | 369 files |
| `backend/src/common/util/json.go` | 319 files |
| `backend/src/apiserver/common/utils.go` | 171 files |
| `backend/src/common/util/consts.go` | 155 files |

## Cross-boundary changes

Flag without clear justification:
- SDK changes that also modify backend code (unless end-to-end proto support)
- Backend changes modifying frontend API clients (should use code generation)
- Proto changes without test pipeline files
- Import violations (see [architecture-context.md](architecture-context.md#critical-dependency-directions))

## Cross-file consistency

When the same pattern is applied to many files (security contexts to 20+ manifests, RBAC to multiple roles):
- Spot-check at least 3 files for consistency
- Suggest automation (script or Kustomize patch) for 10+ files

## Proto change checklist

1. Backward compatible (no removed/renumbered fields)
2. ALL language targets regenerated (Go `.pb.go` + Python `_pb2.py` + frontend if applicable in diff)
3. Test pipelines in `test_data/sdk_compiled_pipelines/valid/`
4. Compiled workflow goldens updated in `test_data/compiled-workflows/`
5. `kubernetes_platform` protos regenerated if `pipeline_spec.proto` changed
6. SDK `pipeline_spec_builder.py` handles new user-facing fields
7. Backend compiler `argocompiler/argo.go` handles fields affecting compilation
8. Version skew analyzed (see [architecture-context.md](architecture-context.md#backward-compatibility))
9. Generated code formatting-only diffs noted in PR description

## Generated files

Never hand-edit. Key files listed in [protobuf-build.md](../../docs/agents/protobuf-build.md#never-edit-directly-generated-files). Reject PRs modifying generated files without regenerating from sources.
