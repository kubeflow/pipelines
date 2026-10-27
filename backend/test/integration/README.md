## Shared backend integration tests

The API suites run in [../v2/api](../v2/api). The retired legacy API harness
has been removed; [../v2/integration](../v2/integration) retains only Kubernetes
pipeline-webhook tests and their Makefile. This directory retains independent
database initialization checks.

### Webhook integration tests

Deploy the Kubernetes-native Kubeflow Pipelines environment with its webhooks,
then run from the repository root:

```sh
make -C backend/test/v2/integration test-webhook
```

The target sets `WEBHOOK_INTEGRATION=true`. Without it, webhook tests are skipped.
These checks use the `pipelines.kubeflow.org/v2beta1` Pipeline and PipelineVersion
CRDs.

### Database initialization

Use an isolated test database, reachable at localhost through port forwarding.
These checks initialize the schema; do not point them at a production database.

```sh
go test ./backend/test/integration -run '^TestDB$' -runIntegrationTests
# PostgreSQL instead of MySQL:
go test ./backend/test/integration -run '^TestDB$' -runIntegrationTests -runPostgreSQLTests
```

Database tests are skipped unless `-runIntegrationTests` is supplied.
