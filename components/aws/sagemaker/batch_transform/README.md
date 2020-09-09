# SageMaker Batch Transform Kubeflow Pipeline component

## Summary
Component to get inferences for an entire dataset in SageMaker from a Kubeflow Pipelines workflow.

## Details
With [batch transform](https://docs.aws.amazon.com/sagemaker/latest/dg/how-it-works-batch.html), you create a batch transform job using a trained model and the dataset, which must be stored in Amazon S3. Use batch transform when you:

* Want to get inferences for an entire dataset and index them to serve inferences in real time
* Don't need a persistent endpoint that applications (for example, web or mobile apps) can call to get inferences
* Don't need the subsecond latency that Amazon SageMaker hosted endpoints provide

## Intended Use
Create a transform job in AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
region | The region where the endpoint is created | No | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
job_name | The name of the transform job. The name must be unique within an AWS Region in an AWS account | Yes | Yes | String | | is a generated name (combination of model_name and 'BatchTransform' string)|
model_name | The name of the model that you want to use for the transform job. Model name must be the name of an existing Amazon SageMaker model within an AWS Region in an AWS account | No | No | String | | |
max_concurrent | The maximum number of parallel requests that can be sent to each instance in a transform job | Yes | Yes | Integer | | 0 |
max_payload | The maximum allowed size of the payload, in MB | Yes | Yes | Integer | The value in max_payload must be greater than, or equal to, the size of a single record | 6 |
batch_strategy | The number of records to include in a mini-batch for an HTTP inference request | Yes | Yes | String | | |
environment | The environment variables to set in the Docker container | Yes | Yes | Dict | Maximum length of 1024. Key Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`. Value Pattern: `[\S\s]*`. Upto 16 key and values entries in the map | |

The following parameters construct [`TransformInput`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformInput.html) object of the CreateTransformJob API. These describe the input source and the way the transform job consumes it.

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
input_location | The S3 location of the data source that is associated with a channel. [Read more on S3Uri](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformS3DataSource.html) | No | No | String | | |
data_type | Used by SageMaker to identify the objects from the S3 bucket to be used for batch transform. [Read more on S3DataType](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformS3DataSource.html) | Yes | Yes | String | `ManifestFile`, `S3Prefix`, `AugmentedManifestFile`| `S3Prefix` |
content_type | The multipurpose internet mail extension (MIME) type of the data. Amazon SageMaker uses the MIME type with each http call to transfer data to the transform job | Yes | Yes | String | | |
split_type | The method to use to split the transform job data files into smaller batches | Yes | Yes | String | `Line`, `RecordIO`, `TFRecord`, `None` | `None` |
compression_type | If the transform data is compressed, specify the compression type | Yes | Yes | String | `GZip`, `None` | `None` |

* `input_location` and `data_type` parameters above are used to construct [`S3DataSource`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformS3DataSource.html) object which is part of [`TransformDataSource`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformDataSource.html) object in [`TransformInput`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformInput.html) part of the CreateTransformJob API.
```
TransformInput={
        'DataSource': {
            'S3DataSource': {
                'S3DataType': 'ManifestFile'|'S3Prefix'|'AugmentedManifestFile',
                'S3Uri': 'string'
            }
        }, 
        ... other input parameters ...
    }
```
[Ref](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_transform_job)

The following parameters are used to construct [`TransformOutput`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformOutput.html) object of the CreateTransformJob API. These describe the results of a transform job.

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
output_location | The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job | No | No | String | | |
accept | The MIME type used to specify the output data. Amazon SageMaker uses the MIME type with each http call to transfer data from the transform job | Yes | Yes | String | | |
assemble_with | Defines how to assemble the results of the transform job as a single S3 object. To concatenate the results in binary format, specify None. To add a newline character at the end of every transformed record, specify Line | Yes | Yes | String | `Line`, `None` | `None`|
output_encryption_key | The AWS Key Management Service key to encrypt the model artifacts at rest using Amazon S3 server-side encryption | Yes | Yes | String | [KmsKeyId formats](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformOutput.html) | |

The following parameters are used to construct [`TransformResources`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformResources.html) object of the CreateTransformJob API. These describe the resources, including ML instance types and ML instance count, to use for the transform job.

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
instance_type | The ML compute instance type for the transform job | Yes | Yes | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge | ml.m4.xlarge |
instance_count | The number of ML compute instances to use in the transform job | Yes | Yes | Integer | | 1 |
resource_encryption_key | The AWS Key Management Service (AWS KMS) key used to encrypt model data on the storage volume attached to the ML compute instance(s) that run the batch transform job. | Yes | Yes | String | [VolumeKmsKeyId formats](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_TransformResources.html) | |

The following parameters are used to construct [`DataProcessing`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_DataProcessing.html) object of the CreateTransformJob API. The data structure used to specify the data to be used for inference in a batch transform job and to associate the data that is relevant to the prediction results in the output.

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
input_filter | A JSONPath expression used to select a portion of the input data to pass to the algorithm. [ReadMore on InputFilter](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_DataProcessing.html) | Yes | Yes | String | | |
output_filter | A JSONPath expression used to select a portion of the joined dataset to save in the output file for a batch transform job. [ReadMore on OutputFilter](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_DataProcessing.html) | Yes | Yes | String | | |
join_source | Specifies the source of the data to join with the transformed data. [ReadMore on JoinSource](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_DataProcessing.html) | Yes | Yes | String | `Input`, `None` | None |

Notes:
* Please use the links in the [Resources section](#Resources) for detailed information on each input parameter and SageMaker APIs used in this component

## Outputs
Name | Description
:--- | :----------
output_location | The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job

## Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

## Samples
### Integrated into a pipeline
MNIST Classification pipeline: [Pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/mnist-classification-pipeline.py) | [Steps](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/README.md)

## Resources
* [Batch Transform on SageMaker](https://docs.aws.amazon.com/sagemaker/latest/dg/how-it-works-batch.html)
* [Create Transform Job API documentation](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateTransformJob.html)
* [Boto3 API reference](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_transform_job)
