# KFServing Component

This is the Kubeflow Pipelines component for KFServing. This uses the [V1beta1 API](https://github.com/kubeflow/kfserving/blob/master/docs/apis/v1beta1/README.md).

## Usage

Load the component with:

```python
import kfp.dsl as dsl
import kfp
from kfp import components

kfserving_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/kfserving/component.yaml')
```

**Note**: To use the previous version of this component which uses the v1alpha2 API and KFServing 0.4.1, then load the following YAML instead:

```yaml
https://raw.githubusercontent.com/kubeflow/pipelines/65bed9b6d1d676ef2d541a970d3edc0aee12400d/components/kubeflow/kfserving/component.yaml
```


### Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| action   | `create` | Action to execute on KFServing. Available options are `create`, `update`, `apply`, and `delete`. Note: `apply` is equivalent to `update` if the resource exists and `create` if not. |
| model_name |  | Name to give to the deployed model/InferenceService |
| model_uri  |  | Path of the S3 or GCS compatible directory containing the  model. |
| canary_traffic_percent | `100` | The traffic split percentage between the candidate model and the last ready model |
| namespace |  | Kubernetes namespace where the KFServing service is deployed. If no namespace is provided, `anonymous` will be used unless a namespace is provided in the `inferenceservice_yaml` argument. |
| framework |  | Machine learning framework for model serving. Currently the supported frameworks are  `tensorflow`, `pytorch`, `sklearn`, `xgboost`, `onnx`, `triton`, `pmml`, and `lightgbm`. |
| custom_model_spec | `{}` | Custom model runtime container spec in JSON. Sample spec: `{"image": "codait/max-object-detector", "port":5000, "name": "test-container"}` |
| inferenceservice_yaml | `{}` | Raw InferenceService serialized YAML for deployment. Use this if you need additional configurations for your InferenceService. |
| autoscaling_target | `0` | Autoscaling Target Number. If not 0, sets the following annotation on the InferenceService: `autoscaling.knative.dev/target` |
| service_account | | ServiceAccount to use to run the InferenceService pod. |
| enable_istio_sidecar | `True` | Whether to enable istio sidecar injection. |
| watch_timeouot | `300` | Timeout in seconds for watching until the InferenceService becomes ready. |
| min_replicas | `-1` | Minimum number of InferenceService replicas. Default of -1 just delegates to pod default of 1. |
| max_replicas | `-1` | Maximum number of InferenceService replicas. |


### Basic InferenceService Creation

The following will use the KFServing component to deploy a TensorFlow model.

```python
@dsl.pipeline(
  name='KFServing Pipeline',
  description='A pipeline for KFServing.'
)
def kfserving_pipeline():
    kfserving_op(
        action='apply',
        model_name='tf-sample',
        model_uri='gs://kfserving-samples/models/tensorflow/flowers',
        framework='tensorflow',
    )
kfp.Client().create_run_from_pipeline_func(kfserving_pipeline, arguments={})
```

Sample op for deploying a PyTorch model:

```python
kfserving_op(
    action='apply',
    model_name='pytorch-test',
    model_uri='gs://kfserving-examples/models/torchserve/image_classifier',
    framework='pytorch'
)
```

### Canary Rollout

Ensure you have an initial model deployed with 100 percent traffic with something like:

```python
kfserving_op(
    action = 'apply',
    model_name='tf-sample',
    model_uri='gs://kfserving-samples/models/tensorflow/flowers',
    framework='tensorflow',
)
```

Deploy the candidate model which will only get a portion of traffic:

```python
kfserving_op(
    action='apply',
    model_name='tf-sample',
    model_uri='gs://kfserving-samples/models/tensorflow/flowers-2',
    framework='tensorflow',
    canary_traffic_percent='10'
)
```

To promote the candidate model, you can either set `canary_traffic_percent` to `100` or simply remove it, then re-run the pipeline:

```python
kfserving_op(
    action='apply',
    model_name='tf-sample',
    model_uri='gs://kfserving-samples/models/tensorflow/flowers-2',
    framework='tensorflow'
)
```

If you instead want to rollback the candidate model, then set `canary_traffic_percent` to `0`, then re-run the pipeline:

```python
kfserving_op(
    action='apply',
    model_name='tf-sample',
    model_uri='gs://kfserving-samples/models/tensorflow/flowers-2',
    framework='tensorflow',
    canary_traffic_percent='0'
)
```

### Deletion

To delete a model, simply set the `action` to `'delete'` and pass in the InferenceService name:

```python
kfserving_op(
    action='delete',
    model_name='tf-sample'
)
```

### Custom Runtime

To pass in a custom model serving runtime, you can use the `custom_model_spec` argument. Currently,
the expected format for `custom_model_spec` coincides with:

```json
{
    "image": "some_image",
    "port": "port_number",
    "name": "custom-container",
    "env" : [{ "name": "some_name", "value": "some_value"}],
    "resources": { "requests": {},  "limits": {}}
}
```

Sample deployment:

```python
container_spec = '{ "image": "codait/max-object-detector", "port":5000, "name": "custom-container"}'
kfserving_op(
    action='apply',
    model_name='custom-simple',
    custom_model_spec=container_spec
)
```

### Deploy using InferenceService YAML

If you need more fine-grained configuration, there is the option to deploy using an InferenceService YAML file:

```python
isvc_yaml = '''
apiVersion: "serving.kubeflow.org/v1beta1"
kind: "InferenceService"
metadata:
  name: "sklearn-iris"
  namespace: "anonymous"
spec:
  predictor:
    sklearn:
      storageUri: "gs://kfserving-samples/models/sklearn/iris"
'''
kfserving_op(
    action='apply',
    inferenceservice_yaml=isvc_yaml
)
```


