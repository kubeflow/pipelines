# MNIST Classification with KMeans

The `mnist-classification-pipeline.py` sample runs a pipeline to train a classficiation model using Kmeans with MNIST dataset on SageMaker. This example was taken from an existing [SageMaker example](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/sagemaker-python-sdk/1P_kmeans_highlevel/kmeans_mnist.ipynb) and modified to work with the Amazon SageMaker Components for Kubeflow Pipelines.

## Prerequisites 

Make sure you have the setup explained in this [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/README.md)


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

Open SageMaker [console](https://us-east-1.console.aws.amazon.com/sagemaker/home?region=us-east-1#/endpoints) and find your endpoint name, You can call endpoint in this way. Please check dataset section to get `train_set`.

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
