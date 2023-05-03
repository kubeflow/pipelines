"""Tests for importing model evaluations."""

import json
import os

import google.auth
from google.cloud import aiplatform
from google.cloud.aiplatform_v1.types.model_evaluation_slice import ModelEvaluationSlice
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import main
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import PROBLEM_TYPE_TO_SCHEMA_URI
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import to_value
from google_cloud_pipeline_components.proto import gcp_resources_pb2

from google.protobuf import json_format
import unittest
from unittest import mock


SCHEMA_URI = 'gs://google-cloud-aiplatform/schema/modelevaluation/classification_metrics_1.0.0.yaml'
DISPLAY_NAME = 'sheesh'
PIPELINE_JOB_ID = 'thisisanid'
METRICS = (
    '{"slicedMetrics": [{"singleOutputSlicingSpec": {},"metrics": '
    '{"regression": {"rootMeanSquaredError": 49.40016,"meanAbsoluteError": '
    '49.11752,"meanAbsolutePercentageError": 28428240000.0,"rSquared": '
    '0.003327712,"rootMeanSquaredLogError": 3.6381562}}}, '
    '{"singleOutputSlicingSpec": {"bytesValue": "MA=="}, "metrics": '
    '{"regression": {"rootMeanSquaredError": 123}}}]}'
)
METRICS_2 = (
    '{"slicedMetrics": [{"singleOutputSlicingSpec": {},"metrics":'
    ' {"classification": {"rootMeanSquaredError": 49.40016}}},'
    ' {"singleOutputSlicingSpec": {"int64Value": "1"}, "metrics":'
    ' {"classification": {"rootMeanSquaredError": 5.87}}},'
    ' {"singleOutputSlicingSpec": {"int64Value": "0"},'
    ' "metrics":{"classification": {"rootMeanSquaredError": 13.226}}}]}'
)
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
    '"totChol": 0.039095268419742674}}]}}'
)
EXPLANATION_2 = (
    '{"explanation": {"attributions": [{"featureAttributions": '
    '{"BMI": 0.11111111111111111, "BPMeds": 0.222222222222222222, '
    '"TenYearCHD": 0.0043604360566092525, "age": '
    '0.04241218286542097, "cigsPerDay": 0.03915845070606673, '
    '"currentSmoker": 0.013928816831374438, "diaBP": '
    '0.08652020580541842, "diabetes": 0.0003118844178436772, '
    '"education": 0.048558606478108966, "glucose": '
    '0.01140927870254686, "heartRate": 0.07151496486736889, '
    '"prevalentHyp": 0.0041231606832198425, "prevalentStroke": '
    '1.0034614999319733e-09, "sysBP": 0.06975447340775223, '
    '"totChol": 0.039095268419742674}}]}}'
)
EXPLANATION_3 = (
    '{"explanation": {"attributions": [{"featureAttributions": {"BMI":'
    ' 0.02266513112755049, "BPMeds": 1.419158208937872e-09, "TenYearCHD":'
    ' 0.013602024742535182}}, {"dimension": "annotationSpec",'
    ' "value": "1", "featureAttributions": {"BMI": 0.024043104300896324,'
    ' "BPMeds": 9.934107462565104e-10, "TenYearCHD": 0.019042834639549255'
    ' }}, {"dimension": "annotationSpec", "value": "0",'
    ' "featureAttributions": {"BMI": 0.01922019819418589, "BPMeds":'
    ' 2.483526865641276e-09, "TenYearCHD": 0.0}}]}}'
)

PROJECT = 'test_project'
LOCATION = 'test_location'
MODEL_EVAL_NAME = (
    f'projects/{PROJECT}/locations/{LOCATION}/models/1234/evaluations/567'
)


def mock_api_call(test_func):
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient,
      'import_model_evaluation',
      autospec=True,
  )
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient,
      'batch_import_model_evaluation_slices',
      autospec=True,
  )
  def mocked_test(self, mock_import_slice, mock_import_eval, mock_auth):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'token'
    mock_auth.return_value = [mock_creds, 'project']
    import_model_evaluation_response = mock.Mock()
    import_model_evaluation_response.name = MODEL_EVAL_NAME
    mock_import_eval.return_value = import_model_evaluation_response
    test_func(self, mock_import_eval, mock_import_slice)

  return mocked_test


class ImportModelEvaluationTest(unittest.TestCase):

  def setUp(self):
    super(ImportModelEvaluationTest, self).setUp()
    metrics_path = self.create_tempfile().full_path
    metrics_path_2 = self.create_tempfile().full_path
    with open(metrics_path, 'w') as f:
      f.write(METRICS)
    with open(metrics_path_2, 'w') as f:
      f.write(METRICS_2)

    self.metrics_path = metrics_path
    self.metrics_path_2 = metrics_path_2
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources'
    )
    self._project = PROJECT
    self._location = LOCATION
    self._model_name = (
        f'projects/{self._project}/locations/{self._location}/models/1234'
    )
    self._model_evaluation_uri_prefix = (
        f'https://{self._location}-aiplatform.googleapis.com/v1/'
    )

    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock_api_call
  def test_import_model_evaluation(self, mock_api, _):
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
        '--display_name',
        DISPLAY_NAME,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'display_name': DISPLAY_NAME,
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_metrics_explanation(self, mock_api, _):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_1)

    main([
        '--metrics',
        self.metrics_path,
        '--metrics_explanation',
        explanation_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'model_explanation': {
                'mean_attributions': [
                    {
                        'feature_attributions': to_value(
                            json.loads(EXPLANATION_1)['explanation'][
                                'attributions'
                            ][0]['featureAttributions']
                        )
                    }
                ]
            },
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_explanation(self, mock_api, _):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics',
        self.metrics_path,
        '--explanation',
        explanation_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'model_explanation': {
                'mean_attributions': [
                    {
                        'feature_attributions': to_value(
                            json.loads(EXPLANATION_2)['explanation'][
                                'attributions'
                            ][0]['featureAttributions']
                        )
                    }
                ]
            },
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_sliced_explanation(self, _, mock_api):
    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_3)

    main([
        '--metrics',
        self.metrics_path_2,
        '--explanation',
        explanation_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[
            {
                'metrics_schema_uri': SCHEMA_URI,
                'metrics': to_value(
                    json.loads(METRICS_2)['slicedMetrics'][1]['metrics'][
                        'classification'
                    ]
                ),
                'slice_': {
                    'dimension': 'annotationSpec',
                    'value': '1',
                },
                'model_explanation': {
                    'mean_attributions': [
                        {
                            'feature_attributions': to_value(
                                json.loads(EXPLANATION_3)['explanation'][
                                    'attributions'
                                ][1]['featureAttributions']
                            )
                        }
                    ]
                },
            },
            {
                'metrics_schema_uri': SCHEMA_URI,
                'metrics': to_value(
                    json.loads(METRICS_2)['slicedMetrics'][2]['metrics'][
                        'classification'
                    ]
                ),
                'slice_': {
                    'dimension': 'annotationSpec',
                    'value': '0',
                },
                'model_explanation': {
                    'mean_attributions': [
                        {
                            'feature_attributions': to_value(
                                json.loads(EXPLANATION_3)['explanation'][
                                    'attributions'
                                ][2]['featureAttributions']
                            )
                        }
                    ]
                },
            },
        ],
    )

  @mock_api_call
  def test_import_model_evaluation_with_explanation_overriding(
      self, mock_api, _
  ):
    explanation_path_1 = self.create_tempfile().full_path
    with open(explanation_path_1, 'w') as f:
      f.write(EXPLANATION_1)

    explanation_path_2 = self.create_tempfile().full_path
    with open(explanation_path_2, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics',
        self.metrics_path,
        '--metrics_explanation',
        explanation_path_1,
        '--explanation',
        explanation_path_2,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'model_explanation': {
                'mean_attributions': [
                    {
                        'feature_attributions': to_value(
                            json.loads(EXPLANATION_2)['explanation'][
                                'attributions'
                            ][0]['featureAttributions']
                        )
                    }
                ]
            },
        },
    )

  @mock_api_call
  def test_import_model_evaluation_gcp_resources(self, mock_api, _):
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()

      # Instantiate GCPResources Proto
      model_evaluation_resources = json_format.Parse(
          serialized_gcp_resources, gcp_resources_pb2.GcpResources()
      )

      self.assertLen(model_evaluation_resources.resources, 2)
      model_evaluation_name = model_evaluation_resources.resources[
          0
      ].resource_uri[len(self._model_evaluation_uri_prefix) :]
      self.assertEqual(model_evaluation_name, MODEL_EVAL_NAME)

  @mock_api_call
  def test_import_model_evaluation_empty_explanation(self, mock_api, _):
    import_model_evaluation_response = mock.Mock()
    mock_api.return_value = import_model_evaluation_response
    import_model_evaluation_response.name = self._model_name

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
    ])

    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
        },
    )

  @mock_api_call
  def test_import_model_evaluation_empty_explanation_with_empty_explanation_override(
      self, mock_api, _
  ):
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
      self, mock_api, _
  ):
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
      self, mock_api, _
  ):
    # This explanation file will get overridden.
    explanation_path_ignored = self.create_tempfile().full_path
    with open(explanation_path_ignored, 'w') as f:
      f.write(EXPLANATION_1)

    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION_2)

    main([
        '--metrics',
        self.metrics_path,
        '--metrics_explanation',
        explanation_path_ignored,
        '--explanation',
        explanation_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'model_explanation': {
                'mean_attributions': [
                    {
                        'feature_attributions': to_value(
                            json.loads(EXPLANATION_2)['explanation'][
                                'attributions'
                            ][0]['featureAttributions']
                        )
                    }
                ]
            },
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_pipeline_id(self, mock_api, _):
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'metadata': to_value({'pipeline_job_id': PIPELINE_JOB_ID}),
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_pipeline_resource_name(
      self, mock_api, _
  ):
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_resource_name',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'metadata': to_value(
                {'pipeline_job_resource_name': PIPELINE_JOB_ID}
            ),
        },
    )

  @mock_api_call
  def test_import_model_evaluation_slice(self, _, mock_api):
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[{
            'metrics_schema_uri': SCHEMA_URI,
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][1]['metrics']['regression']
            ),
            'slice_': {
                'dimension': 'annotationSpec',
                'value': '0',
            },
        }],
    )

  @mock_api_call
  def test_import_model_evaluation_with_classification_metrics_artifact(
      self, mock_api, _
  ):
    main([
        '--classification_metrics',
        self.metrics_path,
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
        '--display_name',
        DISPLAY_NAME,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['classification'],
            'display_name': DISPLAY_NAME,
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_forecasting_metrics_artifact(
      self, mock_api, _
  ):
    main([
        '--forecasting_metrics',
        self.metrics_path,
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
        '--display_name',
        DISPLAY_NAME,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['forecasting'],
            'display_name': DISPLAY_NAME,
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_regression_metrics_artifact(
      self, mock_api, _
  ):
    main([
        '--regression_metrics',
        self.metrics_path,
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
        '--display_name',
        DISPLAY_NAME,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['regression'],
            'display_name': DISPLAY_NAME,
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_dataset_path(self, mock_api, _):
    main([
        '--regression_metrics',
        self.metrics_path,
        '--dataset_path',
        'PATH',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['regression'],
            'metadata': to_value({'evaluation_dataset_path': ['PATH']}),
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_dataset_paths(self, mock_api, _):
    PATHS = ['path1', 'path2']
    main([
        '--regression_metrics',
        self.metrics_path,
        '--dataset_paths',
        json.dumps(PATHS),
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['regression'],
            'metadata': to_value({'evaluation_dataset_path': PATHS}),
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_invalid_dataset_paths(
      self, mock_api, _
  ):
    main([
        '--regression_metrics',
        self.metrics_path,
        '--dataset_paths',
        '{notvalidjson}',
        '--model_name',
        self._model_name,
        '--gcp_resources',
        self._gcp_resources,
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=self._model_name,
        model_evaluation={
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][0]['metrics']['regression']
            ),
            'metrics_schema_uri': PROBLEM_TYPE_TO_SCHEMA_URI['regression'],
        },
    )

  @mock_api_call
  def test_import_model_evaluation_with_over_50_slices(self, _, mock_api):
    # Arrange.
    sliced_metrics = json.loads(METRICS)['slicedMetrics']
    for i in range(2, 52):
      sliced_metrics.append({
          'singleOutputSlicingSpec': {'int64Value': i},
          'metrics': {'regression': {'rootMeanSquaredError': i}},
      })
    metrics = {'slicedMetrics': sliced_metrics}
    with open(self.metrics_path, 'w') as f:
      f.write(json.dumps(metrics))

    model_evaluation_slices_50 = [{
        'metrics_schema_uri': SCHEMA_URI,
        'metrics': to_value(
            metrics['slicedMetrics'][1]['metrics']['regression']
        ),
        'slice_': {
            'dimension': 'annotationSpec',
            'value': '0',
        },
    }]
    for i in range(2, 51):
      model_evaluation_slices_50.append({
          'metrics_schema_uri': SCHEMA_URI,
          'metrics': to_value(
              metrics['slicedMetrics'][i]['metrics']['regression']
          ),
          'slice_': {
              'dimension': 'annotationSpec',
              'value': str(i),
          },
      })

    # Act.
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])

    # Assert.
    # Assert the first call with the batch of 50 slices.
    mock_api.assert_any_call(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=model_evaluation_slices_50,
    )
    # Assert the second call with the batch of 1 slice.
    mock_api.assert_any_call(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[{
            'metrics_schema_uri': SCHEMA_URI,
            'metrics': to_value(
                metrics['slicedMetrics'][51]['metrics']['regression']
            ),
            'slice_': {
                'dimension': 'annotationSpec',
                'value': '51',
            },
        }],
    )
    self.assertEqual(mock_api.call_count, 2)

  @mock_api_call
  def test_import_model_evaluation_slice_with_slice_dimension_no_slicing_spec(
      self, _, mock_api
  ):
    # Arrange.
    sliced_metrics = json.loads(METRICS)['slicedMetrics']
    sliced_metrics[1]['singleOutputSlicingSpec']['dimension'] = 'slice'
    sliced_metrics[1]['singleOutputSlicingSpec']['value'] = 'test_value'
    sliced_metrics[1]['singleOutputSlicingSpec']['slicingSpec'] = {
        'configs': {}
    }
    metrics = {'slicedMetrics': sliced_metrics}
    with open(self.metrics_path, 'w') as f:
      f.write(json.dumps(metrics))

    # Act.
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])

    # Assert.
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[{
            'metrics_schema_uri': SCHEMA_URI,
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][1]['metrics']['regression']
            ),
            'slice_': {
                'dimension': 'slice',
                'value': 'test_value',
            },
        }],
    )

  @mock_api_call
  def test_import_model_evaluation_slice_with_slice_dimension(
      self, _, mock_api
  ):
    # Arrange.
    sliced_metrics = json.loads(METRICS)['slicedMetrics']
    sliced_metrics[1]['singleOutputSlicingSpec']['dimension'] = 'slice'
    sliced_metrics[1]['singleOutputSlicingSpec']['value'] = 'test_value'
    sliced_metrics[1]['singleOutputSlicingSpec']['slicingSpec'] = {
        'configs': {'feature_a': {'value': {'stringValue': 'label_a'}}}
    }
    metrics = {'slicedMetrics': sliced_metrics}
    with open(self.metrics_path, 'w') as f:
      f.write(json.dumps(metrics))

    # Act.
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])

    # Assert.
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[{
            'metrics_schema_uri': SCHEMA_URI,
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][1]['metrics']['regression']
            ),
            'slice_': {
                'dimension': 'slice',
                'value': 'test_value',
                'slice_spec': ModelEvaluationSlice.Slice.SliceSpec.from_json(
                    json.dumps(
                        {
                            'configs': {
                                'feature_a': {
                                    'value': {'stringValue': 'label_a'}
                                }
                            }
                        }
                    )
                ),
            },
        }],
    )

  @mock_api_call
  def test_import_model_evaluation_slice_with_annotation_spec_dimension(
      self, _, mock_api
  ):
    # Arrange.
    sliced_metrics = json.loads(METRICS)['slicedMetrics']
    sliced_metrics[1]['singleOutputSlicingSpec']['dimension'] = 'annotationSpec'
    sliced_metrics[1]['singleOutputSlicingSpec']['value'] = 'test_value'
    metrics = {'slicedMetrics': sliced_metrics}
    with open(self.metrics_path, 'w') as f:
      f.write(json.dumps(metrics))

    # Act.
    main([
        '--metrics',
        self.metrics_path,
        '--problem_type',
        'classification',
        '--model_name',
        self._model_name,
        '--pipeline_job_id',
        PIPELINE_JOB_ID,
        '--gcp_resources',
        self._gcp_resources,
    ])

    # Assert.
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_EVAL_NAME,
        model_evaluation_slices=[{
            'metrics_schema_uri': SCHEMA_URI,
            'metrics': to_value(
                json.loads(METRICS)['slicedMetrics'][1]['metrics']['regression']
            ),
            'slice_': {
                'dimension': 'annotationSpec',
                'value': 'test_value',
            },
        }],
    )


if __name__ == '__main__':
  unittest.main()
