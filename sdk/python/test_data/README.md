# KFP SDK Test Data

Test data in this directory is used for two separate test suites.

1. Read/write tests
> Location: `sdk/python/kfp/compiler/read_write_test.py `

Tests the compiler (write) and load (read) functionality of the SDK. This ensures that pipelines are written and read correctly and idempotently.

These tests require golden snapshots of pipelines and components against with compiled pipelines are compared. To update pipeline golden snapshots:

```bash
for f in sdk/python/test_data/pipelines/*.py ; do echo "$f" && python3 "$f" ; done
```


To update component golden snapshots:
```bash
for f in sdk/python/test_data/components/*.py ; do echo "$f" && python3 "$f" ; done
```


2. Pipeline execution tests
> Location: `test/sdk-execution-tests.py`

These tests ensure that the KFP OSS BE can execute the pipelines. `execute`: may be `false` in the `test_data_config.yaml` for a given `test_case` if the test case (a) isn't a complete example (e.g., a dependency doesn't exist in the image, etc.) or (b) the KFP OSS BE cannot execute the pipeline.
