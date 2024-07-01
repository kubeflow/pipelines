# Copyright 2021 Google LLC. All Rights Reserved.
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
import argparse

parser = argparse.ArgumentParser(
    description='Perform a time-series forecast using a trained Facebook Prophet model.'
)

parser.add_argument(
    '--model',
    type=str,
    help='Path to a trained Prophet model serialized in JSON format',
    required=True)
prediction_range_group = parser.add_mutually_exclusive_group(required=True)
prediction_range_group.add_argument(
    '--future_data_source',
    help='A JSON string or a path to a local CSV file representing the table to make predictions on. Table must have a column named "ds" containing timestamps in a format recognized by pandas. Additionally, if additional regressors were used to train the model, they must also exist in the frame. A prediction will be made for every timestamp present in the table',
    type=str)
prediction_range_group.add_argument(
    '--periods', help='Number of future periods to predict', type=int)
parser.add_argument(
    '--prediction_output_file_path',
    type=str,
    help='Path to a CSV file containing the prediction',
    required=True)
