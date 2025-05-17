import os
from typing import Optional
import unittest
from unittest import mock

from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.dsl import Output
from kfp.local import podman_task_handler
from kfp.local import testing_utilities


class PodmanMockTestCase(unittest.TestCase):

    def setUp(self):
        super().setUp()
        self.podman_mock = mock.Mock()
        patcher = mock.patch('podman.PodmanClient')
        self.mocked_podman_client = patcher.start().return_value

        mock_container = mock.Mock()
        self.mocked_podman_client.containers.create.return_value = mock_container
        # mock successful run
        mock_container.logs.return_value = [
            'fake'.encode('utf-8'),
            'container'.encode('utf-8'),
            'logs'.encode('utf-8'),
        ]
        mock_container.exit_code = 0

    def tearDown(self):
        super().tearDown()
        self.podman_mock.reset_mock()


class TestRunContainer(PodmanMockTestCase):

    def test_no_volumes(self):
        podman_task_handler.run_container(
            client=self.mocked_podman_client,
            image='alpine',
            command=['echo', 'foo'],
            volumes={},
        )

        self.mocked_podman_client.containers.create.assert_called_once_with(
            image='alpine:latest',
            command=['echo', 'foo'],
            volumes={},
            detach=True,
        )

    def test_cwd_volume(self):
        current_test_dir = os.path.dirname(os.path.abspath(__file__))
        podman_task_handler.run_container(
            client=self.mocked_podman_client,
            image='alpine',
            command=['cat', '/localdir/podman_task_handler_test.py'],
            volumes={current_test_dir: {
                'bind': '/localdir',
                'mode': 'ro'
            }},
        )
        self.mocked_podman_client.containers.create.assert_called_once_with(
            image='alpine:latest',
            command=['cat', '/localdir/podman_task_handler_test.py'],
            volumes={current_test_dir: {
                'bind': '/localdir',
                'mode': 'ro'
            }},
            detach=True,
        )


class TestPodmanTaskHandler(PodmanMockTestCase):

    def test_get_volumes_to_mount(self):
        handler = podman_task_handler.PodmanTaskHandler(
            image='alpine',
            full_command=['echo', 'foo'],
            pipeline_root=os.path.abspath('my_root'),
            runner=local.PodmanRunner(),
        )
        volumes = handler.get_volumes_to_mount()
        self.assertEqual(
            volumes, {
                os.path.abspath('my_root'): {
                    'bind': os.path.abspath('my_root'),
                    'mode': 'rw'
                }
            })

    def test_run(self):
        handler = podman_task_handler.PodmanTaskHandler(
            image='alpine',
            full_command=['echo', 'foo'],
            pipeline_root=os.path.abspath('my_root'),
            runner=local.PodmanRunner(),
        )

        handler.run()
        self.mocked_podman_client.containers.create.assert_called_once_with(
            image='alpine:latest',
            command=['echo', 'foo'],
            volumes={
                os.path.abspath('my_root'): {
                    'bind': os.path.abspath('my_root'),
                    'mode': 'rw'
                }
            },
            detach=True,
        )

    def test_pipeline_root_relpath(self):
        with self.assertRaisesRegex(
                ValueError,
                r"'pipeline_root' should be an absolute path to correctly construct the volume mount specification\."
        ):
            podman_task_handler.PodmanTaskHandler(
                image='alpine',
                full_command=['echo', 'foo'],
                pipeline_root='my_relpath',
                runner=local.PodmanRunner(),
            ).run()


class TestE2E(PodmanMockTestCase,
              testing_utilities.LocalRunnerEnvironmentTestCase):

    def setUp(self):
        super().setUp()
        local.init(runner=local.PodmanRunner())

    def test_python(self):

        @dsl.component
        def artifact_maker(x: str, a: Output[Artifact]):
            with open(a.path, 'w') as f:
                f.write(x)

        try:
            artifact_maker(x='foo')
        # cannot get outputs if they aren't created due to mock
        except FileNotFoundError:
            pass

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'python:3.9',
        )
        self.assertTrue(
            any('def artifact_maker' in c for c in kwargs['command']))
        self.assertTrue(kwargs['detach'])
        root_vol_key = [
            key for key in kwargs['volumes'].keys() if 'local_outputs' in key
        ][0]
        self.assertEqual(kwargs['volumes'][root_vol_key]['bind'], root_vol_key)
        self.assertEqual(kwargs['volumes'][root_vol_key]['mode'], 'rw')

    def test_empty_container_component(self):

        @dsl.container_component
        def comp():
            return dsl.ContainerSpec(image='alpine')

        try:
            comp()
        # cannot get outputs if they aren't created due to mock
        except FileNotFoundError:
            pass

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], [])

    def test_container_component(self):

        @dsl.container_component
        def artifact_maker(x: str,):
            return dsl.ContainerSpec(
                image='alpine',
                command=['sh', '-c', f'echo prefix-{x}'],
            )

        try:
            artifact_maker(x='foo')
        # cannot get outputs if they aren't created due to mock
        except FileNotFoundError:
            pass

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], [
            'sh',
            '-c',
            'echo prefix-foo',
        ])
        self.assertTrue(kwargs['detach'])
        root_vol_key = [
            key for key in kwargs['volumes'].keys() if 'local_outputs' in key
        ][0]
        self.assertEqual(kwargs['volumes'][root_vol_key]['bind'], root_vol_key)
        self.assertEqual(kwargs['volumes'][root_vol_key]['mode'], 'rw')

    def test_if_present_with_string_omitted(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine:3.19.0',
                command=[
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=['echo', x],
                        else_=['echo', 'No input provided!'])
                ])

        comp()

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:3.19.0',
        )
        self.assertEqual(kwargs['command'], [
            'echo',
            'No input provided!',
        ])

    def test_if_present_with_string_provided(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine:3.19.0',
                command=[
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=['echo', x],
                        else_=['echo', 'No artifact provided!'])
                ])

        comp(x='foo')

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:3.19.0',
        )
        self.assertEqual(kwargs['command'], [
            'echo',
            'foo',
        ])

    def test_concat_placeholder(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[dsl.ConcatPlaceholder(['prefix-', x, '-suffix'])])

        comp()

        create_mock = self.mocked_podman_client.containers.create
        create_mock.assert_called_once()
        kwargs = create_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['prefix-null-suffix'])
