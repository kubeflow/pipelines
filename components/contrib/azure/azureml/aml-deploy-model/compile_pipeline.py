import os
import kfp.compiler as compiler
import kfp.components as components
from kfp.azure import use_azure_secret
import kfp.dsl as dsl

component_root = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".")
image_repo_name = "<your_acr_name>.azurecr.io/deploy" # the container registery for the container operation and path in the ACR
file_path = os.path.join(component_root, "component.yaml")

# Loading the component.yaml file for deployment operation
deploy_operation = components.load_component_from_file(file_path)

# The deploy_image_name shall be the container image for the operation
# It shall be something like <your_acr_name>.azurecr.io/deploy/aml-deploy-model:latest
deploy_image_name = image_repo_name + '/aml-deploy-model:%s' % ('latest')

def use_image(image_name):
    def _use_image(task):
        task.image = image_name
        return task
    return _use_image

@dsl.pipeline(
    name='AML Component Sample',
    description='Deploy Model using Azure Machine learning'
)
def model_deploy(
    resource_group,
    workspace
):

    operation = deploy_operation(deployment_name='deploymentname',
                                model_name='model_name:1',
                                tenant_id='$(AZ_TENANT_ID)',
                                service_principal_id='$(AZ_CLIENT_ID)',
                                service_principal_password='$(AZ_CLIENT_SECRET)',
                                subscription_id='$(AZ_SUBSCRIPTION_ID)',
                                resource_group=resource_group,
                                workspace=workspace,
                                inference_config='src/inferenceconfig.json',
                                deployment_config='src/deploymentconfig.json'). \
                                apply(use_azure_secret()). \
                                apply(use_image(deploy_image_name))

if __name__ == '__main__':
    compiler.Compiler().compile(model_deploy,  __file__ + '.tar.gz')