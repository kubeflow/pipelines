# Periodic mysql Run Cleanup

**Problem:** [kubeflow-pipelines-samples-v2](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/99f29a66aa2c29128099edb7b1a99445ac0633bb/prow/prowjobs/kubeflow/pipelines/kubeflow-pipelines-presubmits.yaml#L153-L164) tests have been flaky, with spurious timeout and failed network connection failures.

**Explanation:** The underlying source of the failure is overloading of mysql. Repeated polling for run status for many runs executed in parallel combined with slow queries due to a large run history results in failed connections to mysql.

**Solution:** The solution to this problem is deleting old runs in order to speed up query execution time. In order to maintain internal consistency of the database, the run cache must also be cleared, otherwise a cache may hit but the run record may not be found.

**Executing the script:**
The following script will delete all runs created before `DATE_THRESHOLD` as defined in the `delete_old_runs.py` script.

First, set the `kubectl` context to the `kfp-ci` test cluster `gke_kfp-ci_us-central1_kfp-standalone-1` associated with the sample test KFP deployment. Then, run the following script:
```
source ./test/tools/clean-mysql-runs/clean-mysql-runs.sh
```
