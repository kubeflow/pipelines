## Api Server Integration Tests

### WARNING
**These integration tests will delete all the data in your KFP instance, please only use a test cluster to run these.**

### How to run

1. Configure kubectl to connect to your kfp cluster.
2. Run: `NAMESPACE=<kfp-namespace> ./run_tests_locally.sh`.
