# Amazon SageMaker Components for Kubeflow Pipelines

## Summary
With Amazon SageMaker Components for Kubeflow Pipelines (KFP), you can create and monitor training, tuning, endpoint deployment, and batch transform jobs in Amazon SageMaker. By running Kubeflow Pipeline jobs on Amazon SageMaker, you move data processing and training jobs from the Kubernetes cluster to Amazon SageMaker’s machine learning-optimized managed service. The job parameters, status, and outputs from Amazon SageMaker are still accessible from the Kubeflow Pipelines UI.

## Components
Amazon SageMaker Components for Kubeflow Pipelines offer an alternative to launching compute-intensive jobs in Kubernetes and integrate the orchestration benefits of Kubeflow Pipelines. The following Amazon SageMaker components have been created to integrate key SageMaker features into your ML workflows from preparing data, to building, training, and deploying ML models. You can create a Kubeflow Pipeline built entirely using these components, or integrate individual components into your workflow as needed. The components are available in one or two versions. Each version of a component leverages a different backend. For more information on those versions, see [SageMaker Components for Kubeflow Pipelines versions](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#sagemaker-components-versions). 

For an end-to-end tutorial using these components, see [Using Amazon SageMaker Components](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-tutorials.html).

For more example pipelines, see [Sample AWS SageMaker Kubeflow Pipelines](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples).

There is no additional charge for using Amazon SageMaker Components for Kubeflow Pipelines. You incur charges for any Amazon SageMaker resources you use through these components.

### List of Components

> **_Note:_**  We encourage users to utilize Version 2 of a SageMaker component wherever it is available. You can find the version of an AWS SageMaker Components in the docker image tag used in the component specification file *component.yaml*.

<details><summary><b>Ground Truth components</b></summary>
<p>

* **Ground Truth**
  The Ground Truth component enables you to submit SageMaker Ground Truth labeling jobs directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker Ground Truth Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/ground_truth)    | X     |

* **Workteam**
  The Workteam component enables you to create Amazon SageMaker private workteam jobs directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker create private workteam Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/workteam) | X     |

</p></details>
<details><summary><b>Data processing components</b></summary>
<p>

* **Processing**
  The Processing component enables you to submit processing jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker Processing Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/process) | X     |

</p></details>

<details><summary><b>Training components</b></summary>
<p>

* **Training**
  The Training component allows you to submit Amazon SageMaker Training jobs directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker Training Kubeflow Pipelines component version 1](./train) | [SageMaker Training Kubeflow Pipelines component version 2](./TrainingJob)   |

* **Hyperparameter Optimization**
  The Hyperparameter Optimization component enables you to submit hyperparameter tuning jobs to Amazon SageMaker directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker Hyperparameter Optimization Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/hyperparameter_tuning) | X     |

* **RLEstimator**
  The RLEstimator component allows you to submit RLEstimator (Reinforcement Learning) SageMaker Training jobs directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker RLEstimator Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/rlestimator) | X     |

</p></details>

<details><summary><b>Inference components</b></summary>
<p>

* **Hosting Deploy**
  The Hosting components allow you to deploy a model using SageMaker hosting services from a Kubeflow Pipelines workflow.
  <table>
    <thead>
      <tr>
        <th>Version 1 of the component</th>
        <th>Version 2 of the component</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td><a href="https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy">SageMaker Hosting Services - Create Endpoint Kubeflow Pipeline component version 1</a></td>
        <td>Version 2 of the Hosting components consists of the three sub-components needed to create a hosting deployment on SageMaker.
          <ul>
            <li>A <a href="./Model">SageMaker Model Kubeflow Pipelines component version 2</a> responsible for the model artifacts and the model image registry path that contains the inference code.</li>
            <li>A <a href="./EndpointConfig">SageMaker Endpoint Config Kubeflow Pipelines component version 2</a> responsible for defining the configuration of the endpoint such as the instance type, models, number of instances, and serverless inference option.</li>
            <li>A <a href="./Endpoint">SageMaker Endpoint Kubeflow Pipelines component version 2</a> responsible for creating or updating the endpoint on SageMaker as specified in the endpoint configuration.</li>
          </ul>
        </td>
      </tr>
    </tbody>
  </table>

* **Batch Transform component**
  The Batch Transform component enables you to run inference jobs for an entire dataset in Amazon SageMaker from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [SageMaker Batch Transform Kubeflow Pipeline component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform) | X     |  

* **Model Monitor components**
  The Model Monitor components allow you to monitor the quality of SageMaker machine learning models in production from a Kubeflow Pipelines workflow.
    <table>
    <thead>
      <tr>
        <th>Version 1 of the component</th>
        <th>Version 2 of the component</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td>X</td>
        <td>The Model Monitor components consist of four sub-components for monitoring drift in a model.
          <ul>
            <li>A <a href="./DataQualityJobDefinition">SageMaker Data Quality Job Definition Kubeflow Pipelines component version 2</a> responsible for monitoring drift in data quality.</li>
            <li>A <a href="./ModelQualityJobDefinition">SageMaker Model Quality Job Definition Kubeflow Pipelines component version 2</a>  responsible for monitoring drift in model quality metrics.</li>
            <li>A <a href="./ModelBiasJobDefinition">SageMaker Model Bias Job Definition Kubeflow Pipelines component version 2</a>A responsible for monitoring bias in a model's predictions.</li>
            <li>A <a href="./ModelExplainabilityJobDefinition">SageMaker Model Explainability Job Definition Kubeflow Pipelines component version 2</a> responsible for monitoring drift in feature attribution.</li>
          </ul>Additionally, for on-schedule monitoring at a specified frequency, a fifth component, <a href="./MonitoringSchedule">SageMaker Monitoring Schedule Kubeflow Pipelines component version 2</a>, is responsible for monitoring the data collected from a real-time endpoint on a schedule.
        </td>
      </tr>
    </tbody>
  </table>

</p></details>
<details><summary><b>RoboMaker components</b></summary>
<p>

* **Create Simulation Application**
  The Create Simulation Application component allows you to create a RoboMaker Simulation Application directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [RoboMaker Create Simulation app Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/create_simulation_app)|  X  |  

* **Simulation Job**
  The Simulation Job component allows you to run a RoboMaker Simulation Job directly from a Kubeflow Pipelines workflow.
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [RoboMaker Simulation Job Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job)|  X  | 

* **Simulation Job Batch**
  The Simulation Job Batch component allows you to run a RoboMaker Simulation Job Batch directly from a Kubeflow Pipelines workflow.
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [RoboMaker Simulation Job Batch Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job_batch)|  X  |

* **Delete Simulation Application**
  The Delete Simulation Application component allows you to delete a RoboMaker Simulation Application directly from a Kubeflow Pipelines workflow. 
  | Version 1 of the component    | Version 2 of the component |
  | ----------- | ----------- |
  | [RoboMaker Delete Simulation app Kubeflow Pipelines component version 1](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/delete_simulation_app)|  X  |

</p></details>