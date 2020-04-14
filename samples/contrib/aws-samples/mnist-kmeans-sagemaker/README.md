The `mnist-classification-pipeline.py` sample runs a pipeline to train a classficiation model using Kmeans with MNIST dataset on Sagemaker.  
The `kmeans-hpo-pipeline.py` is a single component hyper parameter optimisation pipeline which has default values set to use Kmeans. 

If you do not have `train_data`, `test_data`, and `valid_data` you can use the following code to get sample data which  
(This data can be used for both of these pipelines)

## The sample dataset

This sample is based on the [Train a Model with a Built-in Algorithm and Deploy it](https://docs.aws.amazon.com/sagemaker/latest/dg/ex1.html).

The sample trains and deploy a model based on the [MNIST dataset](http://www.deeplearning.net/tutorial/gettingstarted.html).


Create an S3 bucket and use the following python script to copy `train_data`, `test_data`, and `valid_data.csv` to your buckets.  
(create the bucket in `us-west-2` region if you are gonna use default values of the pipeline)
https://docs.aws.amazon.com/AmazonS3/latest/gsg/CreatingABucket.html

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

# If you are gonna use the default values of the pipeline then 
# give a bucket name which is in us-west-2 region 
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
## SageMaker permission

In order to run this pipeline, we need to prepare an IAM Role to run Sagemaker jobs. You need this `role_arn` to run a pipeline. Check [here](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-roles.html) for details.

This pipeline also use aws-secret to get access to Sagemaker services, please also make sure you have a `aws-secret` in the kubeflow namespace.

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


## Compiling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

```bash
dsl-compile --py mnist-classification-pipeline.py --output mnist-classification-pipeline.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

The pipeline requires several arguments, replace `role_arn` and data path with your settings.

Once the pipeline done, you can go to `batch_transform_ouput` to check your batch prediction results.
You will have an model endpoint in service. Please remember to clean it up.


## Prediction

Open Sagemaker [console](https://us-west-2.console.aws.amazon.com/sagemaker/home?region=us-west-2#/endpoints) and find your endpoint name, You can call endpoint in this way. Please check dataset section to get `train_set`.

```python
import json
import io
import boto3

# Simple function to create a csv from our numpy array
def np2csv(arr):
    csv = io.BytesIO()
    numpy.savetxt(csv, arr, delimiter=',', fmt='%g')
    return csv.getvalue().decode().rstrip()

runtime = boto3.Session().client('sagemaker-runtime')

payload = np2csv(train_set[0][30:31])

response = runtime.invoke_endpoint(EndpointName='Endpoint-20190502202738-LDKG',
                                   ContentType='text/csv',
                                   Body=payload)
result = json.loads(response['Body'].read().decode())
print(result)
```

## Components source

Hyperparameter Tuning:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/hyperparameter_tuning/src)

Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train/src)

Model creation:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/model/src)

Endpoint Deployment:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy/src)

Batch Transformation:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform/src)
