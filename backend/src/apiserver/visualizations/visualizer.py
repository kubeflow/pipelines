# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import os
import json
from tensorflow.python.lib.io import file_io
import pandas as pd
from sklearn.metrics import roc_curve
from minio import Minio
from minio.error import (ResponseError, BucketAlreadyOwnedByYou,
                         BucketAlreadyExists)
import exporter


def ensure_bucket_exists_or_raise(minio_client):
    if minio_client is None:
        raise Exception("A minio client must be provided!")
    try:
        # TODO: Determine bucket name
        # TODO: Determine way to handle bucket location
        minio_client.make_bucket(
            "mlpipeline-visualizations", location="us-west-1")
    except BucketAlreadyOwnedByYou as err:
        pass
    except BucketAlreadyExists as err:
        pass
    except ResponseError as err:
        raise


def main(argv=None):
    parser = argparse.ArgumentParser(description='Visualization Generator')
    parser.add_argument("--type", type=str, default="roc",
                        help="Type of visualization to be generated.")
    parser.add_argument('--predictions', type=str,
                        help='GCS path of prediction file pattern.')
    args = parser.parse_args()

    schema_file = os.path.join(os.path.dirname(args.predictions), 'schema.json')
    schema = json.loads(file_io.read_file_to_string(schema_file))
    names = [x['name'] for x in schema]

    dfs = []
    files = file_io.get_matching_files(args.predictions)
    for file in files:
        with file_io.FileIO(file, 'r') as f:
            dfs.append(pd.read_csv(f, names=names))

    df = pd.concat(dfs)
    df["target"] = df.apply(eval(
        "lambda x: 1 if (x['target'] > x['fare'] * 0.2) else 0"), axis=1)
    # df['target'] = df['target'].apply(lambda x: 1 if x == "true" else 0)
    fpr, tpr, thresholds = roc_curve(df['target'], df["true"])
    df_roc = pd.DataFrame({'fpr': fpr, 'tpr': tpr, 'thresholds': thresholds})
    roc_file = os.path.join(os.getcwd(), 'roc.csv')
    with open(roc_file, 'w') as f:
        df_roc.to_csv(f, columns=["fpr", "tpr", "thresholds"], header=False,
                      index=False)

    # minio client use these to retrieve minio objects/artifacts
    minio_access_key = "minio"
    minio_secret_key = "minio123"
    minio_host = os.getenv("MINIO_SERVICE_SERVICE_HOST", "0.0.0.0")
    minio_port = os.getenv("MINIO_SERVICE_SERVICE_PORT", "9000")
    # construct minio endpoint from host and namespace (optional)
    minio_endpoint = "{}:{}".format(minio_host, minio_port)

    minio_client = Minio(minio_endpoint,
                         access_key=minio_access_key,
                         secret_key=minio_secret_key,
                         secure=False)

    ensure_bucket_exists_or_raise(minio_client=minio_client)

    # Generate HTML and save it to a file
    nb = exporter.code_to_notebook('./{}.py'.format(args.type))
    html = exporter.generate_html_from_notebook(nb)
    output_path = os.path.join(os.getcwd(), 'output.html')
    output = open(output_path, 'w')
    output.write(html)
    output.close()

    # Upload artifact to minio
    try:
        with open(output_path, 'rb') as file_data:
            file_stat = os.stat(output_path)
            # add some sort of id to the visualization html file
            minio_client.put_object('mlpipeline-visualizations', 'object.html',
                                    file_data, file_stat.st_size,
                                    content_type='text/html')
    except ResponseError as err:
        raise


if __name__ == "__main__":
    main()
