## Api Server Integration Tests

### WARNING
**These integration tests will delete all the data in your KFP instance, please only use a test cluster to run these.**

### How to run

1. Configure kubectl to connect to your kfp cluster.
2. Run the following for all integration tests: `NAMESPACE=<kfp-namespace> ./run_tests_locally.sh`.
3. Or run the following to select certain tests: `NAMESPACE=<kfp-namespace> ./run_tests_locally.sh -testify.m Job`.
   Reference: https://stackoverflow.com/a/43312451
