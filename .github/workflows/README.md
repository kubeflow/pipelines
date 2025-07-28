Many workflows have been disabled in this branch because they're not relevant to KFP v1 SDK. Most are self-explanatory. Those that are not will be justified in this document.

## [sdk-execution.yml](sdk-execution.yml)d
d
This test invokes [sdk_execution_tests.py](../../test/sdk-execution-tests/sdk_execution_tests.py) to run a set of test cases against a KFP cluster. The test cases are defined in [test_data_config.yaml](../../kubernetes_platform/python/test/snapshot/test_data_config.yaml). 

These tests only evaluate PVC creation and deletion in `kfp-kubernetes`, which is irrelevant to the v1 SDK. We should probably rename this workflow to more accurately represent what it's doing.



