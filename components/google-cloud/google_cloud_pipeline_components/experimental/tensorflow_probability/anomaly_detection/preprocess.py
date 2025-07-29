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
# ============================================================================
"""Preprocess component for time series data."""
import kfp
from kfp.v2.dsl import Dataset
from kfp.v2.dsl import Input
from kfp.v2.dsl import Output


def preprocess(
    input_dataset: Input[Dataset],
    preprocessed_dataset: Output[Dataset],
    time_col: str = 'timestamp',
    feature_col: str = 'value'
):
    """Regularizes and resamples the input dataset.

  Args:
    input_dataset: Input with GCS path to input time series csv.
    preprocessed_dataset: Output with GCS path to preprocessed csv.
    time_col: Name of csv column with timestamps.
    feature_col: Name of csv column with feature values.

  Returns:
    A preprocessed time series dataframe that may have fewer rows than the
    input time series due to regularization and resampling.
  """
    import pandas as pd
    import tensorflow_probability as tfp
    from tensorflow_probability.python.sts.internal.seasonality_util import freq_to_seconds

    def load_data(path: str) -> pd.DataFrame:
        """Loads pandas dataframe from csv.

    Args:
      path: Path to the csv file.

    Returns:
      A time series dataframe compatible with TFP functions.
    """
        original_df = pd.read_csv(path)
        df = pd.DataFrame()
        df['timestamp'] = pd.to_datetime(original_df[time_col])
        df['value'] = original_df[feature_col].astype('float32')
        df = df.set_index('timestamp')
        return df

    def compare_freq(a: pd.DateOffset, b: pd.DateOffset) -> bool:
        """A comparator for pd.DateOffset objects."""
        return freq_to_seconds(a) < freq_to_seconds(b)

    data = load_data(input_dataset.path)
    data = tfp.sts.regularize_series(data)

    # Resample data with less than daily granularity to have daily frequency.
    # This is because TFP works better with larger granularity data.
    # TODO: Turn resampling frequency into a tunable hyperparameter.
    if compare_freq(data.index.freq, pd.DateOffset(days=1)):
        data = data.resample('D').sum()

    data = data.reset_index()
    data = data.rename({'index': time_col, 'value': feature_col}, axis=1)

    data.to_csv(preprocessed_dataset.path)


# TODO: Update tf-nightly and tfp install after official release of tf==2.6.0. and tfp==0.14.0.
def generate_component_file():
    packages = [
        'pandas', 'tf-nightly',
        'git+https://github.com/tensorflow/probability.git'
    ]
    kfp.components.create_component_from_func_v2(
        preprocess,
        packages_to_install=packages,
        output_component_file='preprocess.yaml'
    )
