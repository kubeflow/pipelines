# Simple pipeline with only train component

An example pipeline with only [train component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train).


## Prerequisites 

Make sure you have the setup explained in this [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/README.md)


## Steps 
1. Compile the pipeline:  
   `dsl-compile --py training-pipeline.py --output training-pipeline.tar.gz`
2. In the Kubeflow UI, upload this compiled pipeline specification (the .tar.gz file) and click on create run.
3. Once the pipeline completes, you can see the outputs under 'Output parameters' in the HPO component's Input/Output section.

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
                        "S3Uri": "s3://<your_bucket_name>/mnist_kmeans_example/train_data",
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

instance_type : ml.m5.2xlarge
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