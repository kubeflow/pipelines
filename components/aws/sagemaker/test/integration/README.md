##Requirements
1. [Conda](https://docs.conda.io/en/latest/miniconda.html)
1. [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
1. Argo CLI: [Mac](https://github.com/argoproj/homebrew-tap), [Linux](https://eksworkshop.com/advanced/410_batch/install/)
1. K8s cluster with Kubeflow pipelines > 0.4.0 installed
1. [IAM Role](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) with a SagemakerFullAccess and S3FullAccess
1. IAM User credentials with SageMakerFullAccess permissions

## Creating S3 buckets with datasets
Create an S3 bucket and use the following python script to copy `train_data`, `test_data`, and `valid_data.csv` to your buckets.  
[create the bucket in AWS region where you want to run your tests](https://docs.aws.amazon.com/AmazonS3/latest/gsg/CreatingABucket.html)

Create a new file named `s3_sample_data_creator.py` with following content :
```python
import pickle, gzip, numpy, urllib.request, json
from urllib.parse import urlparse

# Load the dataset
urllib.request.urlretrieve("http://deeplearning.net/data/mnist/mnist.pkl.gz", "mnist.pkl.gz")
with gzip.open('mnist.pkl.gz', 'rb') as f:
    train_set, valid_set, test_set = pickle.load(f, encoding='latin1')


# Upload dataset to S3
from sagemaker.amazon.common import write_numpy_to_dense_tensor
import io
import boto3

###################################################################
# This is the only thing that you need to change to run this code 
# Give the name of your S3 bucket 
bucket = 'bucket-name' 
###################################################################

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

## Step to run integration tests
1. Configure AWS credentials with access to EKS cluster
1. Fetch kubeconfig to `~/.kube/config` or set `KUBECONFIG` environment variable to point to kubeconfig of the cluster
1. Create a [secret](https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/) named `aws-secret` in kubeflow namespace with credentials of IAM User for SageMakerFullAccess
    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: aws-secret
      namespace: kubeflow
    type: Opaque
    data:
      AWS_ACCESS_KEY_ID: YOUR_BASE64_ACCESS_KEY
      AWS_SECRET_ACCESS_KEY: YOUR_BASE64_SECRET_ACCESS
    ```
    
    > Note: To get base64 string, try `echo -n $AWS_ACCESS_KEY_ID | base64`
1. Create conda environment using environment.yml for running tests. Run `conda env create -f environment.yml`
1. Activate the conda environment `conda activate kfp_test_env`
1. Run port-forward to minio service in background. Example: `kubectl port-forward svc/minio-service 9000:9000 -n kubeflow &`
1. Create (if does not exist) and provide the following arguments to pytest:
    1. `region`: AWS region where test will run. Default - us-west-2
    1. `role-arn`: Sagemaker execution IAM role ARN
    1. `s3-data-bucket`: Regional S3 bucket in which test data is hosted
    1. `minio-service-port`: Localhost port to which minio service is mapped to. Default - 9000
    1. `kfp-namespace`: Cluster namespace where kubeflow pipelines is installed. Default - Kubeflow
1.  cd into this directory and run 
    ```
    pytest --region <> --role-arn <> --s3-data-bucket <> --minio-service-port <> --kfp-namespace <>
    ```
