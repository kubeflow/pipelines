# MNIST Classification with K-Means

The mnist-kmeans-pipeline.py sample runs a pipeline to train a classficiation model using Kmeans with MNIST dataset on SageMaker. This example was taken from an existing [SageMaker example](https://github.com/aws/amazon-sagemaker-examples/blob/8279abfcc78bad091608a4a7135e50a0bd0ec8bb/sagemaker-python-sdk/1P_kmeans_highlevel/kmeans_mnist.ipynb) and modified to work with the Amazon SageMaker Components for Kubeflow Pipelines.

## Prepare dataset

To train a model with SageMaker, we need an S3 bucket to store the dataset and artifacts from the training process. We will use the S3 bucket you [created earlier](../README.md#s3-bucket) and simply use the dataset at `s3://sagemaker-sample-files/datasets/image/MNIST/mnist.pkl.gz`.

1. Clone this repository to use the pipelines and sample scripts.
    ```
    git clone https://github.com/kubeflow/pipelines.git
    cd pipelines/components/aws/sagemaker/TrainingJob/samples/mnist-kmeans-training
    ```
1. Run the following commands to install the script dependencies and upload the processed dataset to your S3 bucket:
    ```
    pip install -r requirements.txt
    python s3-sample-data-creator.py
    ```

## Compile and run the pipelines

1. To compile the pipeline run: `python mnist-kmeans-training-pipeline.py`. This will create a `tar.gz` file.
1. In the Kubeflow Pipelines UI, upload this compiled pipeline specification (the *.tar.gz* file) and click on create run.
2. Once the pipeline completes, you can see the outputs under 'Output parameters' in the Training component's Input/Output section.

Example inputs to this pipeline:
> Note: If you are not using `us-east-1` region you will have to find an training image URI according to the region. https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html. For e.g.: for `us-west-2` the image URI is 174872318107.dkr.ecr.us-west-2.amazonaws.com/kmeans:1

```
s3_bucket_name: my_s3_bucket
sagemaker_role_arn: arn:aws:iam::123456789012:role/service-role/AmazonSageMaker-ExecutionRole
region: us-east-1
training_image: 382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1
hyperparameters: {"k": "10", "feature_dim": "784"}
resourceConfig: {
    "instanceCount": 1,
    "instanceType": "ml.m4.xlarge",
    "volumeSizeInGB": 5,
}
```
