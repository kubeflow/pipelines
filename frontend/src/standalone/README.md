# Offline native pipeline viewer

`kfp visualize` sends a validated Argo Workflow/WorkflowTemplate/CronWorkflow to
this read-only host. `ArgoGraph` projects native templates directly; the shared
`PipelineGraph`/`DagCanvas` widgets supply nested groups, artifact cards, selection,
controls and the native JSON inspector. No pipeline IR or REST client is loaded.

From `frontend/`, run `npm run build:viewer` and commit the generated
`sdk/python/kfp/cli/viewer_assets/` files. Licenses, scripts and styles are embedded
in exported HTML; neither Node nor a CDN is needed by SDK users.

`npm run test:viewer` exercises offline and sandboxed HTML, nested graphs,
selection/zoom retention, imported-file updates and recovery from invalid Python.
The watcher serves only loopback-scoped viewer/state URLs and preserves the last
valid exports after compilation or staging errors.
