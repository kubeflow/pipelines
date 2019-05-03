import argparse
import ibm_boto3
import requests
from ibm_botocore.client import Config
from pyspark.sql import SparkSession

def get_secret(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type=str, help='Object storage bucket name', default="dummy-bucket-name")
    parser.add_argument('--data_url', type=str, help='URL of the data source', default="https://raw.githubusercontent.com/emartensibm/german-credit/binary/credit_risk_training.csv")
    args = parser.parse_args()

    cos_bucket_name = args.bucket_name
    data_url = args.data_url

    cos_url = get_secret("/app/secrets/cos_url")
    cos_apikey = get_secret("/app/secrets/cos_resource_id")
    cos_resource_instance_id = get_secret("/app/secrets/cos_resource_id")

    ''' Download data from data source '''
    filename = data_url
    response = requests.get(data_url, allow_redirects=True)
    if data_url.find('/'):
        filename = data_url.rsplit('/', 1)[1]

    open(filename, 'wb').write(response.content)

    ''' Read data with Spark SQL '''
    spark = SparkSession.builder.getOrCreate()
    df_data = spark.read.csv(path=filename, sep=",", header=True, inferSchema=True)
    df_data.head()

    ''' Upload data to IBM Cloud object storage '''
    cos = ibm_boto3.resource('s3',
                             ibm_api_key_id=cos_apikey,
                             ibm_service_instance_id=cos_resource_instance_id,
                             ibm_auth_endpoint='https://iam.bluemix.net/oidc/token',
                             config=Config(signature_version='oauth'),
                             endpoint_url=cos_url)

    buckets = []
    for bucket in cos.buckets.all():
        buckets.append(bucket.name)

    if cos_bucket_name not in buckets:
        cos.create_bucket(Bucket=cos_bucket_name)

    cos.Bucket(cos_bucket_name).upload_file(filename, filename)

    print('Data ' + filename + ' is uploaded to bucket at ' + cos_bucket_name)
    with open("/tmp/filename.txt", "w") as report:
        report.write(filename)

    df_data.printSchema()

    print("Number of records: " + str(df_data.count()))
