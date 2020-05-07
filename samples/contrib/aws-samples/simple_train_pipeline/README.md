# Simple pipeline with only train component

An example pipeline with only [train component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train).

# Prerequisites 
1. Install Kubeflow on an EKS cluster in AWS. https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/
2. Get and store data in S3 buckets. You can get sample data using this code.  
   Create a new file `s3_sample_data_creator.py` with following content :
   ```buildoutcfg
   import io
   import boto3
   import pickle, gzip, numpy, urllib.request, json
   from urllib.parse import urlparse
   from sagemaker.amazon.common import write_numpy_to_dense_tensor

   
   ###########################################################################################
   # This is the only thing that you need to change in this code 
   # Give the name of your S3 bucket 
   # To use the example input below give a bucket name which is in us-east-1 region 
   bucket = '<bucket-name>' 

   ###########################################################################################
      
   # Load the dataset
   urllib.request.urlretrieve("http://deeplearning.net/data/mnist/mnist.pkl.gz", "mnist.pkl.gz")
   with gzip.open('mnist.pkl.gz', 'rb') as f:
       train_set, valid_set, test_set = pickle.load(f, encoding='latin1')


   # Upload dataset to S3   
   data_key = 'mnist_kmeans_example/data'
   data_location = 's3://{}/{}'.format(bucket, data_key)
   print('Data will be uploaded to: {}'.format(data_location))

   # Convert the training data into the format required by the SageMaker KMeans algorithm
   buf = io.BytesIO()
   write_numpy_to_dense_tensor(buf, train_set[0], train_set[1])
   buf.seek(0)

   boto3.resource('s3').Bucket(bucket).Object(data_key).upload_fileobj(buf)
   ```
   Run this file `python s3_sample_data_creator.py`
3. **Prepare IAM Roles**

   We need two IAM roles to run AWS KFP components. You only have to do this once. (Re-use the Role ARNs if you have done this before)

   **Role 1]** For KFP pods to access AWS Sagemaker. Here are the steps to create it.
   i. Enable OIDC support on the EKS cluster
      ```
      eksctl utils associate-iam-oidc-provider --cluster <cluster_name> \
       --region <cluster_region> --approve
      ```
   ii. Take note of the [OIDC](https://openid.net/connect/) issuer URL. This URL is in the form `oidc.eks.<region>.amazonaws.com/id/<OIDC_ID>` . Note down the URL.
      ```
      aws eks describe-cluster --name <cluster_name> --query "cluster.identity.oidc.issuer" --output text
      ```
   iii. Create a file named trust.json with the following content.   
      Replace `<OIDC_URL>` with your OIDC issuer URL **(Don’t include https://)** and `<AWS_account_number>` with your AWS account number. 
      ```
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Federated": "arn:aws:iam::<AWS_account_number>:oidc-provider/<OIDC_URL>"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
              "StringEquals": {
                "<OIDC_URL>:aud": "sts.amazonaws.com",
                "<OIDC_URL>:sub": "system:serviceaccount:kubeflow:pipeline-runner"
              }
            }
          }
        ]
      }
      ```
   iv. Create an IAM role using trust.json. Make a note of the ARN returned in the output.
      ```
      aws iam create-role --role-name kfp-example-pod-role --assume-role-policy-document file://trust.json
      aws iam attach-role-policy --role-name kfp-example-pod-role --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
      aws iam get-role --role-name kfp-example-pod-role --output text --query 'Role.Arn'
      ```
   v. Edit your pipeline-runner service account.
      ```
      kubectl edit -n kubeflow serviceaccount pipeline-runner
      ```
      Add `eks.amazonaws.com/role-arn: <role_arn>` to annotations, then save the file. Example:   
      ```
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        annotations:
          eks.amazonaws.com/role-arn: <role_arn>
          kubectl.kubernetes.io/last-applied-configuration: |
            {"apiVersion":"v1","kind":"ServiceAccount","metadata":{"annotations":{},"labels":{"app":"pipeline-runner","app.kubernetes.io/component":"pipelines-runner","app.kubernetes.io/instance":"pipelines-runner-0.2.0","app.kubernetes.io/managed-by":"kfctl","app.kubernetes.io/name":"pipelines-runner","app.kubernetes.io/part-of":"kubeflow","app.kubernetes.io/version":"0.2.0"},"name":"pipeline-runner","namespace":"kubeflow"}}
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
   **Role 2]** For sagemaker job to access S3 buckets and other Sagemaker services. This Role ARN is given as an input to the components.
      ```
      SAGEMAKER_EXECUTION_ROLE_NAME=kfp-example-sagemaker-execution-role

      TRUST="{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"sagemaker.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"
      aws iam create-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME}--assume-role-policy-document "$TRUST"
      aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
      aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

      aws iam get-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --output text --query 'Role.Arn'

      # Note down the role arn which is of the form 
      arn:aws:iam::<AWS_acc_num>:role/<role-name>
5. Compile the pipeline:  
   `dsl-compile --py training-pipeline.py --output training-pipeline.tar.gz`
6. In the Kubeflow UI, upload this compiled pipeline specification (the .tar.gz file) and click on create run.
7. Once the pipeline completes, you can see the outputs under 'Output parameters' in the HPO component's Input/Output section.

Example inputs to this pipeline :
```buildoutcfg
region : us-east-1
endpoint_url : <leave this empty>
image : 382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1
training_input_mode : File
hyperparameters : {"k": "10", "feature_dim": "784"}
channels : In this JSON, along with other parameters you need to pass the S3 Uri where you have data

                [
                  {
                    "ChannelName": "train",
                    "DataSource": {
                      "S3DataSource": {
                        "S3Uri": "s3://<your_bucket_name>/mnist_kmeans_example/data",
                        "S3DataType": "S3Prefix",
                        "S3DataDistributionType": "FullyReplicated"
                      }
                    },
                    "ContentType": "",
                    "CompressionType": "None",
                    "RecordWrapperType": "None",
                    "InputMode": "File"
                  }
                ]

instance_type : ml.p2.xlarge
instance_count : 1
volume_size : 50
max_run_time : 3600
model_artifact_path : This is where the output model will be stored 
                      s3://<your_bucket_name>/mnist_kmeans_example/output
output_encryption_key : <leave this empty>
network_isolation : True
traffic_encryption : False
spot_instance : False
max_wait_time : 3600
checkpoint_config : {}
role : Paste the role ARN that you noted down  
       (The IAM role with Full SageMaker permissions and S3 access)
       Example role input->  arn:aws:iam::999999999999:role/SageMakerExecutorKFP
```


# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)