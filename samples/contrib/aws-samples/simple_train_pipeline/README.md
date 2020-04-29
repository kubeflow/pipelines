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
3. Prepare an IAM role with permissions to run SageMaker jobs and access to S3 buckets.   
   
   create a new file "trust.json" with following content
   ```buildoutcfg 
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Sid": "",
         "Effect": "Allow",
         "Principal": {
           "Service": "sagemaker.amazonaws.com"
         },
         "Action": "sts:AssumeRole"
       }
     ]
   }
   ```
   ```buildoutcfg

   # run these commands to create a role named "SageMakerExecutorKFP" with SageMaker and S3 access
   aws iam create-role --role-name SageMakerExecutorKFP --assume-role-policy-document file://trust.json
   aws iam attach-role-policy --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess --role-name SageMakerExecutorKFP
   aws iam attach-role-policy --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess --role-name SageMakerExecutorKFP
   
   # Note down the role ARN
   aws iam get-role --role-name SageMakerExecutorKFP     # | jq .Role.Arn
   ```
4. Add 'aws-secret' to your Kubeflow namespace.
   ```
   # 1. get aws key and secret in base64 format: 

   echo -n "<AWS_ACCESS_KEY_ID>" | base64
   echo -n "<AWS_SECRET_ACCESS_KEY>" | base64

   # 2. Create new file secret.yaml with following content
   
   apiVersion: v1
   kind: Secret
   metadata:
     name: aws-secret
     namespace: kubeflow
   type: Opaque
   data:
     AWS_ACCESS_KEY_ID: <base64_AWS_ACCESS_KEY_ID>
     AWS_SECRET_ACCESS_KEY: <base64_AWS_SECRET_ACCESS_KEY>
     
   # 3. Now apply to the cluster's kubeflow namespace:
 
   kubectl -n kubeflow apply -f secret.yaml 
   ```
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