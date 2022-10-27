import json
import io
import boto3
import pickle
import gzip
import numpy
import sys


if len(sys.argv) < 2:
    print("Must pass your endpoint name")
    exit(1)

ENDPOINT_NAME = sys.argv[1]
# Simple function to create a csv from numpy array
def np2csv(arr):
    csv = io.BytesIO()
    numpy.savetxt(csv, arr, delimiter=",", fmt="%g")
    return csv.getvalue().decode().rstrip()


# Prepare input for the model
# Load the dataset
s3 = boto3.client("s3")
s3.download_file(
    "sagemaker-sample-files", "datasets/image/MNIST/mnist.pkl.gz", "mnist.pkl.gz"
)

with gzip.open("mnist.pkl.gz", "rb") as f:
    train_set, _, _ = pickle.load(f, encoding="latin1")

payload = np2csv(train_set[0][30:31])

# Run prediction aganist the endpoint created by the pipeline
runtime = boto3.Session(region_name="us-east-1").client("sagemaker-runtime")
response = runtime.invoke_endpoint(
    EndpointName=ENDPOINT_NAME, ContentType="text/csv", Body=payload
)
result = json.loads(response["Body"].read().decode())
print(result)
