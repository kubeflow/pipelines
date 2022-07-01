"""Tests for importing model evaluations."""

import json
import os
from unittest import mock

import google.auth
from google.cloud import aiplatform
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import main
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import to_value
from google_cloud_pipeline_components.proto import gcp_resources_pb2

from google.protobuf import json_format
import unittest

SCHEMA_URI = 'gs://google-cloud-aiplatform/schema/modelevaluation/classification_metrics_1.0.0.yaml'
DISPLAY_NAME = 'sheesh'
PIPELINE_JOB_ID = 'thisisanid'
METRICS = ('{"slicedMetrics": [{"singleOutputSlicingSpec": {},"metrics": '
           '{"regression": {"rootMeanSquaredError": '
           '49.40016,"meanAbsoluteError": '
           '49.11752,"meanAbsolutePercentageError": 28428240000.0,"rSquared": '
           '0.003327712,"rootMeanSquaredLogError": 3.6381562}}}]}')
EXPLANATION_1 = (
    '{"explanation": {"attributions": [{"featureAttributions": '
    '{"BMI": 0.11054060991488765, "BPMeds": 0.0005584407939958813, '
    '"TenYearCHD": 0.0043604360566092525, "age": '
    '0.04241218286542097, "cigsPerDay": 0.03915845070606673, '
    '"currentSmoker": 0.013928816831374438, "diaBP": '
    '0.08652020580541842, "diabetes": 0.0003118844178436772, '
    '"education": 0.048558606478108966, "glucose": '
    '0.01140927870254686, "heartRate": 0.07151496486736889, '
    '"prevalentHyp": 0.0041231606832198425, "prevalentStroke": '
    '1.0034614999319733e-09, "sysBP": 0.06975447340775223, '
    '"totChol": 0.039095268419742674}}]}}')
EXPLANATION_2 = ('{"explanation": {"attributions": [{"featureAttributions": '
                 '{"BMI": 0.11111111111111111, "BPMeds": 0.222222222222222222, '
                 '"TenYearCHD": 0.0043604360566092525, "age": '
                 '0.04241218286542097, "cigsPerDay": 0.03915845070606673, '
                 '"currentSmoker": 0.013928816831374438, "diaBP": '
                 '0.08652020580541842, "diabetes": 0.0003118844178436772, '
                 '"education": 0.048558606478108966, "glucose": '
                 '0.01140927870254686, "heartRate": 0.07151496486736889, '
                 '"prevalentHyp": 0.0041231606832198425, "prevalentStroke": '
                 '1.0034614999319733e-09, "sysBP": 0.06975447340775223, '
                 '"totChol": 0.039095268419742674}}]}}')


def mock_api_call(test_func):

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient,
      'import_model_evaluation',
      autospec=True)
  def mocked_test(self, mock_api, mock_auth):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'token'
    mock_auth.return_value = [mock_creds, 'project']
    test_func(self, mock_api)

  return mocked_test


class ImportModelEvaluationTest(unittest.TestCase):

  def setUp(self):
    super(ImportModelEvaluationTest, self).setUp()
    metrics_path = self.create_tempfile().full_path
    with open(metrics_path, 'w') as f:
      f.write(METRICS)
    self.metrics_path = metrics_path
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._project = 'test_project'
    self._location = 'test_location'
    self._model_name = f'projects/{self._project}/locations/{self._location}/models/1234'
    self._model_evaluation_uri_prefix = f'https://{self._location}-aiplatform.googleapis.com/v1/'

  def tearDown(self):
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock_api_call
  def test_import_model_evaluation(self, mock_api):
    main([
        '--metrics', self.metrics_path, '--problem_type', 'classification',
        '--model_name', self._model_name, '--gcp_resources',
        self._gcp_resources, '--display_name', DISPLAY_NAME
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'display_name':
                DISPLAY_NAME,
        })

  @mock_api_call
  def test_import_model_evaluation_with_metrics_explanation(self, mock_api):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_1)

    main([
        '--metrics', self.metrics_path, '--metrics_explanation',
        explanation_path, '--problem_type', 'classification', '--model_name',
        self._model_name, '--gcp_resources', self._gcp_resources
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'model_explanation': {
                'mean_attributions': [{
                    'feature_attributions':
                        to_value(
                            json.loads(EXPLANATION_1)['explanation']
                            ['attributions'][0]['featureAttributions'])
                }]
            },
        })

  @mock_api_call
  def test_import_model_evaluation_with_explanation(self, mock_api):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics', self.metrics_path, '--explanation', explanation_path,
        '--problem_type', 'classification', '--model_name', self._model_name,
        '--gcp_resources', self._gcp_resources
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'model_explanation': {
                'mean_attributions': [{
                    'feature_attributions':
                        to_value(
                            json.loads(EXPLANATION_2)['explanation']
                            ['attributions'][0]['featureAttributions'])
                }]
            },
        })

  @mock_api_call
  def test_import_model_evaluation_with_explanation_overriding(self, mock_api):
    explanation_path_1 = self.create_tempfile().full_path
    with open(explanation_path_1, 'w') as f:
      f.write(EXPLANATION_1)

    explanation_path_2 = self.create_tempfile().full_path
    with open(explanation_path_2, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics', self.metrics_path, '--metrics_explanation',
        explanation_path_1, '--explanation', explanation_path_2,
        '--problem_type', 'classification', '--model_name', self._model_name,
        '--gcp_resources', self._gcp_resources
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'model_explanation': {
                'mean_attributions': [{
                    'feature_attributions':
                        to_value(
                            json.loads(EXPLANATION_2)['explanation']
                            ['attributions'][0]['featureAttributions'])
                }]
            },
        })

  @mock_api_call
  def test_import_model_evaluation_gcp_resources(self, mock_api):
    import_model_evaluation_response = mock.Mock()
    mock_api.return_value = import_model_evaluation_response
    import_model_evaluation_response.name = self._model_name

    main([
        '--metrics', self.metrics_path, '--problem_type', 'classification',
        '--model_name', self._model_name, '--gcp_resources', self._gcp_resources
    ])

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()

      # Instantiate GCPResources Proto
      model_evaluation_resources = json_format.Parse(
          serialized_gcp_resources, gcp_resources_pb2.GcpResources())

      self.assertLen(model_evaluation_resources.resources, 1)
      model_evaluation_name = model_evaluation_resources.resources[
          0].resource_uri[len(self._model_evaluation_uri_prefix):]
      self.assertEqual(model_evaluation_name, self._model_name)

  @mock_api_call
  def test_import_model_evaluation_empty_explanation(self, mock_api):
    import_model_evaluation_response = mock.Mock()
    mock_api.return_value = import_model_evaluation_response
    import_model_evaluation_response.name = self._model_name

    main([
        '--metrics', self.metrics_path, '--problem_type', 'classification',
        '--model_name', self._model_name, '--gcp_resources',
        self._gcp_resources, '--metrics_explanation',
        "{{$.inputs.artifacts['metrics'].metadata['explanation_gcs_path']}}"
    ])

    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
        })

  @mock_api_call
  def test_import_model_evaluation_empty_explanation_with_empty_explanation_override(
      self, mock_api):
    with self.assertRaises(SystemExit):
      main([
          '--metrics',
          self.metrics_path,
          '--problem_type',
          'classification',
          '--model_name',
          self._model_name,
          '--gcp_resources',
          self._gcp_resources,
          '--metrics_explanation',
          "{{$.inputs.artifacts['metrics'].metadata['explanation_gcs_path']}}",
          '--explanation',
          "{{$.inputs.artifacts['explanation'].metadata['explanation_gcs_path']}}",
      ])

  @mock_api_call
  def test_import_model_evaluation_contains_explanation_with_empty_explanation_override(
      self, mock_api):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_1)

    with self.assertRaises(SystemExit):
      main([
          '--metrics',
          self.metrics_path,
          '--problem_type',
          'classification',
          '--model_name',
          self._model_name,
          '--gcp_resources',
          self._gcp_resources,
          '--metrics_explanation',
          explanation_path,
          '--explanation',
          "{{$.inputs.artifacts['explanation'].metadata['explanation_gcs_path']}}",
      ])

  @mock_api_call
  def test_import_model_evaluation_contains_explanation_with_explanation_override(
      self, mock_api):
    # This explanation file will get overridden.
    explanation_path_ignored = self.create_tempfile().full_path
    with open(explanation_path_ignored, 'w') as f:
      f.write(EXPLANATION_1)

    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics', self.metrics_path, '--metrics_explanation',
        explanation_path_ignored, '--explanation', explanation_path,
        '--problem_type', 'classification', '--model_name', self._model_name,
        '--gcp_resources', self._gcp_resources
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'model_explanation': {
                'mean_attributions': [{
                    'feature_attributions':
                        to_value(
                            json.loads(EXPLANATION_2)['explanation']
                            ['attributions'][0]['featureAttributions'])
                }]
            },
        })

  @mock_api_call
  def test_import_model_evaluation_with_pipeline_id(self, mock_api):
    main([
        '--metrics', self.metrics_path, '--problem_type', 'classification',
        '--model_name', self._model_name, '--pipeline_job_id', PIPELINE_JOB_ID,
        '--gcp_resources', self._gcp_resources
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
            'metadata':
                to_value({'pipeline_job_id': PIPELINE_JOB_ID})
        })
