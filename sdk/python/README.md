# KFP native authoring SDK

This fork captures the KFP Python DSL and compiles native Argo WorkflowTemplates,
Workflows and CronWorkflows. It includes the in-container Python bridge and the
shared authoring/Kubernetes schema bindings. KFP REST clients, the old pipeline IR
builder, server-backed management commands and in-process runners are removed.

## Build and install

Use Python 3.11 or newer. From the repository root:

```sh
uv build --package kfp --wheel
python -m pip install dist/kfp-*.whl
```

The wheel contains generated bindings and offline viewer assets. Consumers do not
need protoc or Node. Maintainers regenerate bindings with `make generate-python`
and viewer assets with `npm --prefix frontend run build:viewer`.

## Compile, preview and build components

```sh
kfp dsl compile --py pipeline.py --output pipeline.yaml --disable-caching
kfp visualize pipeline.py --format both -o pipeline.html
kfp visualize pipeline.py --watch
kfp component build --help
```

`kfp visualize` accepts trusted Python or native Argo YAML/JSON. It embeds the
shared graph, artifact cards, nested DAGs and JSON inspector in self-contained
HTML. Markdown contains Mermaid diagrams. Native previews have a 10,000-element
budget; loops are not unrolled. No run is submitted. Python previews disable
caching because deployment image identities are not required for graph inspection.

Only compile trusted Python: decorators execute module-level source while capturing
the graph. Watch mode preserves the previous exports on errors. `--serve` optionally
serves a loopback-scoped viewer; use `--no-open` to avoid opening a browser. Plain
exports and watch-only mode do not serve content or launch a browser.

Portable container component definitions can still be loaded with `kfp.components`
and exported with `component_spec.save_to_component_yaml`. The small definition
serializer uses the shared schema but never builds execution DAG IR or contacts a
backend. Compiled pipeline IR and KFP API resources are not accepted as component
definitions; compile their original Python source to Argo instead.

Use Kubernetes/Argo clients and the native UI for submission and management. See
`samples/argo-native/README.md` and `docs/compatibility.md` in the repository for the
deployed workflow, immutable image snapshots, tests and feature boundaries.
