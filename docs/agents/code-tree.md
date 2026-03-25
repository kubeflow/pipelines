# KFP Code-Tree Knowledge Graph

Pre-built knowledge graph artifacts, query commands, codebase statistics, entry points, hub files, module dependencies, inheritance hierarchies, and workflows for development and PR reviews.

**See also:** [architecture.md](architecture.md) (system architecture), [backend-guide.md](backend-guide.md) (backend files), [sdk-guide.md](sdk-guide.md) (SDK files)

---

## Development workflow

Follow this workflow for any code change. Regenerate the graph first, use queries to scope your change, then verify impact after.

### Before writing code

1. **Regenerate graph**: `python3 tools/code-tree/code_tree.py --repo-root . --incremental -q`
2. **Locate**: `python3 tools/code-tree/query_graph.py --symbol <name>` or `--search <keyword>`
3. **Scope**: `--rdeps <file>` to understand blast radius before changing anything
4. **Read**: Load the relevant guide from [AGENTS.md](../../AGENTS.md), then read the target files

### While writing code

5. **Guard generated files**: Never hand-edit files listed in [protobuf-build.md](protobuf-build.md). Edit sources and regenerate.
6. **Guard hub files**: Extra care when touching high-fanout files (see hub files below). Run `--impact <file> --depth 5` first.
7. **Match patterns**: Follow existing code style. Go: error wrapping with `%w`, no nested goroutines. Python: type hints, docstrings, YAPF formatting.
8. **Cross-boundary changes**: Verify dependency direction rules (see key module dependencies below).

### After writing code

9. **Run relevant tests**: See essential commands in [AGENTS.md](../../AGENTS.md)
10. **Run impact check**: `python3 tools/code-tree/query_graph.py --test-impact <changed-file>`
11. **Lint**: Run the appropriate formatter/linter for the language you changed

---

## Available artifacts

| File | Contents | Use case |
|------|----------|----------|
| `docs/code-tree/summary.md` | Architecture overview, hub files, key types, inheritance, module deps | **Start here** for any exploration |
| `docs/code-tree/graph.json` | Full knowledge graph (nodes + edges) | Programmatic queries with `jq` |
| `docs/code-tree/tags.json` | Flat symbol index (symbols with file:line locations) | Quick symbol lookup |
| `docs/code-tree/modules.json` | Directory-level dependency map | Module impact analysis |

## Querying the graph

Use the query tool at `tools/code-tree/query_graph.py`:

```bash
# Find where a symbol is defined
python3 tools/code-tree/query_graph.py --symbol Pipeline

# Trace what a file depends on
python3 tools/code-tree/query_graph.py --deps sdk/python/kfp/compiler/compiler.py

# Find what imports a file (reverse deps / blast radius)
python3 tools/code-tree/query_graph.py --rdeps sdk/python/kfp/dsl/pipeline_context.py

# Show inheritance hierarchy for a class
python3 tools/code-tree/query_graph.py --hierarchy Artifact

# Search symbols by keyword
python3 tools/code-tree/query_graph.py --search "compiler"

# Get module overview (files, deps, symbols)
python3 tools/code-tree/query_graph.py --module sdk/python/kfp/compiler

# Find entry points (main functions, unimported files)
python3 tools/code-tree/query_graph.py --entry-points

# Extract code chunks from a file for context
python3 tools/code-tree/query_graph.py --chunks sdk/python/kfp/compiler/compiler.py

# Who calls this function/method?
python3 tools/code-tree/query_graph.py --callers Compile

# What does this function call?
python3 tools/code-tree/query_graph.py --callees Container

# Find call paths between two symbols
python3 tools/code-tree/query_graph.py --call-chain CreateRun Compile

# Change impact analysis (transitive)
python3 tools/code-tree/query_graph.py --impact backend/src/v2/driver/container.go
python3 tools/code-tree/query_graph.py --impact Artifact --depth 8

# Which tests are affected by a change?
python3 tools/code-tree/query_graph.py --test-impact sdk/python/kfp/dsl/pipeline_task.py

# What tests cover a file?
python3 tools/code-tree/query_graph.py --test-coverage sdk/python/kfp/compiler/compiler.py

# Graph statistics
python3 tools/code-tree/query_graph.py --stats
```

Add `--json` to any query for machine-readable output.
Use `--depth N` to control impact/call-chain traversal depth (default: 5).

## Codebase statistics

Run `python3 tools/code-tree/query_graph.py --stats` for current numbers. Approximate as of 2026-03:

- **~2,750 files** across 7 languages: Python, Go, TypeScript, Bash, Protobuf, JavaScript, SQL
- **~22,000 symbols**: methods, functions, classes, structs, interfaces, constants, enums
- **~88,000 edges**: imports, contains, calls, tests, inherits
- **~47%** source file test coverage

## Entry points

| Entry point | Purpose |
|-------------|---------|
| `backend/src/apiserver/main.go` | API server (gRPC:8887 + HTTP:8888) |
| `backend/src/v2/cmd/driver/main.go` | V2 driver (ROOT_DAG/DAG/CONTAINER) |
| `backend/src/v2/cmd/compiler/main.go` | Backend compiler (PipelineSpec -> Argo Workflow) |
| `backend/src/v2/cmd/launcher-v2/main.go` | V2 launcher (artifact download/upload + executor invocation) |
| `backend/src/cache/main.go` | Cache server (mutating webhook) |
| `backend/src/agent/persistence/main.go` | Persistence agent |
| `backend/src/crd/controller/scheduledworkflow/main.go` | Scheduled workflow controller |
| `backend/src/crd/controller/viewer/main.go` | Viewer controller |
| `sdk/python/kfp/dsl/executor_main.py` | Python executor entrypoint |

## Hub files (most connected)

These files are imported by the most other files. Changes to them have the widest blast radius:

| File | Imported by |
|------|-------------|
| `backend/src/common/util/time.go` | 369 files |
| `backend/src/common/util/json.go` | 319 files |
| `backend/api/v1beta1/python_http_client/kfp_server_api/rest.py` | 186 files |
| `backend/api/v2beta1/python_http_client/kfp_server_api/rest.py` | 186 files |
| `backend/src/apiserver/common/utils.go` | 171 files |
| `backend/src/common/util/uuid.go` | 163 files |
| `backend/src/common/util/string.go` | 159 files |
| `backend/src/common/util/consts.go` | 155 files |
| `backend/src/common/util/error.go` | 155 files |

## Key module dependencies

```
sdk/python/kfp/compiler       -> sdk/python/kfp/dsl
sdk/python/kfp/client          -> sdk/python/kfp/dsl
backend/src/apiserver/*         -> backend/src/common/util
backend/src/apiserver/template  -> backend/src/v2/compiler/argocompiler
backend/src/apiserver/resource  -> backend/src/apiserver/storage, backend/src/apiserver/template
backend/src/cache/*             -> backend/src/apiserver/archive, backend/src/common/util
backend/src/v2/component        -> backend/src/v2/metadata, backend/src/v2/objectstore
backend/src/v2/driver            -> backend/src/v2/cacheutils, backend/src/v2/metadata, backend/src/v2/objectstore
backend/src/v2/compiler/argocompiler -> api/v2alpha1/go/pipelinespec, backend/src/v2/compiler
backend/src/v2/cmd/driver       -> backend/src/v2/driver, backend/src/v2/metadata, backend/src/v2/cacheutils
backend/src/v2/cmd/launcher-v2  -> backend/src/v2/component, backend/src/v2/client_manager
```

## Key inheritance hierarchies

```
Artifact (sdk/python/kfp/dsl/types/artifact_types.py)
  |- ClassificationMetrics, Dataset, HTML, Markdown, Metrics, Model, SlicedClassificationMetrics

ModelBase (sdk/python)
  |- BinaryPredicate
  |    |- EqualsPredicate, GreaterThanPredicate, LessThenPredicate, ...
  |- ComponentSpec, ContainerSpec, CachingStrategySpec, ...

BaseAPI (backend/api/*/python_http_client)
  |- ExperimentServiceApi, PipelineServiceApi, RunServiceApi, RecurringRunServiceApi, ...

OpenApiException
  |- ApiException, ApiKeyError, ApiTypeError, ApiValueError
```

## Regenerating the graph

Always regenerate before first use in a session:

```bash
# Incremental (fast -- only reparses changed files)
python3 tools/code-tree/code_tree.py --repo-root . --incremental -q

# Full rebuild (after major changes)
python3 tools/code-tree/code_tree.py --repo-root .
```

For a structured PR review workflow using these queries, see [copilot-instructions.md](../../.github/copilot-instructions.md#code-tree-impact-analysis).
