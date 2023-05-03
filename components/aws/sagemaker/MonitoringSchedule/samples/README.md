# Create A SageMaker Monitoring Schedule

This sample demonstrates how to create a SageMaker Monitoring Schedule using two Kubeflow Pipelines components: Monitoring Schedule and Data Quality Job Definition.

## Prerequisites
Follow the steps in [TrainingJob](../../TrainingJob/samples/README.md), and create S3 bucket and IAM role for SageMaker execution.

## Prepare data bucket
To create a monitoring schedule, you need to fill your S3 bucket with [sample data](./model-monitor/) which contains:
- Pre-trained model
- Baselining constrains and statistics

1. Clone this repository to use the pipelines and sample scripts.
    ```
    git clone https://github.com/kubeflow/pipelines.git
    cd pipelines/components/aws/sagemaker/MonitoringSchedule/samples
    ```
1. Run the following commands to upload the sample data to your S3 bucket:
    ```
    aws s3 cp model-monitor s3://$S3_BUCKET_NAME/model-monitor --recursive
    ```

## Setup endpoint to be monitored
Run the following commands to create a SageMaker endpoint:
```
python set_up_endpoint.py --s3_data_bucket $S3_BUCKET_NAME --sagemaker_role_arn $SAGEMAKER_EXECUTION_ROLE_ARN --region $SAGEMAKER_REGION
```

## Compile and run the pipelines
1. To compile the pipeline run: `python MonitoringSchedule-DataQualityJobDefinition-pipeline.py`. This will create a `tar.gz` file.
1. In the Kubeflow Pipelines UI, upload this compiled pipeline specification (the *.tar.gz* file) and click on create run.
1. Once the pipeline completes, you can see the outputs under 'Output parameters' in the Training component's Input/Output section.

Example inputs to this pipeline:
> Note: If you are not using `us-east-1` region you will have to find an training image URI according to the region. https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html. For e.g.: for `us-west-2` the image URI is `159807026194.dkr.ecr.us-west-2.amazonaws.com/sagemaker-model-monitor-analyzer`

## Delete Endpoint and S3 bucket
- [Delete an endpoint service
](https://docs.aws.amazon.com/vpc/latest/privatelink/delete-endpoint-service.html)
- Delete bucket
```
aws delete-bucket --bucket $S3_BUCKET_NAME --region $SAGEMAKER_REGION
```
## Reference
[Sample Notebook - Introduction to Amazon SageMaker Model Monitor](https://sagemaker-examples.readthedocs.io/en/latest/sagemaker_model_monitor/introduction/SageMaker-ModelMonitoring.html)