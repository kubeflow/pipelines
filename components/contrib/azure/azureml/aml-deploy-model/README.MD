# Azure ML Deploy Model

This component deploys trained model with [Azure Machine Learning Service](https://azure.microsoft.com/en-us/services/machine-learning/#documentation). In Azure Machine Learning Service, trained machine learning models could be deployed as web services in the cloud or locally. You can use your model in web services for inferencing with improved quality. See more on [MLOps: Model management, deployment, and monitoring with Azure Machine Learning](https://docs.microsoft.com/en-us/azure/machine-learning/concept-model-management-and-deployment)

This `Azure ML Deploy Model` component, is a self-contained set of code to perform model deployment operation using Azure Machine Learning service.

It performs the following workflow:
- Prepare an inference configuration.
- Prepare an entry script.
- Choose a compute target. 
- Deploy the model to the compute target.

## Inputs

| Name                       | Type   | Required | Description                             |
| -------------------------- | ------ | -------- | ----------------------------------------|
| deployment_name            | string | Y        | Name for the deployed model endpoint    |
| model_name                 | string | Y        | Name for registered model and version   |
| inference_config           | string | Y        | File Path for inferenceconfig.json      |
| deployment_config          | string | Y        | File Path for deploymentconfig.json     |
| tenant_id                  | string | Y        | Azure AD tenant Id                      |
| service_principal_id       | string | Y        | Azure service principal (client) Id     |
| service_principal_password | string | Y        | Azure service principal password        |
| subscription_id            | string | Y        | Azure subscription Id                   |
| resource_group             | string | Y        | Azure resource group                    |
| workspace                  | string | Y        | Azure ML workspace name                 |

## Outputs
Output `output_config` holds description of the deployed web service for the machine learning model.
Output `score_uri` holds endpoint for the deployed model.

## Prerequisites

Ensure Kubeflow has been installed on AKS following [instructions for deploying Kubeflow on Azure](https://www.kubeflow.org/docs/azure/).

Azure Machine Learning service will be used to deploy the model, store the metadata and manage the active deployment. To create a Machine Learning Service workspace: 1. Go to [the Azure portal](https://portal.azure.com) and click on your resource group. 2. Select the **add a new resource** option. 3. Search for **Machine Learning Studio Workspace** and use the default options, taking note of the name you decide for it.

You need to create a [container registry](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-intro) to store those images in the cloud so that Kubeflow can pull the images as they are needed. [See here](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-get-started-portal) for step-by-step guidance to create an Azure Container Registry. Please ensure enable **admin user**, and change the SKU option to **Premium**.

In order for the AKS cluster to have access to pulling images created for execution of the pipeline, you will need to [update](https://docs.microsoft.com/en-us/azure/aks/cluster-container-registry-integration) your cluster so that it is able to pull the images from the container registry we just created by running:

```shell
az aks update -n {myAKSCluster} -g {MyResourceGroup} --attach-acr {REGISTRY_NAME}
```

A service principal is used to allow your pipeline to securely interface with your Azure services without having to directly login in the pipeline and use admin privileges. See [here](https://docs.microsoft.com/en-us/powerapps/developer/common-data-service/walkthrough-register-app-azure-active-directory) for a detailed guide to create a service principal. Also please ensure to add a **Contributor** role to the service principal to be able access Azure resources.

## Usage
In `./src` folder, adjust pre-configured `inferenceconfig.json`, `deploymentconfig.json`, and `environment.yml` to specify the resources for deployment like the compute type, number of cores etc. Additionaly, `deploymentconfig_aks.json` could be specified from parameters to use Azure Kubernetes Service to deploy the web service. `score.py` is the entry script for inference, where `init()` and `run(data)` are the functions to be *edited* to use the model. See here for [a detailed instruction on constructing the entry script](https://docs.microsoft.com/en-us/azure/machine-learning/how-to-deploy-and-where?tabs=azcli). See [Deploy your existing model with Azure](https://docs.microsoft.com/en-us/azure/machine-learning/how-to-deploy-existing-model#entry-script-scorepy) for an example of `score.py`. If additional pip packages are used for `score.py`, please ensure them being added to `environment.yml`. In the end, build and push the image to the Container Registry, the pipeline operation will directly pull the image from ACR.

A sample script to build and push the image:

```shell
docker build . -t {your_ACR_name}.azurecr.io/deploy/{your_image_name}:latest
docker push {your_ACR_name}.azurecr.io/deploy/{your_image_name}:latest
```


An example usage for the component is provided in the `compile_pipeline.py` file. It is a single-operation pipeline to use this component and `use_azure_secret` function from `kfp.azure`. You can also provide service principal parameters as pipeline parameters. 

```
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
                            inference_config='scripts/inferenceconfig.json',
                            deployment_config='scripts/deploymentconfig.json'). \
                            apply(use_azure_secret()). \
                            apply(use_image(deploy_image_name))

if __name__ == '__main__':
    compiler.Compiler().compile(model_deploy,  __file__ + '.tar.gz')
```

**Note:** The deployment name requires alphanumeric characters and the model_name shall be a combination of "model name" and "version number".

Once the deployment has finished, navigate to your Azure Machine Learning Workspace and select the `Endpoints` tab to find your Real-Time endpoints. You can access your model as a web-service by the `REST endpoint`. It should look like "http://{your_rest_endpoint}.{region}.azurecontainer.io/score?{your_data}". Alternatively, the output `score_uri` should also contain the information for the model endpoint. The `output_config` contains detailed information for the deployed web service.