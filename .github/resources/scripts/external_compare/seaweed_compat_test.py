"""Guard the upgrade harness against accepting failed or missing isolation checks."""

import unittest
from unittest.mock import Mock, patch

import seaweed_compat as test
from botocore.exceptions import ClientError


def error(code, status):
    return ClientError(
        {
            'Error': {'Code': code},
            'ResponseMetadata': {'HTTPStatusCode': status},
        },
        'GetObject',
    )


class IsolationTest(unittest.TestCase):
    def test_only_actual_access_denial_is_success(self):
        test.denied(Mock(side_effect=error('AccessDenied', 403)))
        for failure in [
            error('NoSuchKey', 404),
            error('InternalError', 500),
            RuntimeError('connection failed'),
        ]:
            with (
                self.subTest(failure=failure),
                self.assertRaises(type(failure)),
            ):
                test.denied(Mock(side_effect=failure))
        with self.assertRaises(AssertionError):
            test.denied(Mock(return_value={}))

    def test_pod_receipt_rejects_wrong_image_and_unready_pods(self):
        import json
        import tempfile
        from pathlib import Path

        pod = {
            'metadata': {},
            'spec': {
                'containers': [
                    {'name': 'seaweedfs', 'image': 'expected@sha256:abc'}
                ]
            },
            'status': {
                'containerStatuses': [
                    {
                        'name': 'seaweedfs',
                        'ready': True,
                        'imageID': 'sha256:def',
                    }
                ]
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(
                test.compat,
                'kubectl',
                return_value=json.dumps({'items': [pod]}),
            ):
                self.assertEqual(
                    test.pods(Path(directory), 'ok', 'expected@sha256:abc'),
                    'sha256:def',
                )
                with self.assertRaises(AssertionError):
                    test.pods(Path(directory), 'wrong', 'other')
            pod['status']['containerStatuses'][0]['ready'] = False
            with (
                patch.object(
                    test.compat,
                    'kubectl',
                    return_value=json.dumps({'items': [pod]}),
                ),
                self.assertRaises(AssertionError),
            ):
                test.pods(Path(directory), 'unready', 'expected@sha256:abc')


if __name__ == '__main__':
    unittest.main()
