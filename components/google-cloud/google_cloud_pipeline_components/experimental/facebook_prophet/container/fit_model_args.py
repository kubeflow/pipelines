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
"""Argument parsing for fit model component"""
import argparse
from typing import Union


def str_bool_or_int(value: str) -> Union[str, bool, int]:
  if value.lower() == 'true' or value.lower() == 'false':
    return bool(value)
  if value.isdecimal():
    return int(value)
  if value == 'auto':
    return value
  raise app.UsageError(
      '"%s" was not one of "auto", "True", "False", or a number' % value)


parser = argparse.ArgumentParser(
    description='Time-series forecasting with Facebook Prophet')
parser.add_argument(
    '--data_source',
    type=str,
    help='The source of training data. Either the path to a CSV file, a GCS URI, or a BigQuery table.',
    required=True)
parser.add_argument(
    '--features',
    type=str,
    default=[],
    help='List of column names for additional regressors',
    nargs='+')
parser.add_argument(
    '--countries',
    type=str,
    default=[],
    help='List of two-letter country codes to include major holidays from',
    nargs='+')
parser.add_argument(
    '--n_changepoints',
    type=int,
    default=25,
    help='Number of potential changepoints to include selected uniformly from the first changepoint_range proportion of the history.'
)
parser.add_argument(
    '--changepoint_range',
    type=float,
    default=0.8,
    help='Proportion of history in which trend changepoints will be estimated. Defaults to 0.8 for the first 80%%.'
)
parser.add_argument(
    '--yearly_seasonality',
    type=str_bool_or_int,
    default='auto',
    help='Fit yearly seasonality. Can be "auto", True, False, or a number of Fourier terms to generate.'
)
parser.add_argument(
    '--weekly_seasonality',
    type=str_bool_or_int,
    default='auto',
    help='Fit weekly seasonality. Can be "auto", True, False, or a number of Fourier terms to generate.'
)
parser.add_argument(
    '--daily_seasonality',
    type=str_bool_or_int,
    default='auto',
    help='Fit daily seasonality. Can be "auto", True, False, or a number of Fourier terms to generate.'
)
parser.add_argument(
    '--seasonality_mode',
    type=str,
    default='additive',
    help='"additive" (default) or "multiplicative".')
parser.add_argument(
    '--seasonality_prior_scale',
    type=float,
    default=10,
    help='Parameter modulating the strength of the seasonality model. Larger values allow the model to fit larger seasonal fluctuations, smaller values dampen the seasonality. Can be specified for individual seasonalities using add_seasonality.'
)
parser.add_argument(
    '--holidays_prior_scale',
    type=float,
    default=10,
    help='Parameter modulating the strength of the holiday components model, unless overridden in the holidays input.'
)
parser.add_argument(
    '--changepoint_prior_scale',
    type=float,
    default=0.05,
    help='Parameter modulating the flexibility of the automatic changepoint selection. Large values will allow many changepoints, small values will allow few changepoints.'
)
parser.add_argument(
    '--mcmc_samples',
    type=int,
    default=0,
    help='Integer, if greater than 0, will do full Bayesian inference with the specified number of MCMC samples. If 0, will do MAP estimation.'
)
parser.add_argument(
    '--interval_width',
    type=float,
    default=0.8,
    help='Float, width of the uncertainty intervals provided for the forecast. If mcmc_samples=0, this will be only the uncertainty in the trend using the MAP estimate of the extrapolated generative model. If mcmc.samples>0, this will be integrated over all model parameters, which will include uncertainty in seasonality.'
)
parser.add_argument(
    '--uncertainty_samples',
    type=int,
    default=1000,
    help='Number of simulated draws used to estimate uncertainty intervals. Settings this value to 0 or False will disable uncertainty estimation and speed up the calculation.'
)
parser.add_argument(
    '--stan_backend',
    type=str,
    default=None,
    help='str as defined in StanBackendEnum default: None - will try to iterate over all available backends and find the working one.'
)
parser.add_argument(
    '--cross_validation_horizon',
    type=str,
    default=None,
    help='When set enables cross-validation for time series. The value is a string with pd.Timedelta compatible style, e.g., "5 days", "3 hours", "10 seconds"'
)
parser.add_argument(
    '--model',
    type=str,
    help='JSON file path to save fitted model to.',
    required=True)
parser.add_argument(
    '--cross_validation_results',
    type=str,
    help='A table with the forecast, actual value and cutoff from cross-validation. This output is only generated when cross_validation_horizon is set',
)
