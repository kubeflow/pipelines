# Copyright 2021 The TensorFlow Probability Authors.
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

from typing import NamedTuple


def tfp_anomaly_detection(
    num_samples: int = 50,
    max_num_steps: int = 1000,
    jit_compile: bool = True,
    seed: int = None,
    anomaly_threshold: float = 0.01
) -> NamedTuple('PredictionOutput',
                [('time', list), ('observed_series', list), ('all_times', list),
                 ('mean', list), ('upper_limit', list), ('lower_limit', list),
                 ('anomalies', list), ('pvalues', list)]):
  """Uses TFP STS to regularize a time series, fit a model, and predict anomalies.

  Args:
    num_samples: Number of data points to sample from the posterior
      distribution.
    max_num_steps: Number of steps to run the optimizer.
    jit_compile: If True, compiles the loss function and gradient update using
      XLA.
    seed: The random seed to use for sampling.
    anomaly_threshold: Any data point with a pvalue lower than this threshold is
      labelled anomalous.

  Returns:
    time: Timestamps where each index aligns with the indices of the other
    outputs.
    all_times: Same as time for anomaly detection but contains more timestamps
    for forecasting.
    anomalies: Indices of timestamps with anomalies.
    lower_limit: Lowest forecast value computed when fitting the model.
    mean: Mean of the distribution computed when fitting the model.
    upper_limit: Highest forecast value computed when fitting the model.
    observed_series: The time series that was originally input.
    pvalues: For each point, the tail-area probability of the data point.
  """

  import itertools
  import numpy as np
  import pandas as pd
  import tensorflow.compat.v2 as tf
  import tensorflow_probability as tfp
  from tensorflow_probability.python.sts import default_model
  from tensorflow_probability.python.sts.internal import util as sts_util
  from tensorflow_probability.python.sts import one_step_predictive

  def load_data() -> pd.Series:
    """Generates synthetic data with anomalies.

    Returns:
      Pandas dataframe with synthetic data.
    """
    times = pd.date_range(
        start='2020-01-05 00:00:00',
        end='2020-01-12 00:00:00',
        freq=pd.DateOffset(hours=1))

    xs = np.arange(len(times))
    ys = 8 * np.sin(1. / 24. * 2 * np.pi * xs)
    ys += 4 * np.cos(1. / 6 * 2 * np.pi * xs)
    ys += 0.001 * (xs - 70)**2
    ys += 3 * np.random.randn(len(ys))
    ys[137] += 16.0
    ys[138] += 14.0

    return pd.DataFrame(data={'value': ys}, index=times)

  def detect_anomalies(
      regularized_series: pd.Series, model: tfp.sts.StructuralTimeSeries,
      posterior_samples: OrderedDict[str, tf.Tensor], anomaly_threshold: float
  ) -> NamedTuple('PredictionOutput', [('time', list), (
      'observed_series', list), ('all_times', list), (
          'mean', list), ('upper_limit', list), ('lower_limit', list),
                                       ('anomalies', list), ('pvalues', list)]):
    """Given a model, posterior, and anomaly_threshold, identifies anomalous time points in a time series.

    Args:
      regularized_series: A time series with a regular frequency.
      model: A fitted model that approximates the time series.
      posterior_samples: Posterior samples of model parameters.
      anomaly_threshold: The anomaly threshold passed as a parameter in the
        outer function.

    Returns:
      PredictionOutput with the distribution of acceptable time series values
      and detected anomalies.
    """
    [observed_time_series, mask
    ] = sts_util.canonicalize_observed_time_series_with_mask(regularized_series)

    anomaly_threshold = tf.convert_to_tensor(
        anomaly_threshold, dtype=observed_time_series.dtype)

    # The one-step predictive distribution covers the final `T - 1` timesteps.
    predictive_dist = one_step_predictive(
        model,
        regularized_series[:-1],
        posterior_samples,
        timesteps_are_event_shape=False)
    observed_series = observed_time_series[..., 1:, 0]
    times = regularized_series.index[1:]

    # Compute the tail probabilities (pvalues) of the observed series.
    prob_lower = predictive_dist.cdf(observed_series)
    tail_probabilities = 2 * tf.minimum(prob_lower, 1 - prob_lower)

    # Since quantiles of a mixture distribution are not analytically available,
    # use scalar root search to compute the upper and lower bounds.
    predictive_mean = predictive_dist.mean()
    predictive_stddev = predictive_dist.stddev()
    target_log_cdfs = tf.stack(
        [
            tf.math.log(anomaly_threshold / 2.) *
            tf.ones_like(predictive_mean),  # Target log CDF at lower bound.
            tf.math.log1p(-anomaly_threshold / 2.) *
            tf.ones_like(predictive_mean)  # Target log CDF at upper bound.
        ],
        axis=0)
    limits, _, _ = tfp.math.find_root_chandrupatla(
        objective_fn=lambda x: target_log_cdfs - predictive_dist.log_cdf(x),
        low=tf.stack([
            predictive_mean - 100 * predictive_stddev,
            predictive_mean - 10 * predictive_stddev
        ],
                     axis=0),
        high=tf.stack([
            predictive_mean + 10 * predictive_stddev,
            predictive_mean + 100 * predictive_stddev
        ],
                      axis=0))

    # Identify anomalies.
    anomalies = np.less(tail_probabilities, anomaly_threshold)
    if mask is not None:
      anomalies = np.logical_and(anomalies, ~mask)
    observed_anomalies = list(
        itertools.compress(range(len(times)), list(anomalies)))

    times = times.strftime('%Y-%m-%d %H:%M:%S').tolist()
    observed_series = observed_series.numpy().tolist()
    predictive_mean = predictive_mean.numpy().tolist()
    lower_limit = limits[0].numpy().tolist()
    upper_limit = limits[1].numpy().tolist()
    tail_probabilities = tail_probabilities.numpy().tolist()

    return (times, times, observed_series, predictive_mean, lower_limit,
            upper_limit, observed_anomalies, tail_probabilities)

  data = load_data()
  regularized_series = tfp.sts.regularize_series(data)

  # TODO: Add seed to fit_surrogate_posterior.
  model = default_model.build_default_model(regularized_series)
  surrogate_posterior = tfp.sts.build_factored_surrogate_posterior(
      model, seed=seed)
  _ = tfp.vi.fit_surrogate_posterior(
      target_log_prob_fn=model.joint_log_prob(regularized_series),
      surrogate_posterior=surrogate_posterior,
      optimizer=tf.optimizers.Adam(0.1),
      num_steps=max_num_steps,
      jit_compile=jit_compile,
      convergence_criterion=(
          tfp.optimizer.convergence_criteria.SuccessiveGradientsAreUncorrelated(
              window_size=20, min_num_steps=50)))

  posterior_samples = surrogate_posterior.sample(num_samples, seed=seed)
  return detect_anomalies(regularized_series, model, posterior_samples,
                          anomaly_threshold)
