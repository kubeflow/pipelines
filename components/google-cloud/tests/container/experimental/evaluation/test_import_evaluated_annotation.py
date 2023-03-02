from collections import defaultdict
import json

from google.cloud import storage
from google.cloud import aiplatform_v1
from google_cloud_pipeline_components.container.experimental.evaluation import import_evaluated_annotation

import unittest
from unittest import mock


PROJECT = 'test_project'
LOCATION = 'test_location'
MODEL_NAME = f'projects/{PROJECT}/locations/{LOCATION}/models/1234'
MODEL_EVAL_NAME = (
    f'projects/{PROJECT}/locations/{LOCATION}/models/1234/evaluations/567'
)

EVAL_SLICE_NAME_1 = 'projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908/slices/7424503431852652227'
EVAL_SLICE_NAME_2 = 'projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908/slices/7599200443631607093'


ERROR_ANALYSIS_ANNOTATION_1 = {
    'attributedItems': [
        {
            'annotationResourceName': 'projects/123456/locations/us-central1/datasets/114514/dataItems/12345/annotations/57981023935',
            'distance': 1.4385216236114502,
        },
    ],
    'outlierScore': 1.4385216236114502,
    'outlierThreshold': 2.2392595775946518,
}

ERROR_ANALYSIS_ANNOTATION_2 = {
    'attributedItems': [
        {
            'annotationResourceName': 'projects/123456/locations/us-central1/datasets/114514/dataItems/54124/annotations/836409557033',
            'distance': 0.6817775964736938,
        },
    ],
    'outlierScore': 0.6817775964736938,
    'outlierThreshold': 2.2392595775946518,
}


class ImportEvaluatedAnnotationTest(unittest.TestCase):

  @mock.patch.object(
      import_evaluated_annotation, '_make_parent_dirs_and_return_path'
  )
  def test_parse_args(self, mock_make_parent_dirs_and_return_path):
    mock_make_parent_dirs_and_return_path.return_value = '/gcs/pipeline/dir'
    test_argv = [
        '--evaluated_annotation_output_uri',
        'gs://my-bucket/path/to/evaluated_annotation.jsonl',
        '--error_analysis_output_uri',
        'gs://my-bucket/path/to/error_analysis.jsonl',
        '--evaluation_importer_gcp_resources',
        '/gcs/pipeline/eval/gcp_resources',
        '--pipeline_job_id',
        'test',
        '--model_name',
        'model_name',
        '--gcp_resources',
        '/gcs/pipeline/dir',
    ]
    import_evaluated_annotation._parse_args(test_argv)

  @mock.patch.object(json, 'loads')
  def test_get_model_eval_resource_name(self, mock_json):
    # Arrange.
    json_content = {
        'resources': [
            {
                'resourceType': 'ModelEvaluationSlice',
                'resourceUri': 'https://us-central1-aiplatform.googleapis.com/v1/projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908/slices/7599200443631607093',
            },
            {
                'resourceType': 'ModelEvaluationSlice',
                'resourceUri': 'https://us-central1-aiplatform.googleapis.com/v1/projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908/slices/7424503431852652227',
            },
            {
                'resourceType': 'ModelEvaluation',
                'resourceUri': 'https://us-central1-aiplatform.googleapis.com/v1/projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908',
            },
        ]
    }
    mock_json.return_value = json_content
    expected_output = 'projects/977012026409/locations/us-central1/models/3699931394256928768/evaluations/3142641870479572908'
    # Act.
    result = import_evaluated_annotation.get_model_eval_resource_name(
        'https://us-central1-aiplatform.googleapis.com/v1/',
        '/gcs/test/gcp_resources',
    )
    # Assert.
    self.assertEqual(result, expected_output)

  @mock.patch.object(aiplatform_v1, 'ModelServiceClient')
  def test_get_model_evaluation_slices_annotation_spec_map(self, mock_client):
    # Arrange.
    mock_eval_slice1 = mock.MagicMock()
    mock_eval_slice1.slice_.dimension = 'annotationSpec'
    mock_eval_slice1.slice_.value = '1'
    mock_eval_slice1.name = EVAL_SLICE_NAME_1
    mock_eval_slice2 = mock.MagicMock()
    mock_eval_slice2.slice_.dimension = 'annotationSpec'
    mock_eval_slice2.slice_.value = '0'
    mock_eval_slice2.name = EVAL_SLICE_NAME_2
    mock_eval_slices_page_result = [mock_eval_slice1, mock_eval_slice2]

    mock_client.list_model_evaluation_slices.return_value = (
        mock_eval_slices_page_result
    )
    model_evaluation_resource_name = MODEL_EVAL_NAME
    expected_result = {'1': EVAL_SLICE_NAME_1, '0': EVAL_SLICE_NAME_2}
    # Act.
    result = import_evaluated_annotation.get_model_evaluation_slices_annotation_spec_map(
        mock_client, model_evaluation_resource_name
    )
    # Assert.
    self.assertEqual(result, expected_result)

  def test_read_valid_gcs_uri(self):
    # Arrange.
    bucket = mock.MagicMock(spec=storage.Bucket)
    blob = mock.MagicMock(spec=storage.Blob)
    blob.download_as_text.return_value = 'hello, world!'
    bucket.blob.return_value = blob
    client = mock.MagicMock(spec=storage.Client)
    client.bucket.return_value = bucket
    storage.Client = mock.MagicMock(return_value=client)
    # Act.
    result = import_evaluated_annotation.read_gcs_uri_as_text(
        'gs://my-bucket/my-file.txt'
    )
    # Assert.
    self.assertEqual(result, 'hello, world!')
    client.bucket.assert_called_once_with('my-bucket')
    bucket.blob.assert_called_once_with('my-file.txt')
    blob.download_as_text.assert_called_once_with()

  def test_read_invalid_gcs_uri(self):
    # Act & Assert.
    with self.assertRaises(ValueError):
      import_evaluated_annotation.read_gcs_uri_as_text('invalid-gcs-uri')

  @mock.patch.object(import_evaluated_annotation, 'read_gcs_uri_as_text')
  def test_get_error_analysis_map(self, mock_read_gcs_uri_as_text):
    # Arrange.
    output_uri = 'gs://my-bucket/path/to/error_analysis.jsonl'
    error_analysis_map = defaultdict(
        list,
        {
            'annotation_resource_name_1': [ERROR_ANALYSIS_ANNOTATION_1],
            'annotation_resource_name_2': [ERROR_ANALYSIS_ANNOTATION_2],
        },
    )
    error_analysis_file_contents = '\n'.join([
        json.dumps({
            'annotationResourceName': 'annotation_resource_name_1',
            'annotation': ERROR_ANALYSIS_ANNOTATION_1,
        }),
        json.dumps({
            'annotationResourceName': 'annotation_resource_name_2',
            'annotation': ERROR_ANALYSIS_ANNOTATION_2,
        }),
    ])
    mock_read_gcs_uri_as_text.return_value = error_analysis_file_contents
    # Act.
    result = import_evaluated_annotation.get_error_analysis_map(output_uri)
    # Assert.
    self.assertEqual(result, error_analysis_map)
    mock_read_gcs_uri_as_text.assert_called_once_with(output_uri)

  @mock.patch.object(aiplatform_v1, 'ModelServiceClient')
  @mock.patch.object(
      import_evaluated_annotation,
      'get_model_evaluation_slices_annotation_spec_map',
  )
  @mock.patch.object(
      import_evaluated_annotation, 'get_model_eval_resource_name'
  )
  @mock.patch.object(import_evaluated_annotation, '_parse_args')
  def test_main(
      self,
      mock_parse_args,
      mock_get_model_eval_resource_name,
      mock_get_model_evaluation_slices_annotation_spec_map,
      mock_client,
  ):
    # Arrange.
    parsed_args = mock.MagicMock()
    parsed_args.model_name = MODEL_NAME
    mock_parse_args.return_value = parsed_args
    mock_client.return_value = mock.MagicMock()

    model_evaluation_resource_name = MODEL_EVAL_NAME
    mock_get_model_eval_resource_name.return_value = (
        model_evaluation_resource_name
    )

    mock_get_model_evaluation_slices_annotation_spec_map.return_value = {
        '1': EVAL_SLICE_NAME_1,
        '0': EVAL_SLICE_NAME_2,
    }

    # Act.
    import_evaluated_annotation.main([
        '--evaluated_annotation_output_uri',
        'gs://my-bucket/path/to/evaluated_annotation.jsonl',
        '--error_analysis_output_uri',
        'gs://my-bucket/path/to/error_analysis.jsonl',
        '--evaluation_importer_gcp_resources',
        '/gcs/pipeline/eval/gcp_resources',
        '--pipeline_job_id',
        'test',
        '--model_name',
        'model_name',
        '--gcp_resources',
        '/gcs/pipeline/dir',
    ])

    # Assert.
    mock_parse_args.assert_called_once()
    mock_get_model_eval_resource_name.assert_called_once()
    mock_get_model_evaluation_slices_annotation_spec_map.assert_called_once()
