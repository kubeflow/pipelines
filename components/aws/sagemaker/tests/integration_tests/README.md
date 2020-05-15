## Requirements
1. [Conda](https://docs.conda.io/en/latest/miniconda.html)
1. [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
1. Argo CLI: [Mac](https://github.com/argoproj/homebrew-tap), [Linux](https://eksworkshop.com/advanced/410_batch/install/)
1. K8s cluster with Kubeflow pipelines > 0.4.0 installed
1. [IAM Role](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) with a SageMakerFullAccess and S3FullAccess
1. IAM User credentials with SageMakerFullAccess permissions

## Creating S3 buckets with datasets

Change the bucket name and run the python script `[s3_sample_data_creator.py](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker#the-sample-dataset)` to create S3 buckets with mnist dataset in the region where you want to run the tests

## Step to run integration tests
1. Configure AWS credentials with access to EKS cluster
1. Fetch kubeconfig to `~/.kube/config` or set `KUBECONFIG` environment variable to point to kubeconfig of the cluster
1. Create a [secret](https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/) named `aws-secret` in kubeflow namespace with credentials of IAM User for SageMakerFullAccess
    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: aws-secret
      namespace: kubeflow
    type: Opaque
    data:
      AWS_ACCESS_KEY_ID: YOUR_BASE64_ACCESS_KEY
      AWS_SECRET_ACCESS_KEY: YOUR_BASE64_SECRET_ACCESS
    ```
    
    > Note: To get base64 string, run `echo -n $AWS_ACCESS_KEY_ID | base64`
1. Create conda environment using environment.yml for running tests. Run `conda env create -f environment.yml`
1. Activate the conda environment `conda activate kfp_test_env`
1. Run port-forward to minio service in background. Example: `kubectl port-forward svc/minio-service 9000:9000 -n kubeflow &`
1. Provide the following arguments to pytest:
    1. `region`: AWS region where test will run. Default - us-west-2
    1. `role-arn`: SageMaker execution IAM role ARN
    1. `s3-data-bucket`: Regional S3 bucket in which test data is hosted
    1. `minio-service-port`: Localhost port to which minio service is mapped to. Default - 9000
    1. `kfp-namespace`: Cluster namespace where kubeflow pipelines is installed. Default - Kubeflow
1.  cd into this directory and run 
    ```
    pytest --region <> --role-arn <> --s3-data-bucket <> --minio-service-port <> --kfp-namespace <>
    ```
