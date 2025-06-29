# Copyright 2023 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os
from typing import Optional
import unittest
from unittest import mock

import docker
from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.dsl import Output
from kfp.local import docker_task_handler
from kfp.local import testing_utilities


class DockerMockTestCase(unittest.TestCase):

    def setUp(self):
        super().setUp()
        self.docker_mock = mock.Mock()
        patcher = mock.patch('docker.from_env')
        self.mocked_docker_client = patcher.start().return_value

        mock_container = mock.Mock()
        self.mocked_docker_client.containers.run.return_value = mock_container
        # mock successful run
        mock_container.logs.return_value = [
            'fake'.encode('utf-8'),
            'container'.encode('utf-8'),
            'logs'.encode('utf-8'),
        ]
        mock_container.wait.return_value = {'StatusCode': 0}

    def teardown(self):
        super().tearDown()
        self.docker_mock.reset_mock()


class TestRunDockerContainer(DockerMockTestCase):

    def test_no_volumes(self):
        docker_task_handler.run_docker_container(
            docker.from_env(),
            image='alpine',
            command=['echo', 'foo'],
            volumes={},
        )

        self.mocked_docker_client.containers.run.assert_called_once_with(
            image='alpine:latest',
            command=['echo', 'foo'],
            detach=True,
            stdout=True,
            stderr=True,
            volumes={},
        )

    def test_cwd_volume(self):
        current_test_dir = os.path.dirname(os.path.abspath(__file__))
        docker_task_handler.run_docker_container(
            client=docker.from_env(),
            image='alpine',
            command=['cat', '/localdir/docker_task_handler_test.py'],
            volumes={current_test_dir: {
                'bind': '/localdir',
                'mode': 'ro'
            }},
        )
        self.mocked_docker_client.containers.run.assert_called_once_with(
            image='alpine:latest',
            command=['cat', '/localdir/docker_task_handler_test.py'],
            detach=True,
            stdout=True,
            stderr=True,
            volumes={current_test_dir: {
                'bind': '/localdir',
                'mode': 'ro'
            }})


class TestDockerTaskHandler(DockerMockTestCase):

    def test_get_volumes_to_mount(self):
        handler = docker_task_handler.DockerTaskHandler(
            image='alpine',
            full_command=['echo', 'foo'],
            pipeline_root=os.path.abspath('my_root'),
            runner=local.DockerRunner(),
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
        handler = docker_task_handler.DockerTaskHandler(
            image='alpine',
            full_command=['echo', 'foo'],
            pipeline_root=os.path.abspath('my_root'),
            runner=local.DockerRunner(),
        )

        handler.run()
        self.mocked_docker_client.containers.run.assert_called_once_with(
            image='alpine:latest',
            command=['echo', 'foo'],
            detach=True,
            stdout=True,
            stderr=True,
            volumes={
                os.path.abspath('my_root'): {
                    'bind': os.path.abspath('my_root'),
                    'mode': 'rw'
                }
            },
        )

    def test_pipeline_root_relpath(self):
        with self.assertRaisesRegex(
                ValueError,
                r"'pipeline_root' should be an absolute path to correctly construct the volume mount specification\."
        ):
            docker_task_handler.DockerTaskHandler(
                image='alpine',
                full_command=['echo', 'foo'],
                pipeline_root='my_relpath',
                runner=local.DockerRunner(),
            ).run()


class TestAddLatestTagIfNotPresent(unittest.TestCase):

    def test_no_tag(self):
        actual = docker_task_handler.add_latest_tag_if_not_present(
            image='alpine')
        expected = 'alpine:latest'
        self.assertEqual(actual, expected)

    def test_latest_tag(self):
        actual = docker_task_handler.add_latest_tag_if_not_present(
            image='alpine:latest')
        expected = 'alpine:latest'
        self.assertEqual(actual, expected)

    def test_no_tag(self):
        actual = docker_task_handler.add_latest_tag_if_not_present(
            image='alpine:3.19.0')
        expected = 'alpine:3.19.0'
        self.assertEqual(actual, expected)


class TestE2E(DockerMockTestCase,
              testing_utilities.LocalRunnerEnvironmentTestCase):

    def setUp(self):
        super().setUp()
        local.init(runner=local.DockerRunner())

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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'python:3.9',
        )
        self.assertTrue(
            any('def artifact_maker' in c for c in kwargs['command']))
        self.assertTrue(kwargs['detach'])
        self.assertTrue(kwargs['stdout'])
        self.assertTrue(kwargs['stderr'])
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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
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
        self.assertTrue(kwargs['stdout'])
        self.assertTrue(kwargs['stderr'])
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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:3.19.0',
        )
        self.assertEqual(kwargs['command'], [
            'echo',
            'foo',
        ])

    def test_if_present_single_element_with_string_omitted(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine:3.19.0',
                command=[
                    'echo',
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=x,
                        else_='No artifact provided!',
                    )
                ])

        comp()

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:3.19.0',
        )
        self.assertEqual(kwargs['command'], [
            'echo',
            'No artifact provided!',
        ])

    def test_if_present_single_element_with_string_provided(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine:3.19.0',
                command=[
                    'echo',
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=x,
                        else_='No artifact provided!',
                    )
                ])

        comp(x='foo')

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
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

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['prefix-null-suffix'])

    def test_nested_concat_placeholder(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                    dsl.ConcatPlaceholder(
                        ['a', dsl.ConcatPlaceholder(['b', x, 'd'])])
                ])

        comp(x='c')

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['echo', 'abcd'])

    def test_ifpresent_in_concat_provided(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                    dsl.ConcatPlaceholder([
                        'there ',
                        dsl.ConcatPlaceholder([
                            'is ',
                            dsl.IfPresentPlaceholder(
                                input_name='x',
                                then='one thing',
                                else_='another thing')
                        ])
                    ])
                ])

        comp(x='c')

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['echo', 'there is one thing'])

    def test_ifpresent_in_concat_omitted(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                    dsl.ConcatPlaceholder([
                        'there ',
                        dsl.ConcatPlaceholder([
                            'is ',
                            dsl.IfPresentPlaceholder(
                                input_name='x',
                                then='one thing',
                                else_='another thing')
                        ])
                    ])
                ])

        comp()

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['echo', 'there is another thing'])

    def test_concat_in_ifpresent_provided(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=dsl.ConcatPlaceholder([x]),
                        else_=dsl.ConcatPlaceholder(['something', ' ', 'else']))
                ])

        comp(x='something')

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['echo', 'something'])

    def test_concat_in_ifpresent_omitted(self):

        @dsl.container_component
        def comp(x: Optional[str] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=dsl.ConcatPlaceholder([x]),
                        else_=dsl.ConcatPlaceholder(['another', ' ', 'thing']))
                ])

        comp()

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'alpine:latest',
        )
        self.assertEqual(kwargs['command'], ['echo', 'another thing'])


if __name__ == '__main__':
    unittest.main()
