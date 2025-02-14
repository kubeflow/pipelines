import os
import kfp.compiler as compiler
import kfp.components as components
from kfp.azure import use_azure_secret
import kfp.dsl as dsl
import argparse

parser = argparse.ArgumentParser(description='Process inputs.')
parser.add_argument('--image_name', type=str, default='kubeflow_synapse_component')
parser.add_argument('--image_repo_name', type=str, default='kubeflowdemo')
args = parser.parse_args()

component_root = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".")
image_repo_name = args.image_repo_name # the container registery for the container operation and path in the ACR
image_name = args.image_name
file_path = os.path.join(component_root, "component.yaml")

# Loading the component.yaml file for deployment operation
run_job_operation = components.load_component_from_file(file_path)

# The run_job_image_name shall be the container image for the operation
# It shall be something like <your_acr_name>.azurecr.io/deploy/aml-deploy-model:latest
# If you are using a container registry other than Azure Container Registry, update the image name correspondingly
run_job_image_name = image_repo_name + '.azurecr.io/deploy/' + image_name + ':latest'

print(run_job_image_name)

def use_image(image_name):
    def _use_image(task):
        task.image = image_name
        return task
    return _use_image

@dsl.pipeline(
    name='Azure Synapse Component Sample',
    description='Run spark jobs in Azure Synapse'
)
def run_spark_job(
    main_definition_file,
    command_line_arguments
):
    operation = run_job_operation(executor_size='Small',
                                executors=1,
                                main_class_name='""',
                                main_definition_file=main_definition_file,
                                name='kubeflowsynapsetest',
                                tenant_id='$(AZ_TENANT_ID)',
                                service_principal_id='$(AZ_CLIENT_ID)',
                                service_principal_password='$(AZ_CLIENT_SECRET)',
                                subscription_id='$(AZ_SUBSCRIPTION_ID)',
                                resource_group='kubeflow-demo-rg',
                                command_line_arguments=command_line_arguments,
                                spark_pool_name='kubeflowsynapse',
                                language='',
                                reference_files='',
                                configuration='',
                                tags='',
                                spark_pool_config_file='./src/spark_pool_config.yaml',
                                wait_until_job_finished=True,
                                waiting_timeout_in_seconds=3600,
                                workspace_name='kubeflow-demo'). \
                                apply(use_azure_secret()). \
                                apply(use_image(run_job_image_name))

if __name__ == '__main__':
    compiler.Compiler().compile(run_spark_job,  __file__ + '.tar.gz')