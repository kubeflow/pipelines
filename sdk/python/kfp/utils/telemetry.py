# Copyright 2020 Google LLC
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
"""Utilities for KFP usage beacon instruments."""

import frozendict

from typing import Optional, Text

# Mapping from the component url suffix to component names.
_URL_SUFFIX_TO_OOB_COMPONENT_MAP = frozendict.frozendict({
    # AutoML components
    'gcp/automl/create_dataset_for_tables/component.yaml':
        'automl_create_dataset',
    'gcp/automl/create_model_for_tables/component.yaml':
        'automl_create_model',
    'gcp/automl/export_data_to_gcs/component.yaml':
        'automl_export_data',
    'gcp/automl/import_data_from_bigquery/component.yaml':
        'automl_import_bq',
    'gcp/automl/import_data_from_gcs/component.yaml':
        'automl_import_gcs',
    'gcp/automl/prediction_service_batch_predict/component.yaml':
        'automl_batch_predict',
    'gcp/automl/split_dataset_table_column_names/component.yaml':
        'automl_split_table_column',
    # BigQuery component
    'gcp/bigquery/query/component.yaml':
        'bigquery_query',
    # Dataflow components
    'gcp/dataflow/launch_python/component.yaml':
        'dataflow_launch_python',
    'gcp/dataflow/launch_template/component.yaml':
        'dataflow_launch_template',
    # DataProc components
    'gcp/dataproc/create_cluster/component.yaml':
        'dataproc_create_cluster',
    'gcp/dataproc/delete_cluster/component.yaml':
        'dataproc_delete_cluster',
    'gcp/dataproc/submit_hadoop_job/component.yaml':
        'dataproc_submit_hadoop',
    'gcp/dataproc/submit_hive_job/component.yaml':
        'dataproc_submit_hive',
    'gcp/dataproc/submit_pig_job/component.yaml':
        'dataproc_submit_pig',
    'gcp/dataproc/submit_pyspark_job/component.yaml':
        'dataproc_submit_pyspark',
    'gcp/dataproc/submit_spark_job/component.yaml':
        'dataproc_submit_spark',
    'gcp/dataproc/submit_sparksql_job/component.yaml':
        'dataproc_submit_sparksql',
    # CMLE (aka CAIP training) components
    'gcp/ml_engine/batch_predict/component.yaml': 'cmle_batch_predict',
    'gcp/ml_engine/deploy/component.yaml': 'cmle_deploy',
    'gcp/ml_engine/train/component.yaml': 'cmle_train',
    # Google Cloud storage components
    'google-cloud/storage/download/component.yaml': 'gcs_download',
    'google-cloud/storage/download_blob/component.yaml': 'gcs_download_blob',
    'google-cloud/storage/download_dir/component.yaml': 'gcs_download_dir',
    'google-cloud/storage/list/component.yaml': 'gcs_list',
    'google-cloud/storage/upload_to_explicit_uri/component.yaml': 'gcs_upload_explicit',
    'google-cloud/storage/upload_to_unique_uri/component.yaml': 'gcs_upload_unique',
    # Local viz tool components
    'local/confusion_matrix/component.yaml': 'confusion_matrix',
    'local/roc/component.yaml': 'roc',
    # Tensorflow components
    'tensorflow/tensorboard/prepare_tensorboard/component.yaml': 'tensorflow_prepare_tensorboard',
    # Diagnostic components
    'diagnostics/diagnose_me/component.yaml': 'diagnose_me',
    # File system components
    'filesystem/get_file/component.yaml': 'filesystem_get_file',
    'filesystem/get_subdirectory/component.yaml': 'filesystem_get_subdir',
    'filesystem/list_items/component.yaml': 'filesystem_list_items',
    # Dataset components
    'datasets/Chicago_Taxi_Trips/component.yaml': 'dataset_chicago_taxi',
    # XGBoost components
    'XGBoost/Predict/component.yaml': 'xgboost_predict',
    'XGBoost/Train/component.yaml': 'xgboost_train',
    # Git components
    'git/clone/component.yaml': 'git_clone',
})


def get_component_name(uri: Text) -> Optional[Text]:
  """Gets the OOB component name according to its uri.
  
  :param uri: uri used in component loading.
  :return: component name if it's an OOB component, otherwise None.
  """
  for k, v in _URL_SUFFIX_TO_OOB_COMPONENT_MAP.items():
    if k in uri:
      return _URL_SUFFIX_TO_OOB_COMPONENT_MAP[k]
  return None