"""Protocol checks for the bounded in-cluster API availability client."""

import importlib.util
import io
import json
import struct
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

CLIENT_PATH = Path(__file__).with_name('api-availability-client.py')
SPEC = importlib.util.spec_from_file_location(
    'api_availability_client', CLIENT_PATH
)
CLIENT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CLIENT)


class RecordingConnection:
    def __init__(self):
        self.path = None
        self.headers = {}
        self.parts = []

    def putrequest(self, method, path):
        self.path = (method, path)

    def putheader(self, key, value):
        self.headers[key.lower()] = value

    def endheaders(self):
        pass

    def send(self, data):
        self.parts.append(bytes(data))

    def getresponse(self):
        return SimpleNamespace(status=400, read=lambda size: b'invalid')

    def close(self):
        pass


class AvailabilityClientTest(unittest.TestCase):
    def test_multipart_request_sends_one_complete_file_part(self):
        connection = RecordingConnection()
        with mock.patch.object(
            CLIENT.http.client, 'HTTPConnection', return_value=connection
        ):
            status, response = CLIENT.multipart_request(b'dummy', 'small.yaml')
        body = b''.join(connection.parts)
        self.assertEqual(status, 400)
        self.assertEqual(response, b'invalid')
        self.assertEqual(
            connection.path,
            ('POST', '/apis/v2/pipelines/upload?namespace=kfp-audit-attacker'),
        )
        self.assertEqual(int(connection.headers['content-length']), len(body))
        self.assertIn(b'name="uploadfile"; filename="small.yaml"', body)
        self.assertIn(b'\r\n\r\ndummy\r\n', body)
        self.assertTrue(body.endswith(b'--kfp-audit-boundary--\r\n'))

    def test_zip_fixture_fits_application_limit(self):
        fixture = CLIENT.zip_directory_fixture()
        self.assertEqual(len(fixture), 33_554_400)
        self.assertLessEqual(len(fixture), 32 << 20)
        self.assertEqual(fixture[:4], b'PK\x01\x02')
        end = len(fixture) - 22
        self.assertEqual(fixture[end : end + 4], b'PK\x05\x06')
        self.assertEqual(struct.unpack_from('<I', fixture, end + 12)[0], end)

    def test_report_uses_gateway_workflow_field(self):
        with mock.patch.object(
            CLIENT, 'request_status', return_value=(400, b'error')
        ) as request:
            self.assertEqual(CLIENT.report_request(), (400, b'error'))
        path, body, content_type = request.call_args.args
        self.assertEqual(path, '/apis/v2/workflows')
        self.assertEqual(content_type, 'application/json')
        workflow = json.loads(body)
        self.assertTrue(workflow.startswith('{"kind": "Workflow"'))
        self.assertIn(
            '"metadata": {"namespace": "kfp-audit-attacker"', workflow
        )
        self.assertEqual(body.count(b'x'), 16 << 20)

    def test_attack_needs_handler_marker_or_explicit_boundary(self):
        with redirect_stdout(io.StringIO()):
            with mock.patch.object(
                CLIENT,
                'report_request',
                return_value=(400, b'The workflow must have a name'),
            ):
                self.assertEqual(CLIENT.run_attack('report'), 0)
            with mock.patch.object(
                CLIENT, 'report_request', return_value=(403, b'denied')
            ):
                self.assertEqual(CLIENT.run_attack('report'), 0)
            with mock.patch.object(
                CLIENT, 'report_request', return_value=(500, b'gateway error')
            ):
                self.assertEqual(CLIENT.run_attack('report'), 1)
            with mock.patch.object(
                CLIENT,
                'oversized_multipart_request',
                return_value=(413, b'{"error_message":"File size too large"}'),
            ):
                self.assertEqual(CLIENT.run_attack('multipart'), 0)


if __name__ == '__main__':
    unittest.main()
