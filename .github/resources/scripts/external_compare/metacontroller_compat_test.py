"""Guard the real-Metacontroller upgrade assertions."""

import copy
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace

from metacontroller_compat import (
    Exercise,
    credential_fingerprint,
    istio_crd_target,
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


class StopAfterCRDs(Exception):
    pass


class MetacontrollerAssertionsTest(unittest.TestCase):
    def test_uses_source_pinned_crd_prerequisite_not_install_overlay(self):
        sha = 'a' * 40
        expected = (
            'https://github.com/kubeflow/community-distribution/common/istio/istio-crds/base?ref='
            + sha
        )
        required = [
            'destinationrules.networking.istio.io',
            'authorizationpolicies.security.istio.io',
        ]
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            deploy = source / '.github/resources/scripts/deploy-kfp.sh'
            deploy.parent.mkdir(parents=True)
            deploy.write_text(
                'KUBEFLOW_MANIFESTS_SHA="'
                + sha
                + '"\n'
                + 'kubectl apply -k "https://github.com/kubeflow/community-distribution/common/istio/istio-crds/base?ref=${KUBEFLOW_MANIFESTS_SHA}"\n'
            )
            exercise = Exercise.__new__(Exercise)
            exercise.source = source
            exercise.pairs = {
                'source_sha': sha,
                'pairs': {
                    'metacontroller': {'baseline': 'old', 'candidate': 'new'},
                    'seaweedfs': {'baseline': 'store'},
                },
            }
            exercise.seaweed = 'baseline'
            exercise.evidence = source
            exercise.image_receipt = lambda image: {}
            calls = []

            def rendered(target):
                calls.append(str(target))
                return (
                    [
                        {
                            'kind': 'CustomResourceDefinition',
                            'metadata': {'name': name},
                        }
                        for name in required
                    ]
                    if str(target) == expected
                    else []
                )

            def applied(objects):
                self.assertEqual(
                    {obj['metadata']['name'] for obj in objects}, set(required)
                )
                raise StopAfterCRDs

            exercise.c = SimpleNamespace(
                run=lambda *args: sha,
                CLUSTER='test',
                SEAWEED=source / 'storage',
                rendered=rendered,
                seed_old_store=lambda *args: ({}, 'pvc'),
                exercise_profile=lambda *args: None,
                apply=applied,
            )
            with self.assertRaises(StopAfterCRDs):
                exercise.exercise()
            self.assertIn(expected, calls)

    def test_crd_source_rejects_mutable_missing_or_ambiguous_pins(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            deploy = source / '.github/resources/scripts/deploy-kfp.sh'
            deploy.parent.mkdir(parents=True)
            command = 'kubectl apply -k "https://github.com/kubeflow/community-distribution/common/istio/istio-crds/base?ref=${KUBEFLOW_MANIFESTS_SHA}"\n'
            pin = 'KUBEFLOW_MANIFESTS_SHA="' + 'a' * 40 + '"\n'
            for text in [
                command,
                'KUBEFLOW_MANIFESTS_SHA="main"\n' + command,
                pin + pin + command,
                pin + command.replace('istio-crds', 'istio-install'),
            ]:
                deploy.write_text(text)
                with self.subTest(text=text), self.assertRaises(ValueError):
                    istio_crd_target(source)

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
