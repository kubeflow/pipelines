# Run a Pipeline

## Run & View a KFP Pipeline in the MLflow Console

Follow the steps below to submit a pipeline run to the KFP API server and view the run in the MLflow console.

### Prerequisites

* KFP deployment with MLflow integration enabled and connected to a running MLflow server.
  (See the [operator guide](../../../operator-guides/mlflow-plugin.md) for instructions on MLflow deployment and KFP configuration.)
* An uploaded pipeline spec

### Add run-level MLflow config overrides

You can specify [run-level MLflow config overrides](overview.md#run-level-mlflow-config-overrides) on your pipeline run.
Note that run-level MLflow config overrides are not supported in the KFP UI, and to apply these configs, you must submit the run through a direct HTTP request to the KFP API server.
See the following example for correct formatting. Execute the request and skip ahead to step 2 below.

```shell
curl -X POST "http://localhost:3000/apis/v2beta1/runs" \
  -H "Authorization: <auth val>" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "my-run",
    "pipeline_version_reference": {
      "pipeline_id": "<id of uploaded pipeline spec>"
    },
    "plugins_input": {
      "mlflow": {
        "experiment_name": "my-override-experiment"
      }
    }
  }'
```

### Submit your pipeline run

1. Submit your pipeline run to the KFP API server. If you are applying run-level MLflow config overrides, use the cURL request above. Otherwise, you can use the KFP UI.
2. Track your pipeline run in the [MLflow console](https://mlflow.org/docs/latest/ml/tracking/). Click on your experiment, and a list of corresponding runs will appear.

### View individual pipeline component detail

You can access the details of individual pipeline components in the MLflow console. Click on the + icon next to the pipeline run name, and its subcomponents will appear.
Depending on the complexity of your [pipeline structure](pipeline-structure-in-mlflow.md), these subcomponents may expand further.
Click on a nested entry to view its logged input parameters and scalar metric artifacts.
