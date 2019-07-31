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

from pathlib import PurePath

class GCSHelper(object):
  """ GCSHelper manages the connection with the GCS storage """

  @staticmethod
  def get_blob_from_gcs_uri(gcs_path):
    from google.cloud import storage
    pure_path = PurePath(gcs_path)
    gcs_bucket = pure_path.parts[1]
    gcs_blob = '/'.join(pure_path.parts[2:])
    client = storage.Client()
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(gcs_blob)
    return blob

  @staticmethod
  def upload_gcs_file(local_path, gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.upload_from_filename(local_path)

  @staticmethod
  def remove_gcs_blob(gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.delete()

  @staticmethod
  def download_gcs_blob(local_path, gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.download_to_filename(local_path)
