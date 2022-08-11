# GCP Resource Proto
The gcp_resource is a special parameter that if a component adopts it, the component can take advantage of better supports in Vertex Pipelines in the following ways
* Better UI experience. Vertex Pipelines UI can recognize this parameter, and provide a customized view of the resource's logs and status in the Pipeline console.
* Better cancellation. The resource will be automatically cancelled when the Pipeline is cancelled.
* More cost-effective execution. Supported by dataflow only. See [wait_gcp_resources](https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/experimental/wait_gcp_resources/component.yaml) for details.

## Installation

```shell
pip install -U google-cloud-pipeline-components
```

## Usage
To write a resource as an output parameter
```
from google_cloud_pipeline_components.experimental.proto.gcp_resources_pb2 import GcpResources
from google.protobuf.json_format import MessageToJson

dataflow_resources = GcpResources()
dr = dataflow_resources.resources.add()
dr.resource_type='DataflowJob'
dr.resource_uri='https://dataflow.googleapis.com/v1b3/projects/[your-project]/locations/us-east1/jobs/[dataflow-job-id]'

with open(gcp_resources, 'w') as f:
    f.write(MessageToJson(dataflow_resources))

```

To deserialize the resource
```
from google.protobuf.json_format import Parse

input_gcp_resources = Parse(payload, GcpResources())
# input_gcp_resources is ready to be used. For example, input_gcp_resources.resources
```


## Supported resource_type
You can set the resource_type with arbitrary string. But only the following types will have the benefits listed above.
This list will be expanded to support more types in the future.
* BatchPredictionJob
* BigQueryJob
* CustomJob
* DataflowJob
* HyperparameterTuningJob
* VertexLro
