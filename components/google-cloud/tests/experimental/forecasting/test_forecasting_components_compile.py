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

import os

from google_cloud_pipeline_components import aiplatform
from google_cloud_pipeline_components.experimental import forecasting
import kfp
from kfp.v2 import compiler

import unittest


class ForecastingComponentsCompileTest(unittest.TestCase):
    """Tests for forecasting components."""

    def setUp(self):
        super().setUp()
        self._project = 'test_project'
        self._display_name = 'test_display_name'
        self._package_path = 'pipeline.json'
        self._location = 'us-central1'
        self._bq_source = 'bq://test_project.test_dataset.training_input'

    def tearDown(self):
        super().tearDown()
        if os.path.exists(self._package_path):
            os.remove(self._package_path)

    def test_tabular_data_pipeline_component_ops_compile(self):
        """Checks that a training pipeline compiles successfully."""
        input_tables = [self._bq_source]
        model_feature_columns = ['col1', 'col2']

        @kfp.dsl.pipeline(name='forecasting-training')
        def pipeline():
            training_validation = forecasting.ForecastingValidationOp(
                input_tables=input_tables,
                validation_theme='FORECASTING_TRAINING')
            training_preprocess = forecasting.ForecastingPreprocessingOp(
                project=self._project,
                input_tables=input_tables,
                preprocessing_bigquery_dataset='bq://project.dataset')
            training_preprocess.after(training_validation)
            prepare_data_for_train_op = forecasting.ForecastingPrepareDataForTrainOp(
                input_tables=input_tables,
                preprocess_metadata=(
                        training_preprocess.outputs['preprocess_metadata']),
                model_feature_columns=model_feature_columns)
            dataset_create_op = aiplatform.TimeSeriesDatasetCreateOp(
                display_name=self._display_name,
                bq_source=self._bq_source,
                project=self._project,
                location=self._location)
            aiplatform.AutoMLForecastingTrainingJobRunOp(
                display_name=self._display_name,
                time_series_identifier_column=(
                  prepare_data_for_train_op.outputs[
                          'time_series_identifier_column']),
                time_series_attribute_columns=(
                  prepare_data_for_train_op.outputs[
                          'time_series_attribute_columns']),
                available_at_forecast_columns=(
                  prepare_data_for_train_op.outputs[
                          'available_at_forecast_columns']),
                unavailable_at_forecast_columns=(
                  prepare_data_for_train_op.outputs[
                          'unavailable_at_forecast_columns']),
                column_transformations=(
                  prepare_data_for_train_op.outputs[
                          'column_transformations']),
                dataset=dataset_create_op.outputs['dataset'],
                target_column=(
                        prepare_data_for_train_op.outputs['target_column']),
                time_column=prepare_data_for_train_op.outputs['time_column'],
                forecast_horizon=7,
                data_granularity_unit=(
                    prepare_data_for_train_op.outputs['data_granularity_unit']),
                data_granularity_count=(
                    prepare_data_for_train_op.outputs[
                            'data_granularity_count']),
                budget_milli_node_hours=1000,
                project=self._project,
                location=self._location,
                optimization_objective='minimize-rmse',
                additional_experiments='["enable_model_compression"]')

        compiler.Compiler().compile(
            pipeline_func=pipeline, package_path=self._package_path
        )
