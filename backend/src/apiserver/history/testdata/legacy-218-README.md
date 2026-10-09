# Release-2.18 archive fixtures

These are actual `transfer.Engine.Export` results, not JSON assembled from the
native importer's DTOs. Tests independently reproduce their SHA-256
digests before conversion.

- `legacy-218-v2-export.json`: release-2.18 base `20bb14761` with the
  `codex/transfer-218-runtime-parameters` exporter change; explicit nonempty run
  and schedule runtime parameter overrides, SQL-only historical task.
- `legacy-218-mlmd-v2-export.json`: same updated exporter through its generated
  MLMD protobuf RPC adapter, with a DAG root, child and cached source execution,
  one live artifact at an unchanged shared-bucket URI, and a named output port.

Generation uses the release package's `fixture`, `testDB`, `memoryMetadata`, and
`descriptorConn` test helpers. The installation UUID is fixed to
`f9f317ad-bc46-4ad8-8a96-c0495f004beb`. Both archived versions use the resource
package's `v2SpecHelloWorld`; the schedule has interval 3600 and service account
`pipeline-runner`. The v2 row overrides are `{"text":"run override","large_integer":9007199254740993}` and
`{"text":"schedule override"}`. The MLMD fixture adapts `lineageGraph("run")`:
execution 11 is `system.DAGExecution`, executions 10 and 12 have parent 11, all
three are associated with run context 5, every execution has `task_name` and
`namespace` custom properties, and the live artifact 20 has output port `model`.
The source run also contains persisted state history with a nonempty error.
No external database, Kubernetes cluster, object store, or MLMD server is needed
to consume the fixtures.
