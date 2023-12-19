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
            image='alpine',
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
            image='alpine',
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
            # TODO: update to not use executor_main once container components
            # supported
            full_command=['kfp.dsl.executor_main', 'something else'],
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
            # TODO: update to not use executor_main once container components
            # supported
            full_command=['kfp.dsl.executor_main', 'something else'],
            pipeline_root=os.path.abspath('my_root'),
            runner=local.DockerRunner(),
        )

        handler.run()
        self.mocked_docker_client.containers.run.assert_called_once_with(
            image='alpine',
            command=['kfp.dsl.executor_main', 'something else'],
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
                # TODO: update to not use executor_main once container components
                # supported
                full_command=['kfp.dsl.executor_main', 'something else'],
                pipeline_root='my_relpath',
                runner=local.DockerRunner(),
            ).run()


class TestPullImage(DockerMockTestCase):

    def test_with_tag(self):
        docker_task_handler.pull_image(
            client=docker.from_env(), image='foo:123')
        self.mocked_docker_client.images.pull.assert_called_once_with(
            repository='foo', tag='123')

    def test_with_no_tag(self):
        docker_task_handler.pull_image(client=docker.from_env(), image='foo')
        self.mocked_docker_client.images.pull.assert_called_once_with(
            repository='foo', tag='latest')


class TestE2E(DockerMockTestCase,
              testing_utilities.LocalRunnerEnvironmentTestCase):

    def test(self):
        local.init(runner=local.DockerRunner())

        @dsl.component
        def artifact_maker(x: str, a: Output[Artifact]):
            with open(a.path, 'w') as f:
                f.write(x)

        try:
            artifact_maker(x='foo')
        except Exception:
            # cannot get outputs if they aren't created due to mock
            pass

        run_mock = self.mocked_docker_client.containers.run
        run_mock.assert_called_once()
        kwargs = run_mock.call_args[1]
        self.assertEqual(
            kwargs['image'],
            'python:3.7',
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


if __name__ == '__main__':
    unittest.main()
