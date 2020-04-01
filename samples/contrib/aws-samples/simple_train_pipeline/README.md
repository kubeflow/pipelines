# Simple pipeline with only train component

An example pipeline with only train component.

# Prerequisites 
1. Install kubeflow on an EKS cluster in AWS. https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/
2. Get and store data in S3 buckets. You can get sample data using this code  
   `aws s3 cp s3://m-kfp-mnist/mnist_kmeans_example/data s3://<your_bucket_name>/mnist_kmeans_example/data`
3. Prepare an IAM role with permissions to run SageMaker jobs and access to S3 buckets. https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html  
   Go to you AWS account -> IAM -> Roles -> Create Role -> select sagemaker service -> Next -> Next -> Next -> Give some name -> Create Role -> Click on the role that you created -> Attach policy -> AmazonS3FullAccess -> Attach Policy -> Note down the role ARN
4. Add 'aws-secret' to your kubeflow namespace.
5. Compile the pipeline:  
   `dsl-compile --py training-pipeline.py --output training-pipeline.tar.gz`
6. In the Kubeflow UI, upload this compiled pipeline specification (the .tar.gz file) and click on create run.
7. Once the pipeline completes, you can see the outputs under 'Output parameters' in the HPO component's Input/Output section.

Example inputs to this pipeline :
```buildoutcfg
region : us-east-1
endpoint_url : leave this empty
image : 382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1
training_input_mode : File
hyperparameters : {"k": "10", "feature_dim": "784"}
channels : In this JSON, along with other parametes you need to pass the S3 Uri where you have data

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
model_artifact_path : <some_s3_bucket_where_output_will_be_stored>
output_encryption_key : leave this empty
network_isolation : True
traffic_encryption : False
spot_instance : False
max_wait_time : 3600
checkpoint_config : {}
role : You need an IAM role with Full sagemaker permissions and S3 access 
       Example role input->  arn:aws:iam::999999999999:role/test_temp_role
```


# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)