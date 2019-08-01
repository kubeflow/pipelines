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

import tensorflow_data_validation as tfdv

# The following variables are provided through dependency injection. These
# variables come from the specified input path and arguments provided by the
# API post request.
#
# input_path

train_stats = tfdv.generate_statistics_from_csv(data_location=input_path)

tfdv.visualize_statistics(train_stats)
