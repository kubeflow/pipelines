# KFP Code-Tree Knowledge Graph

Pre-built knowledge graph artifacts, query commands, codebase statistics, entry points, hub files, module dependencies, inheritance hierarchies, and workflows for development and PR reviews.

**See also:** [architecture.md](architecture.md) (system architecture), [backend-guide.md](backend-guide.md) (backend files), [sdk-guide.md](sdk-guide.md) (SDK files)

---

## Available artifacts

| File | Contents | Use case |
|------|----------|----------|
| `docs/code-tree/summary.md` | Architecture overview, hub files, key types, inheritance, module deps | **Start here** for any exploration |
| `docs/code-tree/graph.json` | Full knowledge graph (24,901 nodes, 87,625 edges) | Programmatic queries with `jq` |
| `docs/code-tree/tags.json` | Flat symbol index (22,153 symbols with file:line locations) | Quick symbol lookup |
| `docs/code-tree/modules.json` | Directory-level dependency map (587 modules) | Module impact analysis |

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

- **2,748 files** (1,912 source + 836 test) across 7 languages: Python (1,343), Go (777), TypeScript (407), Bash (108), Protobuf (90), JavaScript (21), SQL (2)
- **22,153 symbols**: 11,431 methods, 6,374 functions, 1,696 classes, 1,280 structs, 765 interfaces, 490 constants, 117 enums
- **87,625 edges**: 37,371 imports, 22,153 contains, 19,397 calls, 7,873 tests, 831 inherits
- **904/1,912** source files have test coverage (47%)

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

When significant code changes occur, regenerate:

```bash
python3 tools/code-tree/code_tree.py --repo-root .
```

The query reference above with inline comments is self-explanatory. For a structured PR review workflow using these queries, see [copilot-instructions.md](../../.github/copilot-instructions.md#code-tree-impact-analysis).
