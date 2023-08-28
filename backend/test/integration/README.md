## Api Server Integration Tests

### WARNING
**These integration tests will delete all the data in your KFP instance, please only use a test cluster to run these.**

### How to run

The default integration test will test the default Database, MySQL.

1. Configure kubectl to connect to your kfp cluster.
2. Run the following for all integration tests: `NAMESPACE=<kfp-namespace> ./run_tests_locally.sh`.
3. Or run the following to select certain tests: `NAMESPACE=<kfp-namespace> ./run_tests_locally.sh -testify.m Job`.
   Reference: https://stackoverflow.com/a/43312451

### Run database tests with PostgreSQL

To run this test, you need to first deploy the PostgreSQL images on your Kubernetes cluster. For how to deploy, 
see [instructions here](../../../manifests/kustomize/third-party/postgresql/README.md).

When testing against postgreSQL, all integration tests with MySQL will be disabled. Use an argument `postgres` to run 
test against a PostgreSQL database:
```
NAMESPACE=<kfp-namespace> ./run_tests_locally.sh postgres
```


