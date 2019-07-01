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
from nbformat.v4 import new_notebook
from minio import Minio
from minio.error import (ResponseError, BucketAlreadyOwnedByYou,
                         BucketAlreadyExists)
import exporter


def ensure_bucket_exists_or_raise(minio_client):
    if minio_client is None:
        raise Exception('A minio client must be provided!')
    try:
        # TODO: Determine bucket name
        # TODO: Determine way to handle bucket location
        minio_client.make_bucket(
            'mlpipeline-visualizations', location='us-west-1')
    except BucketAlreadyOwnedByYou as err:
        pass
    except BucketAlreadyExists as err:
        pass
    except ResponseError as err:
        raise


def main(argv=None):
    parser = argparse.ArgumentParser(description='Visualization Generator')
    parser.add_argument('--type', type=str, default='roc',
                        help='Type of visualization to be generated.')
    parser.add_argument('--trueclass', type=str, default='true',
                        help='The name of the class as true value. If' +
                             'missing, assuming it is binary classification ' +
                             'and default to "true".')
    parser.add_argument('--predictions', type=str,
                        help='GCS path of prediction file pattern.')
    parser.add_argument('--target_lambda', type=str,
                        help='a lambda function as a string to determine ' +
                             'positive or negative. For example, "lambda x: ' +
                             'x[\'a\'] and x[\'b\']".')
    parser.add_argument('--true_score_column', type=str, default='true',
                        help='The name of the column for positive probability.')
    args = parser.parse_args()

    # minio client use these to retrieve minio objects/artifacts
    minio_access_key = 'minio'
    minio_secret_key = 'minio123'
    minio_host = os.getenv('MINIO_SERVICE_SERVICE_HOST', '0.0.0.0')
    minio_port = os.getenv('MINIO_SERVICE_SERVICE_PORT', '9000')
    # construct minio endpoint from host and namespace (optional)
    minio_endpoint = '{}:{}'.format(minio_host, minio_port)

    minio_client = Minio(minio_endpoint,
                         access_key=minio_access_key,
                         secret_key=minio_secret_key,
                         secure=False)

    ensure_bucket_exists_or_raise(minio_client)

    nb = new_notebook()
    nb.cells.append(exporter.create_cell_from_args(args))
    nb.cells.append(exporter.create_cell_from_file('./{}.py'.format(args.type)))
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


if __name__ == '__main__':
    main()
