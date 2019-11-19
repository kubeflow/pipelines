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

import mock
import unittest
import os

from kfp_component.google.storage import download_blob

DOWNLOAD_BLOB_MODULE = 'kfp_component.google.storage._download_blob'

@mock.patch(DOWNLOAD_BLOB_MODULE + '.os')
@mock.patch(DOWNLOAD_BLOB_MODULE + '.open')
@mock.patch(DOWNLOAD_BLOB_MODULE + '.storage.Client')
class DownloadBlobTest(unittest.TestCase):

    def test_download_blob_succeed(self, mock_storage_client,
        mock_open, mock_os):
        mock_os.path.dirname.return_value = '/foo'
        mock_os.path.exists.return_value = False

        download_blob('gs://foo/bar.py', 
            '/foo/bar.py')

        mock_blob = mock_storage_client().bucket().blob()
        mock_blob.download_to_file.assert_called_once()
        mock_os.makedirs.assert_called_once()