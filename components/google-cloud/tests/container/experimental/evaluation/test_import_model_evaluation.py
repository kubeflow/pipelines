from unittest import mock
from google3.testing.pybase.googletest import TestCase

from google.cloud import aiplatform
from google_cloud_pipeline_components.container.experimental.evaluation.import_model_evaluation import main, to_value

from os import path
import json
import google.auth

SCHEMA_URI = 'gs://google-cloud-aiplatform/schema/modelevaluation/classification_metrics_1.0.0.yaml'
MODEL_NAME = 'projects/project/locations/fake-location/models/1234'
METRICS = ('{"slicedMetrics": [{"singleOutputSlicingSpec": {},"metrics": '
           '{"regression": {"rootMeanSquaredError": '
           '49.40016,"meanAbsoluteError": '
           '49.11752,"meanAbsolutePercentageError": 28428240000.0,"rSquared": '
           '0.003327712,"rootMeanSquaredLogError": 3.6381562}}}]}')
EXPLANATION = ('{"explanation": {"attributions": [{"featureAttributions": '
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


class TestImport(TestCase):

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient,
      'import_model_evaluation',
      autospec=True)
  def test_import(self, mock_api, mock_auth):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'token'
    mock_auth.return_value = [mock_creds, 'project']

    metrics_path = self.create_tempfile().full_path
    with open(metrics_path, 'w') as f:
      f.write(METRICS)

    main([
        '--metrics', metrics_path, '--problem_type', 'classification',
        '--model_name', MODEL_NAME
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_NAME,
        model_evaluation={
            'metrics':
                to_value(
                    json.loads(METRICS)['slicedMetrics'][0]['metrics']
                    ['regression']),
            'metrics_schema_uri':
                SCHEMA_URI,
        })

    explanation_path = self.create_tempfile().full_path
    with open(explanation_path, 'w') as f:
      f.write(EXPLANATION)

    main([
        '--metrics', metrics_path, '--problem_type', 'classification',
        '--model_name', MODEL_NAME, '--explanation', explanation_path
    ])
    mock_api.assert_called_with(
        mock.ANY,
        parent=MODEL_NAME,
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
                            json.loads(EXPLANATION)['explanation']
                            ['attributions'][0]['featureAttributions'])
                }]
            },
        })
