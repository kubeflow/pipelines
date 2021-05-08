import unittest

import mock
from kfp_component.google.dataflow import launch_flex_template

MODULE = 'kfp_component.google.dataflow._launch_flex_template'


@mock.patch(MODULE + '.storage')
@mock.patch('kfp_component.google.dataflow._common_ops.display')
@mock.patch(MODULE + '.KfpExecutionContext')
@mock.patch(MODULE + '.DataflowClient')
class LaunchFlexTemplateTest(unittest.TestCase):

    def test_launch_flex_template_succeed(
        self, mock_client, mock_context, mock_display, mock_storage
    ):
        mock_context().__enter__().context_id.return_value = 'context-1'
        mock_storage.Client().bucket().blob().exists.return_value = False
        mock_client().launch_flex_template.return_value = {
            'job': {
                'id': 'job-1'
            }
        }
        expected_job = {'id': 'job-1', 'currentState': 'JOB_STATE_DONE'}
        mock_client().get_job.return_value = expected_job

        result = launch_flex_template(
            project_id='project-1',
            location='us-central1',
            launch_parameters={
                "parameters": {
                    "foo": "bar"
                },
                "environment": {
                    "numWorkers": 1,
                    "zone": "us-central1"
                },
                "containerSpecGcsPath": "gs://foo/bar",
            },
            staging_dir='gs://staging/dir'
        )
        self.assertEqual(expected_job, result)
        mock_client().launch_flex_template.assert_called_once()
        mock_storage.Client().bucket().blob(
        ).upload_from_string.assert_called_with('job-1,us-central1')

    def test_launch_flex_template_retry_succeed(
        self, mock_client, mock_context, mock_display, mock_storage
    ):
        mock_context().__enter__().context_id.return_value = 'context-1'
        mock_storage.Client().bucket().blob().exists.return_value = True
        mock_storage.Client().bucket().blob(
        ).download_as_string.return_value = b'job-1,'

        pending_job = {'currentState': 'JOB_STATE_PENDING'}
        expected_job = {'id': 'job-1', 'currentState': 'JOB_STATE_DONE'}

        mock_client().get_job.side_effect = [pending_job, expected_job]

        result = launch_flex_template(
            project_id='project-1',
            location='us-central1',
            launch_parameters={
                "parameters": {
                    "foo": "bar"
                },
                "environment": {
                    "numWorkers": 1,
                    "zone": "us-central1"
                },
                "containerSpecGcsPath": "gs://foo/bar",
            },
            staging_dir='gs://staging/dir'
        )
        self.assertEqual(expected_job, result)
        mock_client().launch_flex_template.assert_not_called()

    def test_launch_flex_template_fail(
        self, mock_client, mock_context, mock_display, mock_storage
    ):
        mock_context().__enter__().context_id.return_value = 'context-1'
        mock_storage.Client().bucket().blob().exists.return_value = False
        mock_client().launch_flex_template.return_value = {
            'job': {
                'id': 'job-1'
            }
        }
        failed_job = {'id': 'job-1', 'currentState': 'JOB_STATE_FAILED'}
        mock_client().get_job.return_value = failed_job

        with self.assertRaises(RuntimeError):
            launch_flex_template(
                project_id='project-1',
                location='us-central1',
                launch_parameters={
                    "parameters": {
                        "foo": "bar"
                    },
                    "environment": {
                        "numWorkers": 1,
                        "zone": "us-central1"
                    },
                    "containerSpecGcsPath": "gs://foo/bar",
                },
                staging_dir='gs://staging/dir'
            )
