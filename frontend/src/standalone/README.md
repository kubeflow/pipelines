# Standalone pipeline viewer

The `kfp visualize` CLI exports a self-contained HTML document. This directory
contains its thin HTML host. It uses the same `components/graph/PipelineGraph`
and `pages/v2/DagCanvas` as the application's pipeline-details page, without a
KFP API server, router, or authentication state.

## Screenshots

![Standalone graph with inline nested DAGs and artifact nodes](../../../docs/images/standalone-viewer-graph.png)

![Shared KFP node inspector showing model output and container details](../../../docs/images/standalone-viewer-inspector.png)

## Rebuild the SDK assets

From `frontend/`, using `.nvmrc` and the npm version in `package.json`:

```sh
npm ci
npm run build:viewer
```

Commit the generated `sdk/python/kfp/cli/viewer_assets/` files alongside renderer
changes. They are included in the Python wheel/sdist so SDK users do not need
Node or a CDN. Vite emits third-party licenses, which are embedded in exports.
The checked-in frontend lockfile pins the renderer's dependencies; rebuilding
these assets is required after upgrading those dependencies.

The `sdk-viewer-assets` pre-commit hook rebuilds on frontend or bundled-asset
changes and fails if generated files change. Stage the regenerated assets and
retry the commit. Use the pinned Node/npm versions and rerun `npm ci` after
lockfile changes; the hook uses installed dependencies rather than installing
them on every commit. Pre-commit CI installs the pinned toolchain and locked
dependencies before running the same hook. SDK users still need neither Node
nor npm. Existing HTML exports must be regenerated after upgrading the SDK.

```sh
# From frontend/
npm run test:ui -- src/standalone/Viewer.test.tsx
npm run typecheck

# Real browser + Python CLI: offline HTML, sandboxed iframe, both watch formats,
# preserved zoom/selection, imported-file changes, and compile-error recovery.
# Requires the repository .venv (or KFP_PYTHON=/path/to/python).
npx playwright install chromium
npm run test:viewer

# From the repository root
uv run pytest sdk/python/kfp/cli/visualize_test.py
```

## Watch and serve

`kfp visualize pipeline.py -w` updates the HTML artifact without a server.
`--format both -w` updates both HTML and Mermaid Markdown from one compilation.
Add `-s` / `--serve` to opt into the loopback HTML viewer; use `-w -s` for live
updates, or `-s` alone to serve a snapshot. Serving opens the default browser
once; pass `--no-open` / `-n` to suppress the launch while keeping the server and
printed URL. The browser smoke test uses `-n` and drives its own headless browser.
Serving is available with any export format. HTML is the default; `html`, `both`,
and `markdown` are supported.

## Data and updates

The CLI sends the validated **full PipelineSpec**, including deployment details,
to the HTML viewer. The existing `StaticFlow`/`GroupedFlow` modules convert it;
`PipelineGraph` supplies the shared canvas and `StaticNodeDetailsV2` inspector.
Task/artifact cards, inline group expansion, graph controls, and the MUI theme
are the application's own modules, not standalone copies. The bundle includes
the application's generated Tailwind styles and matches its 13px root font.

Python's `kfp.cli.visualization.validate_spec` checks each reachable DAG
component definition once, including task display names, producer references,
and cycles. It also requires the `deploymentSpec` field (an empty object is
allowed), matching the shared graph/inspector's contract. Incomplete references
or missing deployment data fail before files or served state are replaced.
HTML never expands reusable-component instance paths. `build_view` validates
and projects those paths only for Mermaid Markdown, with a budget of 10,000
DAGs, tasks, and dependency edges in total to prevent exponential output growth.
Exceeding the budget reports an error and leaves the previous export intact;
use HTML for heavily reused pipelines.

The HTML embeds `{state, live}` as escaped JSON. Offline artifacts and served
snapshots never fetch. Only a page served with both `--watch` and `--serve`
polls `./state`, using revisions to avoid re-layout on
unchanged polls. A changed graph retains its canvas instance, zoom, and selected
task, as well as inline expansion state. The inspector resolves its selection
against the latest spec, updating renamed nodes and closing for deleted nodes.
No persistent graph identity is supplied, so opaque-origin HTML previews do not
need access to local storage. Errors retain the last valid view.

The renderer is read-only: dragging nodes does not modify a pipeline. It does
not display execution status, expand runtime loop iterations, or provide
source-code editing.
