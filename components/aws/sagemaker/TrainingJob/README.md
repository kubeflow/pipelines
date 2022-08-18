# SageMaker Training Kubeflow Pipelines component v2

Component to create [SageMaker Training jobs](https://docs.aws.amazon.com/sagemaker/latest/dg/how-it-works-training.html) in a Kubeflow Pipelines workflow.

| :warning: WARNING: This component is in Preview Release Phase. It is not GA |
| --- |

## Overview

The Amazon SageMaker components for Kubeflow Pipelines version 1(v1.1.x or below) uses [Boto3 (AWS SDK for Python)](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html) as the backend to create and manage resources on SageMaker. SageMaker components version 2(v2.0.0-alpha2 or above) uses the [ACK Service Controller for SageMaker](https://github.com/aws-controllers-k8s/sagemaker-controller) to do the same. AWS introduced [ACK](https://aws-controllers-k8s.github.io/community/) to facilitate a Kubernetes-native way of managing AWS Cloud resources. ACK includes a set of AWS service-specific controllers, one of which is the SageMaker controller. The SageMaker controller makes it easier for machine learning developers and data scientists who use Kubernetes as their control plane to train, tune, and deploy machine learning models in Amazon SageMaker.

Creating SageMaker resouces using the controller allows you to create and monitor the resources as part of a Kubeflow Pipelines workflow(same as version 1 of components) and additionally provides you a flexible and consistent experience to manage the SageMaker resources from other environments such as using the Kubernetes command line tool(kubectl) or the other Kubeflow applications such as Notebooks.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started

Follow this guide to get started with using the SageMaker Training Job pipeline component version 2.
### Prerequisites
1. An existing [Kubeflow deployment](https://awslabs.github.io/kubeflow-manifests/docs/deployment/). This guide assumes you have already installed Kubeflow, if you do not have an existing Kubeflow deployment, choose one of the deployment options from the [Kubeflow on AWS Deployment guide](https://awslabs.github.io/kubeflow-manifests/docs/deployment/).
    > *Note:* If you were using the [Kubeflow pipelines standalone](https://www.kubeflow.org/docs/components/pipelines/installation/standalone-deployment/#deploying-kubeflow-pipelines) deployment. You can continue to use it.
1. Install the [ACK Service Controller for SageMaker](https://github.com/aws-controllers-k8s/sagemaker-controller) version 0.4.2+. Follow the [ML with ACK SageMaker Controller tutorial](https://aws-controllers-k8s.github.io/community/docs/tutorials/sagemaker-example/) to install the SageMaker Controller.
    > Note: You only have to install the controller, so you do NOT have to run [Train an XGBoost Model](https://aws-controllers-k8s.github.io/community/docs/tutorials/sagemaker-example/#train-an-xgboost-model) section
1. This guide assumes you have already installed the following tools on your local machine or an EC2 instance:
    - [kubectl](https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html) – A command line tool for working with Kubernetes clusters.
    - [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html) - A command line tool for working with Amazon EKS clusters

### Setup
1. Configure RBAC permissions for the service account used by kubeflow pipeline pods in the user/profile namespace. The pipeline runs are executed in user namespaces using the `default-editor` Kubernetes service account.
    > *Note*: In Kubeflow pipeline standalone deployment, the pipeline runs are executed in kubeflow namespace using the `pipeline-runner` service account
    - Set the environment variable value for `PROFILE_NAMESPACE`(e.g. `kubeflow-user-example-com`) according to your profile and `SERVICE_ACCOUNT` name according to your installation:
        ```
        # For full Kubeflow installation use your profile namespace
        # For Standalone installation use kubeflow
        export PROFILE_NAMESPACE=kubeflow-user-example-com
        
        # For full Kubeflow installation use default-editor
        # For Standalone installation use pipeline-runner
        export KUBEFLOW_PIPELINE_POD_SERVICE_ACCOUNT=default-editor
        ```
    - Create a [RoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-example) that grants service account access to manage sagemaker custom resources
        ```
        cat > manage_sagemaker_cr.yaml <<EOF
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
          name: manage-sagemaker-cr
          namespace: ${PROFILE_NAMESPACE}
        subjects:
        - kind: ServiceAccount
          name: ${KUBEFLOW_PIPELINE_POD_SERVICE_ACCOUNT}
          namespace: ${PROFILE_NAMESPACE}
        roleRef:
          kind: ClusterRole
          name: ack-sagemaker-controller
          apiGroup: rbac.authorization.k8s.io
        EOF

        kubectl apply -f manage_sagemaker_cr.yaml
        ```
    - Check rolebinding was created by running `kubectl get rolebinding manage-sagemaker-cr -n ${PROFILE_NAMESPACE} -oyaml`    
1. (Optional) If you are also using the SageMaker components version 1. Grant SageMaker access to the service account used by kubeflow pipeline pods.
    - Export your cluster name and cluster region
      ```
      export CLUSTER_NAME=
      export CLUSTER_REGION=
      ```
    - ```
      eksctl create iamserviceaccount --name ${KUBEFLOW_PIPELINE_POD_SERVICE_ACCOUNT} --namespace ${PROFILE_NAMESPACE} --cluster ${CLUSTER_NAME} --region ${CLUSTER_REGION} --attach-policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess --override-existing-serviceaccounts --approve
      ```

## Samples
Head over to the [samples](./samples/) directory and follow the README to create jobs on SageMaker.

## Inputs Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [TrainingJob CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/trainingjob/) for the respective structure and pass the input in JSON format. For e.g. the `resourceConfig` in the `TrainingJob` CRD looks like:

```
resourceConfig: 
  instanceCount: integer
  instanceType: string
  volumeKMSKeyID: string
  volumeSizeInGB: integer
```

the `resource_config` input for the component would be:

```
resourceConfig = {
  "instanceCount": 1,
  "instanceType": "ml.m4.xlarge",
  "volumeSizeInGB": 5,
}
```

You might also want to look at the [TrainingJob API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateTrainingJob.html) for a detailed explaination of parameters.

## References
- [Model Training on SageMaker](https://docs.aws.amazon.com/sagemaker/latest/dg/how-it-works-training.html)
- [TrainingJob CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/trainingjob/)
- [TrainingJob API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateTrainingJob.html)
- [Debug, monitor, and profile training jobs in real time using SageMaker Debugger](https://docs.aws.amazon.com/sagemaker/latest/dg/train-debugger.html)