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
import re

def is_gcs_path(path):
    """Check if the path is a gcs path"""
    return path.startswith('gs://')

def parse_blob_path(path):
    """Parse a gcs path into bucket name and blob name

    Args:
        path (str): the path to parse.

    Returns:
        (bucket name in the path, blob name in the path)

    Raises:
        ValueError if the path is not a valid gcs blob path.

    Example:

        `bucket_name, blob_name = parse_blob_path('gs://foo/bar')`
        `bucket_name` is `foo` and `blob_name` is `bar`
    """
    match = re.match('gs://([^/]+)/(.+)$', path)
    if match:
        return match.group(1), match.group(2)
    raise ValueError('Path {} is invalid blob path.'.format(
        path))