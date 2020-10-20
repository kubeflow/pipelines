import argparse
import requests
from pyspark.sql import SparkSession
from minio import Minio
from minio.error import ResponseError
import re
from pathlib import Path


def get_secret_creds(path):
    with open(path, 'r') as f:
        cred = f.readline().strip('\'')
    f.close()
    return cred

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type=str, help='Object storage bucket name', default="dummy-bucket-name")
    parser.add_argument('--data_url', type=str, help='URL of the data source', required=True)
    parser.add_argument('--output_filename_path', type=str, help='Output path for filename', default='/tmp/filename')
    args = parser.parse_args()

    cos_bucket_name = args.bucket_name
    data_url = args.data_url

    cos_endpoint = get_secret_creds("/app/secrets/cos_endpoint")
    cos_access_key = get_secret_creds("/app/secrets/cos_access_key")
    cos_secret_key = get_secret_creds("/app/secrets/cos_secret_key")

    ''' Remove possible http scheme for Minio '''
    url = re.compile(r"https?://")
    cos_endpoint = url.sub('', cos_endpoint)

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

    ''' Upload data to Cloud object storage '''
    cos = Minio(cos_endpoint,
                access_key=cos_access_key,
                secret_key=cos_secret_key,
                secure=True)

    if not cos.bucket_exists(cos_bucket_name):
        try:
            cos.make_bucket(cos_bucket_name)
        except ResponseError as err:
            print(err)

    cos.fput_object(cos_bucket_name, filename, filename)

    print('Data ' + filename + ' is uploaded to bucket at ' + cos_bucket_name)
    Path(args.output_filename_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_filename_path).write_text(filename)

    df_data.printSchema()

    print("Number of records: " + str(df_data.count()))
