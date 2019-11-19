# Copyright 2018 Google LLC
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

import logging
import os

from google.cloud import storage
from ._common_ops import parse_blob_path

def download_blob(source_blob_path, destination_file_path):
    """Downloads a blob from the bucket.
    
    Args:
        source_blob_path (str): the source blob path to download from.
        destination_file_path (str): the local file path to download to.
    """
    bucket_name, blob_name = parse_blob_path(source_blob_path)
    storage_client = storage.Client()
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(blob_name)

    dirname = os.path.dirname(destination_file_path)
    if not os.path.exists(dirname):
        os.makedirs(dirname)

    with open(destination_file_path, 'wb+') as f:
        blob.download_to_file(f)

    logging.info('Blob {} downloaded to {}.'.format(
        source_blob_path,
        destination_file_path))