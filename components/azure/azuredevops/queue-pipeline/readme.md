# Queue Pipeline Task

This task enables you to queue an [Azure Pipelines](https://docs.microsoft.com/en-us/azure/devops/pipelines/?view=azure-devops) pipeline from a Kubeflow pipeline. For example, this task may be used to queue the deployment of a model via Azure Pipelines after the model is trained and registered by the Kubeflow pipeline.

## Inputs

|Name|Type|Required|Description|
|---|---|---|---|
|organization|string|Y|The Azure DevOps organization that contains the pipeline to be queued. https[]()://dev.azure.com/`organization`/project/_build?definitionId=id|
|project|string|Y|The Azure DevOps project that contains the pipeline to be queued. https[]()://dev.azure.com/organization/`project`/_build?definitionId=id|
|id|string|Y|The id of the pipeline definition to queue. Shown in the url as *pipelineId* or *definitionId*. https[]()://dev.azure.com/organization/project/_build?definitionId=`id`|
|pat_env|string|one of pat_env or pat_path_env|The name of the environment variable containing the PAT for Azure Pipelines authentication|
|pat_path_env|string|one of pat_env or pat_path_env|The name of the environment variable containing a path to the PAT for Azure Pipelines authentication|
|sourch_branch|string||The branch of the source code for queuing the pipeline.|
|source_version|string||The version (e.g. commit id) of the source code for queuing the pipeline.|
|parameters|string||Json serialized string of key-values pairs e.g. `{ 'x': '1', 'y': '2' }`. These values can be accessed as `$(x)` and `$(y)` in the Azure Pipelines pipeline.|

## Outputs

Output `output_url_path` holds uri for the newly queued pipeline.

## Usage

```python
import os
import kfp.compiler as compiler
import kfp.components as components
from kfp.azure import use_azure_secret
from kubernetes.client.models import V1EnvVar


component_root = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".")
image_repo_name = "<ACR_NAME>.azurecr.io/myrepo"
queue_pipeline_op = components.load_component_from_file(os.path.join(component_root, 'queue-pipeline\component.yaml'))
queue_pipeline_image_name = image_repo_name + '/queue_pipeline:%s' % ('latest')
secret_name = "azdopat"
secret_path = "/app/secrets"
pat_path_env = "PAT_PATH"
secret_file_path_in_volume = "azdopat"
organization = # organization
project = # project
pipeline_id = # id

def use_image(image_name):
    def _use_image(task):
        task.image = image_name
        return task
    return _use_image

@dsl.pipeline(
    name='Azure Sample',
    description='Queue Azure DevOps pipeline '
)
def azdo_sample():

    operations['Queue AzDO pipeline'] = queue_pipeline_op(
                                        organization=organization,
                                        project=project,
                                        id=pipeline_id,
                                        pat_path_env=pat_path_env). \
                                        apply(use_secret(secret_name=secret_name, secret_volume_mount_path=secret_path). \
                                        apply(use_azure_secret()). \
                                        apply(use_image(queue_pipeline_image_name)). \
                                        add_env_variable(V1EnvVar(
                                                            name=pat_path_env,
                                                            value=secret_path + "/" + secret_file_path_in_volume))

if __name__ == '__main__':
    compiler.Compiler().compile(azdo_sample,  __file__ + '.tar.gz')
```
