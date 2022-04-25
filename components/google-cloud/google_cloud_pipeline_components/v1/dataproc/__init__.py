# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Google Cloud Pipeline Dataproc Batch components."""

import os

try:
  from kfp.v2.components import load_component_from_file
except ImportError:
  from kfp.components import load_component_from_file

__all__ = [
    'DataprocPySparkBatchOp',
    'DataprocSparkBatchOp',
    'DataprocSparkRBatchOp',
    'DataprocSparkSqlBatchOp'
]

DataprocPySparkBatchOp = load_component_from_file(
        os.path.join(os.path.dirname(__file__), 'create_pyspark_batch/component.yaml'))

DataprocSparkBatchOp = load_component_from_file(
        os.path.join(os.path.dirname(__file__), 'create_spark_batch/component.yaml'))

DataprocSparkRBatchOp = load_component_from_file(
        os.path.join(os.path.dirname(__file__), 'create_spark_r_batch/component.yaml'))

DataprocSparkSqlBatchOp = load_component_from_file(
        os.path.join(os.path.dirname(__file__), 'create_spark_sql_batch/component.yaml'))
