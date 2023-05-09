# Amazon SageMaker Components for Kubeflow Pipelines

## Summary
With Amazon SageMaker Components for Kubeflow Pipelines (KFP), you can create and monitor training, tuning, endpoint deployment, and batch transform jobs in Amazon SageMaker. By running Kubeflow Pipeline jobs on Amazon SageMaker, you move data processing and training jobs from the Kubernetes cluster to Amazon SageMaker’s machine learning-optimized managed service. The job parameters, status, and outputs from Amazon SageMaker are still accessible from the Kubeflow Pipelines UI.

## Components
Amazon SageMaker Components for Kubeflow Pipelines offer an alternative to launching compute-intensive jobs in Kubernetes and integrate the orchestration benefits of Kubeflow Pipelines. The following Amazon SageMaker components have been created to integrate 6 key Amazon SageMaker features into your ML workflows. You can create a Kubeflow Pipeline built entirely using these components, or integrate individual components into your workflow as needed. 

For an end-to-end tutorial using these components, see [Using Amazon SageMaker Components](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-tutorials.html).

For more example pipelines, see [Sample AWS SageMaker Kubeflow Pipelines](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples).

There is no additional charge for using Amazon SageMaker Components for Kubeflow Pipelines. You incur charges for any Amazon SageMaker resources you use through these components.

## Versioning
The version of the AWS SageMaker Components is determined by the docker image tag used in the component specification YAML file.

### Training components

#### Training

The Training component allows you to submit Amazon SageMaker Training jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Training Kubeflow Pipelines component version 2](./TrainingJob). 

For more information about Version 1 of Training component see [SageMaker Training Kubeflow Pipelines component version 1](./train).

#### RLEstimator

The RLEstimator component allows you to submit RLEstimator (Reinforcement Learning) SageMaker Training jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker RLEstimator Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/rlestimator).

#### Hyperparameter Optimization

The Hyperparameter Optimization component enables you to submit hyperparameter tuning jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Hyperparameter Optimization Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/hyperparameter_tuning).

#### Processing

The Processing component enables you to submit processing jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Processing Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/process).


### Inference components

#### Hosting Deploy

The Hosting component allows you to submit Amazon SageMaker Hosting deployments directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Endpoint Kubeflow Pipelines component version 2](./Endpoint), [SageMaker Endpoint Config Kubeflow Pipelines component version 2](./EndpointConfig), [SageMaker Model Kubeflow Pipelines component version 2](./Model)

For more information about Version 1 of Hosting components see [SageMaker Hosting Services - Create Endpoint Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy).

#### Batch Transform component

The Batch Transform component enables you to run inference jobs for an entire dataset in Amazon SageMaker from a Kubeflow Pipelines workflow. For more information, see [SageMaker Batch Transform Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform).

### Model Monitor components

The Model Monitor components allow you to setup Amazon SageMaker monitoring job definition and schedule directly from a Kubeflow Pipelines workflow.

#### Monitoring Job Definition

The Monitoring Job Definition components allow you to create a monitoring job definition that can be used to create a monitoring schedule directly from a Kubeflow Pipelines workflow. For more information, see:

- [SageMaker Data Quality Job Definition Kubeflow Pipelines component version 2](./DataQualityJobDefinition)
- [SageMaker Model Bias Job Definition Kubeflow Pipelines component version 2](./ModelBiasJobDefinition)
- [SageMaker Model Explainability Job Definition Kubeflow Pipelines component version 2](./ModelExplainabilityJobDefinition)
- [SageMaker Model Quality Job Definition Kubeflow Pipelines component version 2](./ModelQualityJobDefinition)

#### Monitoring Schedule

Monitoring Schedule component to create a monitoring schedule that regularly starts Amazon SageMaker Processing Jobs to monitor the data captured for an Amazon SageMaker Endpoint, see [SageMaker Monitoring Schedule Kubeflow Pipelines component version 2](./MonitoringSchedule).


### Ground Truth components

#### Ground Truth 

The Ground Truth component enables you to to submit Amazon SageMaker Ground Truth labeling jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker Ground Truth Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/ground_truth).

#### Workteam

The Workteam component enables you to create Amazon SageMaker private workteam jobs directly from a Kubeflow Pipelines workflow. For more information, see [SageMaker create private workteam Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/workteam).


### RoboMaker components

#### Create Simulation Application

The Create Simulation Application component allows you to create a RoboMaker Simulation Application directly from a Kubeflow Pipelines workflow. For more information, see [RoboMaker Create Simulation app Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/create_simulation_app).

#### Simulation Job

The Simulation Job component allows you to run a RoboMaker Simulation Job directly from a Kubeflow Pipelines workflow. For more information, see [RoboMaker Simulation Job Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job).

#### Simulation Job Batch

The Simulation Job Batch component allows you to run a RoboMaker Simulation Job Batch directly from a Kubeflow Pipelines workflow. For more information, see [RoboMaker Simulation Job Batch Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job_batch).

#### Delete Simulation Application

The Delete Simulation Application component allows you to delete a RoboMaker Simulation Application directly from a Kubeflow Pipelines workflow. For more information, see [RoboMaker Delete Simulation app Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/delete_simulation_app).
