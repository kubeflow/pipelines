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
"""Anomaly detection component using TensorFlow Probability."""
import kfp
from kfp.v2.dsl import Dataset
from kfp.v2.dsl import Input
from kfp.v2.dsl import Output


def tfp_anomaly_detection(
    input_dataset: Input[Dataset],
    output_dataset: Output[Dataset],
    time_col: str = 'timestamp',
    feature_col: str = 'value',
    timestamp_format: str = '%Y-%m-%d %H:%M:%S',
    anomaly_threshold: float = 0.01,
    use_gibbs_predictive_dist: bool = True,
    num_warmup_steps: int = 50,
    num_samples: int = 100,
    jit_compile: bool = False,
    seed: int = None
):
    """Uses TFP STS detect_anomalies to regularize a time series, fit a model, and predict anomalies.

  Args:
    input_dataset: Input with GCS path to input time series csv.
    output_dataset: Output with GCS path to output predictions csv.
    time_col: Name of csv column with timestamps.
    feature_col: Name of csv column with feature values.
    timestamp_format: Datetime format to serialize timestamps with.
    anomaly_threshold: Confidence level for anomaly detection.
    use_gibbs_predictive_dist: Whether the predictive distribution is derived
      from Gibbs samples of the latent level.
    num_warmup_steps: Number of steps to take before collecting samples.
    num_samples: Number of steps to take while sampling parameter values.
    jit_compile: Whether to compile the sampler with XLA.
    seed: PRNG seed.

  Returns:
    Path to output predictions csv with the following fields
      timestamp: Timestamps from the input time series.
      value: Observed values from the input time series.
      anomaly_score: Probability that the data point is an anomaly.
      tail_probability: Probability that the data point occurs.
      label: Whether the data point is predicted to be an anomaly.
      lower_limit: Lowest acceptable forecast value from model.
      mean: Mean forecast value from model.
      upper_limit: Highest acceptable forecast value from model.
  """

    import pandas as pd
    import tensorflow.compat.v2 as tf
    import tensorflow_probability as tfp
    from tensorflow_probability.python.sts import anomaly_detection as tfp_ad
    from tensorflow_probability.python.sts.anomaly_detection.anomaly_detection_lib import PredictionOutput

    logger = tf.get_logger()

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

    def format_predictions(predictions: PredictionOutput) -> pd.DataFrame:
        """Saves predictions in a standardized csv format and fills missing values.

    Args:
      predictions: Anomaly detection output with fields times,
        observed_time_series, is_anomaly, tail_probabilities, lower_limit, mean,
        upper_limit.

    Returns:
      predictions_df: A formatted pandas DataFrame compatible with scoring on
      the Numenta Anomaly Benchmark.
    """
        anomaly_scores = 1 - predictions.tail_probabilities
        predictions_df = pd.DataFrame(
            data={
                'timestamp':
                    predictions.times.strftime(timestamp_format).tolist(),
                'value':
                    predictions.observed_time_series.numpy().tolist(),
                'anomaly_score':
                    anomaly_scores.numpy().tolist(),
                'tail_probability':
                    predictions.tail_probabilities.numpy().tolist(),
                'label':
                    predictions.is_anomaly.numpy().astype(int).tolist(),
                'lower_limit':
                    predictions.lower_limit.numpy().tolist(),
                'mean':
                    predictions.mean.numpy().tolist(),
                'upper_limit':
                    predictions.upper_limit.numpy().tolist(),
            }
        )
        return predictions_df

    data = load_data(input_dataset.path)
    logger.info(
        'Input dataset has {0} rows. If you run out of memory you should increase set_memory_limit in your pipeline.'
        .format(len(data))
    )
    predictions = tfp_ad.detect_anomalies(
        data, anomaly_threshold, use_gibbs_predictive_dist, num_warmup_steps,
        num_samples, jit_compile, seed
    )
    predictions_df = format_predictions(predictions)
    predictions_df.to_csv(output_dataset.path)


# TODO: Update tf-nightly and tfp install after official release of tf==2.6.0. and tfp==0.14.0.
def generate_component_file():
    packages = [
        'pandas', 'tf-nightly',
        'git+https://github.com/tensorflow/probability.git'
    ]
    kfp.components.create_component_from_func_v2(
        tfp_anomaly_detection,
        packages_to_install=packages,
        output_component_file='component.yaml'
    )
