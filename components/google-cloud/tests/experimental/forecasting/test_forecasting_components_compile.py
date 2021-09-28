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
"""Test forecasting components to ensure they compile without error."""

import unittest

import os
from google_cloud_pipeline_components.aiplatform import TimeSeriesDatasetCreateOp
from google_cloud_pipeline_components.experimental.forecasting import ForecastingTrainingWithExperimentsOp
import kfp
from kfp.v2 import compiler


class ForecastingComponetsCompileTest(unittest.TestCase):

    def setUp(self):
        super(ForecastingComponetsCompileTest, self).setUp()
        self._project = 'test_project'
        self._display_name = 'test_display_name'
        self._package_path = 'pipeline.json'
        self._location = 'us-central1'
        self._bq_source = 'bq://test_project.test_dataset.training_input'

    def tearDown(self):
        if os.path.exists(self._package_path):
            os.remove(self._package_path)

    def test_tabular_data_pipeline_component_ops_compile(self):

        @kfp.dsl.pipeline(name='forecasting-training')
        def pipeline():
            dataset_create_op = TimeSeriesDatasetCreateOp(
                display_name=self._display_name,
                bq_source=self._bq_source,
                project=self._project,
                location=self._location,
            )

            train_op = ForecastingTrainingWithExperimentsOp(
                display_name=self._display_name,
                time_series_identifier_column='datetime',
                time_series_attribute_columns='["location_id", "product_id"]',
                available_at_forecast_columns='["datetime", "year", "ml_use"]',
                unavailable_at_forecast_columns='["gross_quantity"]',
                column_transformations=(
                    '[{"numeric": {"column_name": "gross_quantity"}}]'
                ),
                dataset=dataset_create_op.outputs['dataset'],
                target_column='gross_quantity',
                time_column='datetime',
                forecast_horizon=7,
                data_granularity_unit='day',
                data_granularity_count=1,
                budget_milli_node_hours=1000,
                project=self._project,
                location=self._location,
                optimization_objective='minimize-rmse',
                additional_experiments='["enable_model_compression"]',
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline, package_path=self._package_path
        )
