# Sample AWS SageMaker Kubeflow Pipelines 

This folder contains many example pipelines which use [AWS SageMaker Components for KFP](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker). The following sections explain the setup needed to run these pipelines. Once you are done with the setup, [simple_train_pipeline](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/simple_train_pipeline) is a good place to start if you have never used these components before.



## Prerequisites 

1. You need a cluster with Kubeflow installed on it. [Install Kubeflow on AWS cluster](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)
2. Install the following on your local machine or EC2 instance (These are recommended tools. Not all of these are required)
    1. [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html). If you are using an IAM user, configure your [Access Key ID, Secret Access Key](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys) and preferred AWS Region by running:
       `aws configure`  
    2. [aws-iam-authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) version 0.1.31 and above  
    3. [eksctl](https://github.com/weaveworks/eksctl) version above 0.15  
    4. [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) version needs to be your k8s version +/- 1 minor version.
    5. [KFP SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/#install-the-kubeflow-pipelines-sdk) (installs the dsl-compile and kfp cli)


## IAM Permissions 

To use AWS KFP Components the KFP component pods need access to AWS SageMaker.
There are two ways you can give them access to SageMaker. 
(You need EKS cluster for Option 1)

**Option 1** (Recommended) [IAM roles for service account](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html). 
   1. Enable OIDC support on EKS cluster 
      ```
      eksctl utils associate-iam-oidc-provider --cluster <cluster_name> \
       --region <cluster_region> --approve
      ```
   2. Take note of the OIDC issuer URL. This URL is in the form `oidc.eks.<region>.amazonaws.com/id/<OIDC_ID>` . Note down the URL.
      ```
      aws eks describe-cluster --name <cluster_name> --query "cluster.identity.oidc.issuer" --output text
      ```
   3. Create a file named trust.json with the following content.   
      Replace `<OIDC_URL>` with your OIDC issuer URL **(Don’t include https://)** and `<AWS_ACCOUNT_NUMBER>` with your AWS account number. 
      ```
      # Replace these two with proper values 
      OIDC_URL="<OIDC_URL>"
      AWS_ACC_NUM="<AWS_ACCOUNT_NUMBER>"
      
      # Run this to create trust.json file
      cat <<EOF > trust.json
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Federated": "arn:aws:iam::$AWS_ACC_NUM:oidc-provider/$OIDC_URL"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
              "StringEquals": {
                "$OIDC_URL:aud": "sts.amazonaws.com",
                "$OIDC_URL:sub": "system:serviceaccount:kubeflow:pipeline-runner"
              }
            }
          }
        ]
      }
      EOF
      ```
   4. Create an IAM role using trust.json. Make a note of the ARN returned in the output.
      ```
      aws iam create-role --role-name kfp-example-pod-role --assume-role-policy-document file://trust.json
      aws iam attach-role-policy --role-name kfp-example-pod-role --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
      aws iam get-role --role-name kfp-example-pod-role --output text --query 'Role.Arn'
      ```
   5. Edit your pipeline-runner service account.
      ```
      kubectl edit -n kubeflow serviceaccount pipeline-runner
      ```
      Add `eks.amazonaws.com/role-arn: <role_arn>` to annotations, then save the file. Example: **(add only line 5)**  
      ```
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        annotations:
          eks.amazonaws.com/role-arn: <role_arn>
        creationTimestamp: "2020-04-16T05:48:06Z"
        labels:
          app: pipeline-runner
          app.kubernetes.io/component: pipelines-runner
          app.kubernetes.io/instance: pipelines-runner-0.2.0
          app.kubernetes.io/managed-by: kfctl
          app.kubernetes.io/name: pipelines-runner
          app.kubernetes.io/part-of: kubeflow
          app.kubernetes.io/version: 0.2.0
        name: pipeline-runner
        namespace: kubeflow
        resourceVersion: "11787"
        selfLink: /api/v1/namespaces/kubeflow/serviceaccounts/pipeline-runner
        uid: d86234bd-7fa5-11ea-a8f2-02934be6dc88
      secrets:
      - name: pipeline-runner-token-dkjrk
      ``` 
**Option 2** Store the IAM credentials as a `aws-secret` in kubernetes cluster. Then use those in the components.
   1. You need credentials for an IAM user with SageMakerFullAccess. Apply them to k8s cluster.
      Replace `AWS_ACCESS_KEY_IN_BASE64` and `AWS_SECRET_ACCESS_IN_BASE64`.
      > Note: To get base64 string you can do `echo -n $AWS_ACCESS_KEY_ID | base64`
      ```
      cat <<EOF | kubectl apply -f -
      apiVersion: v1
      kind: Secret
      metadata:
        name: aws-secret
        namespace: kubeflow
      type: Opaque
      data:
        AWS_ACCESS_KEY_ID: <AWS_ACCESS_KEY_IN_BASE64>
        AWS_SECRET_ACCESS_KEY: <AWS_SECRET_ACCESS_IN_BASE64>
      EOF
      ```
   2. Use the stored `aws-secret` in pipeline code by adding this line to each component in your pipeline `.apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))`   
      [Kubeflow Document](https://www.kubeflow.org/docs/aws/pipeline/)  
      [Example Code](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/simple_train_pipeline/training-pipeline.py#L76) (uncomment this line)

## Inputs to the pipeline

### Sample MNIST dataset

Use the following python script to copy train_data, test_data, and valid_data to your bucket.  
[Create a bucket](https://docs.aws.amazon.com/AmazonS3/latest/gsg/CreatingABucket.html) in `us-east-1` region if you don't have one already. 
For the purposes of this demonstration, all resources will be created in the us-east-1 region.


Create a new file named s3_sample_data_creator.py with following content :
```
import pickle, gzip, numpy, urllib.request, json
from urllib.parse import urlparse

###################################################################
# This is the only thing that you need to change to run this code 
# Give the name of your S3 bucket 
bucket = '<bucket-name>' 

# If you are gonna use the default values of the pipeline then 
# give a bucket name which is in us-east-1 region 
###################################################################


# Load the dataset
urllib.request.urlretrieve("http://deeplearning.net/data/mnist/mnist.pkl.gz", "mnist.pkl.gz")
with gzip.open('mnist.pkl.gz', 'rb') as f:
    train_set, valid_set, test_set = pickle.load(f, encoding='latin1')


# Upload dataset to S3
from sagemaker.amazon.common import write_numpy_to_dense_tensor
import io
import boto3

train_data_key = 'mnist_kmeans_example/train_data'
test_data_key = 'mnist_kmeans_example/test_data'
train_data_location = 's3://{}/{}'.format(bucket, train_data_key)
test_data_location = 's3://{}/{}'.format(bucket, test_data_key)
print('training data will be uploaded to: {}'.format(train_data_location))
print('training data will be uploaded to: {}'.format(test_data_location))

# Convert the training data into the format required by the SageMaker KMeans algorithm
buf = io.BytesIO()
write_numpy_to_dense_tensor(buf, train_set[0], train_set[1])
buf.seek(0)

boto3.resource('s3').Bucket(bucket).Object(train_data_key).upload_fileobj(buf)

# Convert the test data into the format required by the SageMaker KMeans algorithm
write_numpy_to_dense_tensor(buf, test_set[0], test_set[1])
buf.seek(0)

boto3.resource('s3').Bucket(bucket).Object(test_data_key).upload_fileobj(buf)

# Convert the valid data into the format required by the SageMaker KMeans algorithm
numpy.savetxt('valid-data.csv', valid_set[0], delimiter=',', fmt='%g')
s3_client = boto3.client('s3')
input_key = "{}/valid_data.csv".format("mnist_kmeans_example/input")
s3_client.upload_file('valid-data.csv', bucket, input_key)
```
Run this file `python s3_sample_data_creator.py`

### Role Input

This role is used by SageMaker jobs created by the KFP to access the S3 buckets and other AWS resources.
Run these commands to create the sagemaker-execution-role.   
Note down the Role ARN. You need to give this Role ARN as input in pipeline.

```
TRUST="{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"sagemaker.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"
aws iam create-role --role-name kfp-example-sagemaker-execution-role --assume-role-policy-document "$TRUST"
aws iam attach-role-policy --role-name kfp-example-sagemaker-execution-role --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
aws iam attach-role-policy --role-name kfp-example-sagemaker-execution-role --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess
aws iam get-role --role-name kfp-example-sagemaker-execution-role --output text --query 'Role.Arn'

# note down the Role ARN. 
```

