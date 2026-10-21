## Shared backend integration tests

The v2 API integration suites are in [../v2/integration](../v2/integration).
This directory retains database initialization checks that are independent of
the removed KFP v1 API. Kubernetes pipeline-webhook coverage lives alongside
the v2 API integration suites.

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
