"""Guard the real-Metacontroller upgrade assertions."""

import copy
import unittest

from metacontroller_compat import (
    credential_fingerprint,
    require_live_images,
    require_unchanged_secret,
    select_statefulset,
    set_metacontroller_image,
)

IMAGE = 'ghcr.io/metacontroller/metacontroller:v4.17.9@sha256:' + 'a' * 64


def secret():
    return {
        'metadata': {'uid': 'original'},
        'data': {'accesskey': 'YWNjZXNz', 'secretkey': 'c2VjcmV0'},
    }


def pods():
    return {
        'items': [
            {
                'metadata': {'name': 'metacontroller-0', 'uid': 'pod-uid'},
                'spec': {
                    'containers': [{'name': 'metacontroller', 'image': IMAGE}]
                },
                'status': {
                    'containerStatuses': [
                        {
                            'name': 'metacontroller',
                            'ready': True,
                            'imageID': 'ghcr.io/metacontroller/metacontroller@sha256:'
                            + 'b' * 64,
                        }
                    ]
                },
            }
        ]
    }


def manifests():
    return [
        {
            'kind': 'StatefulSet',
            'metadata': {'name': 'metacontroller'},
            'spec': {
                'template': {
                    'spec': {
                        'containers': [
                            {'name': 'metacontroller', 'image': 'old'}
                        ]
                    }
                }
            },
        },
        {
            'kind': 'CustomResourceDefinition',
            'metadata': {'name': 'decoratorcontrollers.metacontroller.k8s.io'},
        },
    ]


class MetacontrollerAssertionsTest(unittest.TestCase):
    def test_credential_values_are_not_written_as_receipt(self):
        fingerprint = credential_fingerprint(secret())
        self.assertEqual(len(fingerprint), 64)
        self.assertNotIn('YWNjZXNz', fingerprint)
        self.assertNotIn('c2VjcmV0', fingerprint)

    def test_secret_identity_and_credentials_must_both_survive(self):
        before = secret()
        require_unchanged_secret(before, copy.deepcopy(before))
        for field in ('uid', 'accesskey', 'secretkey'):
            after = copy.deepcopy(before)
            after['metadata' if field == 'uid' else 'data'][field] = 'changed'
            with self.subTest(field=field), self.assertRaises(AssertionError):
                require_unchanged_secret(before, after)
        with self.assertRaises(AssertionError):
            credential_fingerprint({'data': {'accesskey': 'only-one'}})

    def test_actual_ready_pod_immutable_reference_is_required(self):
        self.assertEqual(
            require_live_images(pods(), IMAGE)[0]['requested_image'], IMAGE
        )
        for mutation in (
            'wrong-ref',
            'not-ready',
            'missing-id',
            'missing-pod',
            'duplicate-pod',
        ):
            actual = pods()
            if mutation == 'wrong-ref':
                actual['items'][0]['spec']['containers'][0]['image'] = (
                    'unverified:latest'
                )
            elif mutation == 'not-ready':
                actual['items'][0]['status']['containerStatuses'][0][
                    'ready'
                ] = False
            elif mutation == 'missing-id':
                actual['items'][0]['status']['containerStatuses'][0][
                    'imageID'
                ] = ''
            elif mutation == 'missing-pod':
                actual['items'] = []
            else:
                actual['items'] *= 2
            with (
                self.subTest(mutation=mutation),
                self.assertRaises(AssertionError),
            ):
                require_live_images(actual, IMAGE)

    def test_only_expected_workload_image_changes(self):
        objects = manifests()
        before = copy.deepcopy(objects)
        set_metacontroller_image(objects, IMAGE)
        expected = copy.deepcopy(before)
        expected[0]['spec']['template']['spec']['containers'][0]['image'] = (
            IMAGE
        )
        self.assertEqual(objects, expected)
        with self.assertRaises(ValueError):
            set_metacontroller_image(objects, 'unverified:latest')
        with self.assertRaises(AssertionError):
            select_statefulset(objects + [objects[0]])


if __name__ == '__main__':
    unittest.main()
