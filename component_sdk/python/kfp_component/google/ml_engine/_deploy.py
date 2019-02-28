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
import os

from fire import decorators

from google.cloud import storage
from .. import common as gcp_common
from ..storage import parse_blob_path
from ._create_model import create_model
from ._create_version import create_version
from ._set_default_version import set_default_version

@decorators.SetParseFns(python_version=str, runtime_version=str)
def deploy(model_uri, project_id, model_id=None, version_id=None, 
    runtime_version=None, python_version=None, version=None, 
    replace_existing_version=False, set_default=False, wait_interval=30):
    """Deploy a model to MLEngine from GCS URI

    Args:
        model_uri (str): required, the GCS URI which contains a model file.
            Common used TF model search path (export/exporter) will be used 
            if exist. 
        project_id (str): required, the ID of the parent project.
        model_id (str): optional, the user provided name of the model.
        version_id (str): optional, the user provided name of the version. 
            If it is not provided, the operation uses a random name.
        runtime_version (str): optinal, the Cloud ML Engine runtime version 
            to use for this deployment. If not set, Cloud ML Engine uses 
            the default stable version, 1.0. 
        python_version (str): optinal, the version of Python used in prediction. 
            If not set, the default version is '2.7'. Python '3.5' is available
            when runtimeVersion is set to '1.4' and above. Python '2.7' works 
            with all supported runtime versions.
        version (str): optional, the payload of the new version.
        replace_existing_version (boolean): boolean flag indicates whether to replace 
            existing version in case of conflict.
        set_default (boolean): boolean flag indicates whether to set the new
            version as default version in the model.
        wait_interval (int): the interval to wait for a long running operation.
    """
    model_uri = _search_tf_model_common_exporter_dir(model_uri)
    gcp_common.dump_file('/tmp/kfp/output/ml_engine/model_uri.txt', 
        model_uri)
    model = create_model(project_id, model_id)
    model_name = model.get('name')
    version = create_version(model_name, model_uri, version_id,
        runtime_version, python_version, version, replace_existing_version,
        wait_interval)
    if set_default:
        version_name = version.get('name')
        version = set_default_version(version_name)
    return version

def _search_tf_model_common_exporter_dir(model_uri):
    bucket_name, blob_name = parse_blob_path(model_uri)
    storage_client = storage.Client()
    bucket = storage_client.get_bucket(bucket_name)
    exporter_path = os.path.join(blob_name, 'export/exporter/')
    iterator = bucket.list_blobs(prefix=exporter_path, delimiter='/')
    for _ in iterator.pages:
        # Iterate to the last page
        pass
    if iterator.prefixes:
        prefixes = list(iterator.prefixes)
        prefixes.sort(reverse=True)
        return 'gs://{}/{}'.format(bucket_name, prefixes[0])
    return model_uri
