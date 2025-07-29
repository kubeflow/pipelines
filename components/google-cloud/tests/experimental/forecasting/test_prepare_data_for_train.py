
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
"""Test Vertex Forecasting prepare data for train module."""

import json
import unittest

from google_cloud_pipeline_components.experimental.forecasting.prepare_data_for_train.component import prepare_data_for_train


class PrepareForTrainTests(unittest.TestCase):

    def _primary_table_specs(self):
        return {
            'bigquery_uri': 'bq://my_project.my_dataset.sales',
            'table_type': 'FORECASTING_PRIMARY',
            'forecasting_primary_table_metadata': {
                'time_column': 'datetime',
                'target_column': 'gross_quantity',
                'time_series_identifier_columns': ['product_id', 'location_id'],
                'unavailable_at_forecast_columns': ['sale_dollars', 'state_bottle_cost', 'state_bottle_retail'],
                'time_granularity': {'unit': 'DAY', 'quantity': 1},
                'predefined_splits_column': 'ml_use',
                'weight_column': 'weight',
            }
        }

    def _attribute_table_specs(self):
        return {
            'bigquery_uri': 'bq://my_project.my_dataset.product',
            'table_type': 'FORECASTING_ATTRIBUTE',
            'forecasting_attribute_table_metadata': {
                'primary_key_column': 'product_id'
            }
        }

    def _preprocessed_table_metadata(self):
        return {
            'column_metadata': {
                'product_id': {
                    'type': 'STRING',
                    'tag': 'primary_table'
                },
                'gross_quantity': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'location_id': {
                    'type': 'STRING',
                    'tag': 'primary_table'
                },
                'datetime': {
                    'type': 'DATETIME',
                    'tag': 'primary_table'
                },
                'sale_dollars': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'state_bottle_cost': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'state_bottle_retail': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'ml_use': {
                    'type': 'STRING',
                    'tag': 'primary_table'
                },
                'stock': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'weight': {
                    'type': 'FLOAT',
                    'tag': 'primary_table'
                },
                'price': {
                    'type': 'FLOAT',
                    'tag': 'attribute_table'
                },
                'vertex__timeseries__id': {
                    'type': 'STRING',
                    'tag': 'vertex_time_series_key'
                },
                'vertex__attribute': {
                    'type': 'STRING',
                    'tag': 'fe_static'
                },
                'vertex__holiday': {
                    'type': 'STRING',
                    'tag': 'fe_available_past_future'
                },
                'vertex__weather': {
                    'type': 'STRING',
                    'tag': 'fe_available_past_only'
                },
            },
            'processed_bigquery_table_uri':
                'bq://my_project.my_dataset.preprocess_2021_01_01T20_00_00_000Z',
        }

    def test_prepare_data_for_train(self):
        input_table_specs = [self._primary_table_specs(), self._attribute_table_specs()]

        expected_outputs = (
            'vertex__timeseries__id',
            '["product_id", "location_id", "price", "vertex__attribute"]',
            '["datetime", "stock", "vertex__holiday"]',
            '["gross_quantity", "sale_dollars", "state_bottle_cost", "state_bottle_retail", "vertex__weather"]',
            '[{"categorical": {"column_name": "product_id"}}, {"numeric": {"column_name": "gross_quantity"}}, {"categorical": {"column_name": "location_id"}}, {"timestamp": {"column_name": "datetime"}}, {"numeric": {"column_name": "sale_dollars"}}, {"numeric": {"column_name": "state_bottle_cost"}}, {"numeric": {"column_name": "state_bottle_retail"}}, {"numeric": {"column_name": "stock"}}, {"numeric": {"column_name": "price"}}, {"categorical": {"column_name": "vertex__attribute"}}, {"categorical": {"column_name": "vertex__holiday"}}, {"categorical": {"column_name": "vertex__weather"}}]',
            'bq://my_project.my_dataset.preprocess_2021_01_01T20_00_00_000Z',
            'gross_quantity',
            'datetime',
            'ml_use',
            'weight',
            'day',
            '1',
        )

        outputs = prepare_data_for_train(
            input_tables=json.dumps(input_table_specs),
            preprocess_metadata=json.dumps(self._preprocessed_table_metadata()))
        self.assertEqual(outputs, expected_outputs)

    def test_prepare_data_for_train_selected_model_feature_columns(self):
        input_table_specs = [self._primary_table_specs(), self._attribute_table_specs()]
        model_feature_columns = ['datetime', 'gross_quantity', 'product_id', 'sale_dollars']

        expected_outputs = (
            'vertex__timeseries__id',
            '["product_id", "location_id", "price", "vertex__attribute"]',
            '["datetime", "stock", "vertex__holiday"]',
            '["gross_quantity", "sale_dollars", "state_bottle_cost", "state_bottle_retail", "vertex__weather"]',
            '[{"categorical": {"column_name": "product_id"}}, {"numeric": {"column_name": "gross_quantity"}}, {"timestamp": {"column_name": "datetime"}}, {"numeric": {"column_name": "sale_dollars"}}]',
            'bq://my_project.my_dataset.preprocess_2021_01_01T20_00_00_000Z',
            'gross_quantity', 'datetime',
            'ml_use',
            'weight',
            'day',
            '1',
        )
        outputs = prepare_data_for_train(
            input_tables=json.dumps(input_table_specs),
            preprocess_metadata=json.dumps(self._preprocessed_table_metadata()),
            model_feature_columns=json.dumps(model_feature_columns))
        self.assertEqual(outputs, expected_outputs)
