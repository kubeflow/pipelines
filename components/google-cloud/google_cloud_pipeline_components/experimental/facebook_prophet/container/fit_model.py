#!/usr/bin/env python3
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
"""Pipeline component for fitting a Facebook Prophet model

For a detailed description of the input and output of this component see
fit_model_component.yaml
"""
import json
import pathlib

from fit_model_args import parser
from prophet import Prophet
from prophet.serialize import model_to_json
from prophet.diagnostics import cross_validation
from util import DataSource
from typing import Dict, Any, List, Optional, Tuple
import pandas as pd
import os


def fit_model(
    data_source: str,
    prophet_constructor_args: Dict[str, Any],
    features: List[str] = [],
    countries: List[str] = [],
    cv_horizon: Optional[str] = None,
) -> Tuple[Prophet, Optional[pd.DataFrame]]:
  df = DataSource(data_source).load()
  m = Prophet(**prophet_constructor_args)

  for regressor in features:
    m.add_regressor(regressor)

  for country in countries:
    m.add_country_holidays(country_name=country)

  m.fit(df)

  return m, cross_validation(m, cv_horizon) if cv_horizon is not None else None


if __name__ == '__main__':
  args = parser.parse_args()

  # If lists are passed as JSON arrays decode them
  if len(args.features) == 1:
    try:
      features = json.loads(args.features[0])
      args.features = features
    except json.decoder.JSONDecodeError:
      pass
  if len(args.countries) == 1:
    try:
      countries = json.loads(args.countries[0])
      args.countries = countries
    except json.decoder.JSONDecodeError:
      pass

  prophet_constructor_args = {
      'n_changepoints': args.n_changepoints,
      'changepoint_range': args.changepoint_range,
      'yearly_seasonality': args.yearly_seasonality,
      'weekly_seasonality': args.weekly_seasonality,
      'daily_seasonality': args.daily_seasonality,
      'seasonality_mode': args.seasonality_mode,
      'seasonality_prior_scale': args.seasonality_prior_scale,
      'holidays_prior_scale': args.holidays_prior_scale,
      'changepoint_prior_scale': args.changepoint_prior_scale,
      'mcmc_samples': args.mcmc_samples,
      'interval_width': args.interval_width,
      'uncertainty_samples': args.uncertainty_samples,
      'stan_backend': args.stan_backend
  }

  model, cv_results = fit_model(args.data_source, prophet_constructor_args,
                                args.features, args.countries, args.cross_validation_horizon)

  os.makedirs(pathlib.Path(args.model).parent.absolute(), exist_ok=True)
  if args.cross_validation_results is not None:
    os.makedirs(pathlib.Path(args.cross_validation_results).parent.absolute(), exist_ok=True)

  with open(args.model, 'w') as f:
    json.dump(model_to_json(model), f)

  if cv_results is not None and args.cross_validation_results is not None:
    cv_results.to_csv(args.cross_validation_results)
