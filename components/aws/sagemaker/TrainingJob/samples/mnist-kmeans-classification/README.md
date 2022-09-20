# MNIST Classification with KMeans

This sample aims to showcase the usage of v1 components and the new v2 training job component together in a Kubeflow pipeline workflow. The `mnist-classification-pipeline.py` sample runs a pipeline to train a classficiation model using Kmeans with MNIST dataset on SageMaker. This example was taken from an existing [SageMaker example](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/sagemaker-python-sdk/1P_kmeans_highlevel/kmeans_mnist.ipynb) and modified to work with the Amazon SageMaker Components for Kubeflow Pipelines.

## Prerequisites 

1. Make sure you meet all the configuration prerequisites *including the optional step* for using the v2 components from sagemaker/TrainingJob README (https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/TrainingJob). 
2. The optional step in the guide (https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/TrainingJob) is required for this sample to run. Here are the commands in case you missed it:
    1. Grant SageMaker access to the service account used by kubeflow pipeline pods. Export your cluster name and cluster region
    2. export CLUSTER_NAME=
       export CLUSTER_REGION=
    3. eksctl create iamserviceaccount --name ${KUBEFLOW_PIPELINE_POD_SERVICE_ACCOUNT} --namespace ${PROFILE_NAMESPACE} --cluster ${CLUSTER_NAME} --region ${CLUSTER_REGION} --attach-policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess --override-existing-serviceaccounts --approve
3. You have created an S3 bucket and SageMaker execution role created from the sample prerequisites (https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/TrainingJob/samples#prerequisites).

### Sample MNIST dataset

The following commands will copy the data extraction pre-processing script to an S3 bucket which we will use to store artifacts for the pipeline.

1. [Create a bucket](https://docs.aws.amazon.com/AmazonS3/latest/gsg/CreatingABucket.html) in `us-east-1` region if you don't have one already. 
For the purposes of this demonstration, all resources will be created in the us-east-1 region.

2. Upload the `mnist-kmeans-sagemaker/kmeans_preprocessing.py` file to your bucket with the prefix `mnist_kmeans_example/processing_code/kmeans_preprocessing.py`.
This can be done with the following command, replacing `<bucket-name>` with the name of the bucket you previously created in `us-east-1`:
    ```
    aws s3 cp mnist-kmeans-sagemaker/kmeans_preprocessing.py s3://<bucket-name>/mnist_kmeans_example/processing_code/kmeans_preprocessing.py
    ```

## Compile and run the pipelines

1. To compile the pipeline run: `python mnist-classification-pipeline.py`. This will create a `tar.gz` file.
2. In the Kubeflow Pipelines UI, upload this compiled pipeline specification (the *.tar.gz* file) and click on create run.
3. Provide the sagemaker execution `role_arn` you created and `bucket_name` you created as pipeline inputs.
4. Once the pipeline completes, you can go to `batch_transform_output` to check your batch prediction results.
You will also have an model endpoint in service. Refer to [Prediction section](#Prediction) below to run predictions aganist your deployed model aganist the endpoint. Please remember to clean up the endpoint.

## Prediction

1. Find your endpoint name either by,
  - Opening SageMaker [console](https://us-east-1.console.aws.amazon.com/sagemaker/home?region=us-east-1#/endpoints),  or
  - Clicking the `sagemaker-deploy-model-endpoint_name` under `Output artifacts` of `SageMaker - Deploy Model` component of the pipeline run

2. Setup AWS credentials with `sagemaker:InvokeEndpoint` access. [Sample commands](https://sagemaker.readthedocs.io/en/stable/workflows/kubernetes/using_amazon_sagemaker_components.html#configure-permissions-to-run-predictions)
3. Update the `ENDPOINT_NAME` variable in the script below
4. Run the script below to invoke the endpoint

```python
import json
import io
import boto3
import pickle
import urllib.request
import gzip
import numpy


ENDPOINT_NAME="<your_endpoint_name>"

# Simple function to create a csv from numpy array
def np2csv(arr):
    csv = io.BytesIO()
    numpy.savetxt(csv, arr, delimiter=',', fmt='%g')
    return csv.getvalue().decode().rstrip()

# Prepare input for the model
# Load the dataset
s3 = boto3.client("s3")
s3.download_file(
    "sagemaker-sample-files", "datasets/image/MNIST/mnist.pkl.gz", "mnist.pkl.gz")

with gzip.open("mnist.pkl.gz", "rb") as f:
    train_set, _, _ = pickle.load(f, encoding="latin1")

payload = np2csv(train_set[0][30:31])

# Run prediction aganist the endpoint created by the pipeline
runtime = boto3.Session(region_name='us-east-1').client('sagemaker-runtime')
response = runtime.invoke_endpoint(EndpointName=ENDPOINT_NAME,
                                   ContentType='text/csv',
                                   Body=payload)
result = json.loads(response['Body'].read().decode())
print(result)
```