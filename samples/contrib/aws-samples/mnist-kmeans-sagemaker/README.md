# MNIST Classification with KMeans

The `mnist-classification-pipeline.py` sample pipeline shows how to create an end to end ML workflow to train and deploy a model on SageMaker. We will train a classification model using Kmeans algorithm with MNIST dataset on SageMaker. Additionally, this sample also demonstrates how to use SageMaker components v1 and v2 together in a Kubeflow pipeline workflow. This example was taken from an existing [SageMaker example](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/sagemaker-python-sdk/1P_kmeans_highlevel/kmeans_mnist.ipynb) and modified to work with the Amazon SageMaker Components for Kubeflow Pipelines.

## Prerequisites 

1. Make sure you have completed all the pre-requisites mentioned in this [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/README.md).

### Sample MNIST dataset

1. Clone this repository to use the pipelines and sample scripts.
    ```
    git clone https://github.com/kubeflow/pipelines.git
    cd pipelines/samples/contrib/aws-samples/mnist-kmeans-sagemaker
    ```
The following commands will copy the data extraction pre-processing script to an S3 bucket which we will use to store artifacts for the pipeline.

2. [Create a bucket](https://docs.aws.amazon.com/AmazonS3/latest/gsg/CreatingABucket.html) in `us-east-1` region. 
For the purposes of this demonstration, all resources will be created in the us-east-1 region.
    -  Specify your S3_BUCKET_NAME
        ```
        export S3_BUCKET_NAME=
        ```
    ```
    export SAGEMAKER_REGION=us-east-1
    if [[ $SAGEMAKER_REGION == "us-east-1" ]]; then
        aws s3api create-bucket --bucket ${S3_BUCKET_NAME} --region ${SAGEMAKER_REGION}
    else
        aws s3api create-bucket --bucket ${S3_BUCKET_NAME} --region ${SAGEMAKER_REGION} \
        --create-bucket-configuration LocationConstraint=${SAGEMAKER_REGION}
    fi

    echo ${S3_BUCKET_NAME}
    ```
3. Upload the `mnist-kmeans-sagemaker/kmeans_preprocessing.py` file to your bucket with the prefix `mnist_kmeans_example/processing_code/kmeans_preprocessing.py`.
    ```
    aws s3 cp kmeans_preprocessing.py s3://${S3_BUCKET_NAME}/mnist_kmeans_example/processing_code/kmeans_preprocessing.py
    ```

## Compile and run the pipelines

1. To compile the pipeline run: `python mnist-classification-pipeline.py`. This will create a `tar.gz` file.
2. In the Kubeflow Pipelines UI, upload this compiled pipeline specification (the *.tar.gz* file) and click on create run.
3. Provide the sagemaker execution `role_arn` you created and `bucket_name` you created as pipeline inputs.
4. Once the pipeline completes, you can go to `batch_transform_output` to check your batch prediction results.
You will also have an model endpoint in service. Refer to [Prediction section](#Prediction) below to run predictions aganist your deployed model aganist the endpoint.

## Prediction

1. Find your endpoint name by:
    - Checking the `sagemaker_resource_name` field under Output artifacts of the Endpoint component in the pipeline run.
    ```
      export ENDPOINT_NAME=
    ```
2. Setup AWS credentials with `sagemaker:InvokeEndpoint` access. [Sample commands](https://sagemaker.readthedocs.io/en/v1.60.2/kubernetes/using_amazon_sagemaker_components.html#configure-permissions-to-run-predictions)
4. Run the script below to invoke the endpoint
  ```
    python invoke_endpoint.py $ENDPOINT_NAME
  ```

## Cleaning up the endpoint

You can find the model/endpoint configuration name in the `sagemaker_resource_name` field under Output artifacts of the EndpointConfig/Model component in the pipeline run.

```
export ENDPOINT_CONFIG_NAME=
export MODEL_NAME=
```
To delete all the endpoint resources use:

Note: The namespace for the standard kubeflow installation is "kubeflow". For multi-tenant installations the namespace is located at the left in the navigation bar.

```
export MY_KUBEFLOW_NAMESPACE=

kubectl delete endpoint $ENDPOINT_NAME -n $MY_KUBEFLOW_NAMESPACE
kubectl delete endpointconfig $ENDPOINT_CONFIG_NAME -n $MY_KUBEFLOW_NAMESPACE
kubectl delete model $MODEL_NAME -n $MY_KUBEFLOW_NAMESPACE
```

## Components source

Hyperparameter Tuning:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/hyperparameter_tuning/src)

Training: 
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/TrainingJob/src)


Endpoint:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/Endpoint/src)

Endpoint Config:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/EndpointConfig/src)

Model:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/Modelv2/src)

Batch Transformation:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform/src)
