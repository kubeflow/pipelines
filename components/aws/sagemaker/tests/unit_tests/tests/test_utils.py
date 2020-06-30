import unittest
import json

from unittest.mock import patch, call, Mock, MagicMock, mock_open
from botocore.exceptions import ClientError

from common import _utils


# Test common util functions used across the components
class UtilsTestCase(unittest.TestCase):
    def test_cw_logging_successfully(self):
        mock_cw_client = MagicMock()
        mock_cw_client.describe_log_streams.return_value = {'logStreams': [{'logStreamName': 'logStream1'},
                                                                           {'logStreamName': 'logStream2'}]}

        def my_get_log_events(logStreamName, **kwargs):
            if logStreamName =='logStream1':
                return {'events': [{'message': 'fake log logStream1 line1'},
                                   {'message': 'fake log logStream1 line2'}
                                   ]}
            elif logStreamName =='logStream2':
                return {'events': [{'message': 'fake log logStream2 line1'},
                                   {'message': 'fake log logStream2 line2'}
                                   ]}
        mock_cw_client.get_log_events.side_effect = my_get_log_events

        with patch('logging.Logger.info') as infoLog:
            _utils.print_logs_for_job(mock_cw_client, '/aws/sagemaker/FakeJobs', 'fake_job_name')
            print (infoLog.call_args_list)
            calls = [call('fake log logStream1 line1'),
                     call('fake log logStream1 line2'),
                     call('fake log logStream2 line1'),
                     call('fake log logStream2 line2')
                     ]
            infoLog.assert_has_calls(calls, any_order=True)


    def test_cw_logging_error(self):
        mock_cw_client = MagicMock()
        mock_exception = ClientError({"Error": {"Message": "CloudWatch broke"}}, "describe_log_streams")
        mock_cw_client.describe_log_streams.side_effect = mock_exception

        with patch('logging.Logger.error') as errorLog:
            _utils.print_logs_for_job(mock_cw_client, '/aws/sagemaker/FakeJobs', 'fake_job_name')
            errorLog.assert_called()

    def test_write_output_string(self):
        with patch("common._utils.Path", MagicMock()) as mock_path:
            _utils.write_output("/tmp/output-path", "output-value")

        mock_path.assert_called_with("/tmp/output-path")
        mock_path("/tmp/output-path").parent.mkdir.assert_called()
        mock_path("/tmp/output-path").write_text.assert_called_with("output-value")

    def test_write_output_json(self):
        # Ensure working versions of each type of JSON input
        test_cases = [{"key1": "value1"}, ["val1", "val2"], "string-val"]

        for case in test_cases:
            with patch("common._utils.Path", MagicMock()) as mock_path:
                _utils.write_output("/tmp/test-output", case, json_encode=True)

                mock_path.assert_called_with("/tmp/test-output")
                mock_path("/tmp/test-output").parent.mkdir.assert_called()
                mock_path("/tmp/test-output").write_text.assert_called_with(
                    json.dumps(case)
                )
