# Amazon SageMaker Components for Kubeflow Pipelines

## Summary
With Amazon SageMaker Components for Kubeflow Pipelines (KFP), you can create and monitor training, tuning, endpoint deployment, and batch transform jobs in Amazon SageMaker. By running Kubeflow Pipeline jobs on Amazon SageMaker, you move data processing and training jobs from the Kubernetes cluster to Amazon SageMaker’s machine learning-optimized managed service. The job parameters, status, logs, and outputs from Amazon SageMaker are still accessible from the Kubeflow Pipelines UI.


## Components
Amazon SageMaker Components for Kubeflow Pipelines offer an alternative to launching compute-intensive jobs in Kubernetes and integrate the orchestration benefits of Kubeflow Pipelines. The following Amazon SageMaker components have been created to integrate 6 key Amazon SageMaker features into your ML workflows. You can create a Kubeflow Pipeline built entirely using these components, or integrate individual components into your workflow as needed. 

There is no additional charge for using Amazon SageMaker Components for Kubeflow Pipelines. You incur charges for any Amazon SageMaker resources you use through these components.

### Training components

#### Training

The Training component allows you to submit Amazon SageMaker Training jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Training Kubeflow Pipelines component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train).

#### Hyperparameter Optimization

The Hyperparameter Optimization component enables you to submit hyperparameter tuning jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Hyperparameter Optimization Kubeflow Pipeline component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/hyperparameter_tuning).

#### Processing

The Processing component enables you to submit processing jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Processing Kubeflow Pipeline component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/process).


### Inference components

#### Hosting Deploy

The Deploy component enables you to deploy a model in Amazon SageMaker Hosting from a Kubeflow Pipelines workflow. For more information, see [SageMaker Hosting Services - Create Endpoint Kubeflow Pipeline component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy).

#### Batch Transform component

The Batch Transform component enables you to run inference jobs for an entire dataset in Amazon SageMaker from a Kubeflow Pipelines workflow. For more information, see [SageMaker Batch Transform Kubeflow Pipeline component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform).


### Ground Truth components

#### Ground Truth 

The Ground Truth component enables you to to submit Amazon SageMaker Ground Truth labeling jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Ground Truth Kubeflow Pipelines component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/ground_truth).

#### Workteam

The Workteam component enables you to create Amazon SageMaker private workteam jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker create private workteam Kubeflow Pipelines component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/workteam).


