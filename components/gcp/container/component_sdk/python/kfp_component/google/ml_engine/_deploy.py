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

from fire import decorators

from google.cloud import storage
from .. import common as gcp_common
from ..storage import parse_blob_path
from ._create_model import create_model
from ._create_version import create_version
from ._set_default_version import set_default_version

KNOWN_MODEL_NAMES = ['saved_model.pb', 'saved_model.pbtext', 'model.pkl', 'model.pkl', 'model.pkl']

@decorators.SetParseFns(python_version=str, runtime_version=str)
def deploy(model_uri, project_id,
    model_uri_output_path, model_name_output_path, version_name_output_path,
    model_id=None, version_id=None, 
    runtime_version=None, python_version=None, model=None, version=None, 
    replace_existing_version=False, set_default=False, wait_interval=30):
    """Deploy a model to MLEngine from GCS URI

    Args:
        model_uri (str): Required, the GCS URI which contains a model file.
            If no model file is found, the same path will be treated as an export
            base directory of a TF Estimator. The last time-stamped sub-directory
            will be chosen as model URI.
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
        model (dict): Optional, the JSON payload of the new model. The schema follows 
            [REST Model resource](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models).
        version (dict): Optional, the JSON payload of the new version. The schema follows
            the [REST Version resource](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models.versions)
        replace_existing_version (boolean): boolean flag indicates whether to replace 
            existing version in case of conflict.
        set_default (boolean): boolean flag indicates whether to set the new
            version as default version in the model.
        wait_interval (int): the interval to wait for a long running operation.
    """
    storage_client = storage.Client()
    model_uri = _search_dir_with_model(storage_client, model_uri)
    gcp_common.dump_file(model_uri_output_path, model_uri)
    model = create_model(project_id, model_id, model,
        model_name_output_path=model_name_output_path,
    )
    model_name = model.get('name')
    version = create_version(model_name, model_uri, version_id,
        runtime_version, python_version, version, replace_existing_version,
        wait_interval, version_name_output_path=version_name_output_path,
    )
    if set_default:
        version_name = version.get('name')
        version = set_default_version(version_name)
    return version

def _search_dir_with_model(storage_client, model_root_uri):
    bucket_name, blob_name = parse_blob_path(model_root_uri)
    bucket = storage_client.bucket(bucket_name)
    if not blob_name.endswith('/'):
        blob_name += '/'
    it = bucket.list_blobs(prefix=blob_name, delimiter='/')
    for resource in it:
        basename = os.path.basename(resource.name)
        if basename in KNOWN_MODEL_NAMES:
            logging.info('Found model file under {}.'.format(model_root_uri))
            return model_root_uri
    model_dir = _search_tf_export_dir_base(storage_client, bucket, blob_name)
    if not model_dir:
        model_dir = model_root_uri
    return model_dir

def _search_tf_export_dir_base(storage_client, bucket, export_dir_base):
    logging.info('Searching model under export base dir: {}.'.format(export_dir_base))
    it = bucket.list_blobs(prefix=export_dir_base, delimiter='/')
    for _ in it.pages:
        # Iterate to the last page to get the full prefixes.
        pass
    timestamped_dirs = []
    for sub_dir in it.prefixes:
        dir_name = os.path.basename(os.path.normpath(sub_dir))
        if dir_name.isdigit():
            timestamped_dirs.append(sub_dir)

    if not timestamped_dirs:
        logging.info('No timestamped sub-directory is found under {}'.format(export_dir_base))
        return None

    last_timestamped_dir = max(timestamped_dirs)
    logging.info('Found timestamped sub-directory: {}.'.format(last_timestamped_dir))
    return 'gs://{}/{}'.format(bucket.name, last_timestamped_dir)
