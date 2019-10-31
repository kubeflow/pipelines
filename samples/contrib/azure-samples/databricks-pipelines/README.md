# Introduction to Azure Databricks Operator pipeline samples

This folder contains several [Kubeflow Pipeline](https://www.kubeflow.org/docs/pipelines/) samples 
which show how to manipulate [Databricks](https://azure.microsoft.com/services/databricks/) 
resources using the [Azure Databricks Operator for Kubernetes](
https://github.com/microsoft/azure-databricks-operator). 

- databricks_operator_*.py samples show how to use [ResourceOp](
https://www.kubeflow.org/docs/pipelines/sdk/manipulate-resources/#resourceop) to manipulate
Databricks resources.
- databricks_pkg_*.py samples show how to use [Azure Databricks for Kubeflow Pipelines](
../kfp-azure-databricks/) 
package to manipulate Databricks resources.

## Setup

1) [Create an Azure Databricks workspace](
    https://docs.microsoft.com/en-us/azure/databricks/getting-started/try-databricks?toc=https%3A%2F%2Fdocs.microsoft.com%2Fen-us%2Fazure%2Fazure-databricks%2FTOC.json&bc=https%3A%2F%2Fdocs.microsoft.com%2Fen-us%2Fazure%2Fbread%2Ftoc.json#--step-2-create-an-azure-databricks-workspace)
2) [Deploy the Azure Databricks Operator for Kubernetes](
    https://github.com/microsoft/azure-databricks-operator/blob/master/docs/deploy.md)
3) All these samples reference 'sparkpi.jar' library. This library can be found here: [Create and run a 
jar job](https://docs.databricks.com/dev-tools/api/latest/examples.html#create-and-run-a-jar-job). 
Upload it to [Databricks File System](
https://docs.microsoft.com/en-us/azure/databricks/data/databricks-file-system) using e.g. [DBFS 
CLI](https://docs.microsoft.com/en-us/azure/databricks/dev-tools/databricks-cli#dbfs-cli).
4) [Install the Kubeflow Pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
5) Install Azure Databricks for Kubeflow Pipelines package:
```
pip install -e "git+https://github.com/kubeflow/pipelines#egg=kfp-azure-databricks&subdirectory=samples/contrib/azure-samples/kfp-azure-databricks" --upgrade
```
To uninstall use:
```
pip uninstall kfp-azure-databricks
```

