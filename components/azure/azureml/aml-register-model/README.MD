# Azure ML Register Model

This component registers trained ML model with [Azure Machine Learning Service](https://azure.microsoft.com/en-us/services/machine-learning/#documentation) and tags the model with Kubeflow run id.

## Inputs

| Name                       | Type   | Required | Description                      |
| -------------------------- | ------ | -------- | -------------------------------- |
| base_path                  | string | Y        | Base path to the model           |
| model_file                 | string | Y        | Model file for example .pkl      |
| model_name                 | string | Y        | Model Name                       |
| tenant_id                  | string | Y        | Azure AD tenant Id               |
| service_principal_id       | string | Y        | Azure service principal Id       |
| service_principal_password | string | Y        | Azure service principal password |
| subscription_id            | string | Y        | Azure subscription Id            |
| resource_group             | string | Y        | Azure resource group             |
| workspace                  | string | Y        | Azure ML workspace name          |
| run_id                     | string | Y        | Kubeflow pipeline run Id         |

## Outputs

Output `output_model_path` holds registered model name.

## Usage

```
import os
import kfp.compiler as compiler
import kfp.components as components
from kfp.azure import use_azure_secret

component_root = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".")
image_repo_name = "[ACR_NAME].azurecr.io/myrepo"
register_op = components.load_component_from_file(os.path.join(component_root, 'aml-register-model/component.yaml'))
register_image_name = image_repo_name + '/aml-register-model:%s' % ('latest')

def use_image(image_name):
    def _use_image(task):
        task.image = image_name
        return task
    return _use_image

@dsl.pipeline(
    name='AML Sample ',
    description='AML Register Model'
)
def model_train(
    resource_group,
    workspace,
):

    operations['register to AML'] = register_op(base_path='path_to_model',
                                                model_file='latest.h5',
                                                model_name='model_name',
                                                tenant_id='$(AZ_TENANT_ID)',
                                                service_principal_id='$(AZ_CLIENT_ID)',
                                                service_principal_password='$(AZ_CLIENT_SECRET)',
                                                subscription_id='$(AZ_SUBSCRIPTION_ID)',
                                                resource_group=resource_group,
                                                workspace=workspace,
                                                run_id=dsl.RUN_ID_PLACEHOLDER). \
                                                apply(use_azure_secret()). \
                                                apply(use_image(register_image_name))

if __name__ == '__main__':
    compiler.Compiler().compile(model_train,  __file__ + '.tar.gz')
```
