from setuptools import setup

setup(
    name='kfp-azure-databricks',
    version='0.2.0',
    description='Python package to manage Azure Databricks on Kubeflow Pipelines using Azure Databricks operator for Kubernetes',
    url='https://github.com/kubeflow/pipelines/tree/master/samples/contrib/azure-samples/kfp-azure-databricks',
    packages=['databricks']
)
