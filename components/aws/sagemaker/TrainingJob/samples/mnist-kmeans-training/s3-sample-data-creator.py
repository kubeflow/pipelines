import pickle, gzip, numpy
import os
import boto3

# If you are gonna use the default values of the pipeline then
# give a bucket name which is in us-east-1 region

bucket = os.getenv("S3_BUCKET_NAME", None)
if not bucket:
    raise Exception("S3_BUCKET_NAME not exported")

# Load the dataset
s3 = boto3.client("s3")
s3.download_file(
    "sagemaker-sample-files", "datasets/image/MNIST/mnist.pkl.gz", "mnist.pkl.gz"
)

with gzip.open("mnist.pkl.gz", "rb") as f:
    train_set, valid_set, test_set = pickle.load(f, encoding="latin1")


# Upload dataset to S3
from sagemaker.amazon.common import write_numpy_to_dense_tensor
import io

train_data_key = "mnist_kmeans_example/train_data"
test_data_key = "mnist_kmeans_example/test_data"
train_data_location = "s3://{}/{}".format(bucket, train_data_key)
test_data_location = "s3://{}/{}".format(bucket, test_data_key)
print("training data will be uploaded to: {}".format(train_data_location))
print("training data will be uploaded to: {}".format(test_data_location))

# Convert the training data into the format required by the SageMaker KMeans algorithm
buf = io.BytesIO()
write_numpy_to_dense_tensor(buf, train_set[0], train_set[1])
buf.seek(0)

boto3.resource("s3").Bucket(bucket).Object(train_data_key).upload_fileobj(buf)

# Convert the test data into the format required by the SageMaker KMeans algorithm
write_numpy_to_dense_tensor(buf, test_set[0], test_set[1])
buf.seek(0)

boto3.resource("s3").Bucket(bucket).Object(test_data_key).upload_fileobj(buf)

# Convert the valid data into the format required by the SageMaker KMeans algorithm
numpy.savetxt("valid-data.csv", valid_set[0], delimiter=",", fmt="%g")
s3_client = boto3.client("s3")
input_key = "{}/valid_data.csv".format("mnist_kmeans_example/input")
s3_client.upload_file("valid-data.csv", bucket, input_key)
