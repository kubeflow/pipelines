# Driver retry runtime validation

The `E2EDriverRetry` Ginkgo label runs against a real KFP deployment and Argo
controller. CI runs it with Argo 4.0/MySQL and Argo 4.1/PostgreSQL. It requires a
dedicated deployment: the serial specs temporarily replace its driver plugin
ConfigMap image, and restore it during cleanup. After each change, they roll
the `workflow-controller` Deployment in the test namespace and wait for all
replacement replicas to be ready and all old controller pods to disappear.
This ensures Argo loads the intended image before creating any agent pod.
The cluster credentials therefore need permission to patch that Deployment.

The test-only `faultdriver` links the production authenticated executor-plugin
handler and uses its real Kubernetes and KFP API clients. It changes only the
HTTP response boundary after a successful production driver invocation:

- `restart-driver` records the committed response in the agent's shared volume,
  then exits with status 86 before returning it to Argo. Kubernetes restarts the
  plugin container, erasing its in-memory request cache. The test verifies the
  actual restart, successful replay, stable native task/output identity,
  preserved inputs, and exactly one component execution. A restart that outlasts
  Argo's HTTP retry may consume a configured infrastructure-error attempt.
- `exhaust-driver` reports a failed response after each committed driver handoff
  in generation 0. The test verifies all three Argo attempts, the finalizer,
  persisted task failure and the cleanup's `FAILED` input **while the workflow
  remains running**. It then calls the public RetryRun API and verifies that
  generation 1 succeeds with the original native task identity.

The marker and injected failures are confined to this binary. Production driver
images contain neither fault switches nor the wrapper. Pipeline components use
Python's standard library and need no SDK installation.

To run on a dedicated Kind KFP deployment from the current source:

```bash
context=$(mktemp -d)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -o "$context/driver-retry-test" ./backend/test/driverretry/faultdriver
docker build -f backend/test/driverretry/Dockerfile \
  -t kfp-driver-retry-test:local "$context"
kind --name kfp load docker-image kfp-driver-retry-test:local
KFP_DRIVER_RETRY_TEST_IMAGE=kfp-driver-retry-test:local \
  ginkgo -v --label-filter=E2EDriverRetry ./backend/test/end2end -- -namespace=kubeflow
```

Use the architecture of the cluster nodes for `GOARCH`. The usual E2E API
connection configuration and Kubernetes credentials are required. The suite
fails when the test image is not specified, rather than silently omitting the
fault injection. Keep the Ginkgo reports and workflow/pod logs for evidence.
