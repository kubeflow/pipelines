# Copyright 2021 The Kubeflow Authors
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
"""Tests for `components` command group in KFP CLI."""
import contextlib
import functools
import os
import pathlib
import subprocess
import textwrap
from typing import List, Optional, Union
import unittest
from unittest import mock

from click import testing
import docker

docker = mock.Mock()

from kfp.cli import component


def _make_component(
    func_name: str,
    base_image: Optional[str] = None,
    target_image: Optional[str] = None,
    packages_to_install: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    pip_index_urls: Optional[List[str]] = None,
) -> str:
    return textwrap.dedent('''
    from kfp.dsl import *

    @component(
        base_image={base_image},
        target_image={target_image},
        packages_to_install={packages_to_install},
        output_component_file={output_component_file},
        pip_index_urls={pip_index_urls})
    def {func_name}():
        pass
    ''').format(
        base_image=repr(base_image),
        target_image=repr(target_image),
        packages_to_install=repr(packages_to_install),
        output_component_file=repr(output_component_file),
        pip_index_urls=repr(pip_index_urls),
        func_name=func_name)


def _write_file(filename: str, file_contents: str):
    filepath = pathlib.Path(filename)
    filepath.parent.mkdir(exist_ok=True, parents=True)
    filepath.write_text(file_contents)


def _write_components(filename: str, component_template: Union[List[str], str]):
    if isinstance(component_template, list):
        file_contents = '\n\n'.join(component_template)
    else:
        file_contents = component_template
    _write_file(filename=filename, file_contents=file_contents)


class Test(unittest.TestCase):

    def setUp(self) -> None:
        self.runner = testing.CliRunner()
        self.cli = component.component
        component._DOCKER_IS_PRESENT = True

        patcher = mock.patch('docker.from_env')
        self._docker_client = patcher.start().return_value
        self._docker_client.images.build.return_value = [{
            'stream': 'Build logs'
        }]
        self._docker_client.images.push.return_value = [{'status': 'Pushed'}]
        self.addCleanup(patcher.stop)

        with contextlib.ExitStack() as stack:
            stack.enter_context(self.runner.isolated_filesystem())
            self._working_dir = pathlib.Path.cwd()
            self.addCleanup(stack.pop_all().close)

        return super().setUp()

    def assert_file_exists(self, path: str):
        path_under_test_dir = self._working_dir / path
        self.assertTrue(path_under_test_dir, f'File {path} does not exist!')

    def assert_file_exists_and_contains(self, path: str, expected_content: str):
        self.assert_file_exists(path)
        path_under_test_dir = self._working_dir / path
        got_content = path_under_test_dir.read_text()
        self.assertEqual(got_content, expected_content)

    def test_kfp_config_for_single_file(self):
        preprocess_component = _make_component(
            func_name='preprocess', target_image='custom-image')
        train_component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py',
                          [preprocess_component, train_component])

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

        self.assert_file_exists_and_contains(
            'kfp_config.ini',
            textwrap.dedent('''\
            [Components]
            preprocess = components.py
            train = components.py

            '''))

    def test_kfp_config_for_single_file_under_nested_directory(self):
        preprocess_component = _make_component(
            func_name='preprocess', target_image='custom-image')
        train_component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('dir1/dir2/dir3/components.py',
                          [preprocess_component, train_component])

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

        self.assert_file_exists_and_contains(
            'kfp_config.ini',
            textwrap.dedent('''\
            [Components]
            preprocess = dir1/dir2/dir3/components.py
            train = dir1/dir2/dir3/components.py

            '''))

    def test_kfp_config_for_multiple_files(self):
        component = _make_component(
            func_name='preprocess', target_image='custom-image')
        _write_components('preprocess_component.py', component)

        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('train_component.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

        self.assert_file_exists_and_contains(
            'kfp_config.ini',
            textwrap.dedent('''\
            [Components]
            preprocess = preprocess_component.py
            train = train_component.py

            '''))

    def test_kfp_config_for_multiple_files_under_nested_directories(self):
        component = _make_component(
            func_name='preprocess', target_image='custom-image')
        _write_components('preprocess/preprocess_component.py', component)

        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('train/train_component.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

        self.assert_file_exists_and_contains(
            'kfp_config.ini',
            textwrap.dedent('''\
            [Components]
            preprocess = preprocess/preprocess_component.py
            train = train/train_component.py

            '''))

    def test_target_image_must_be_the_same_in_all_components(self):
        component_one = _make_component(func_name='one', target_image='image-1')
        component_two = _make_component(func_name='two', target_image='image-1')
        _write_components('one_two/one_two.py', [component_one, component_two])

        component_three = _make_component(
            func_name='three', target_image='image-2')
        component_four = _make_component(
            func_name='four', target_image='image-3')
        _write_components('three_four/three_four.py',
                          [component_three, component_four])

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 1)

    def test_target_image_must_be_the_same_in_all_components_with_base_image(
            self):
        component_one = _make_component(
            func_name='one', base_image='image-1', target_image='target-image')
        component_two = _make_component(
            func_name='two', base_image='image-1', target_image='target-image')
        _write_components('one_two/one_two.py', [component_one, component_two])

        component_three = _make_component(
            func_name='three',
            base_image='image-2',
            target_image='target-image')
        component_four = _make_component(
            func_name='four', base_image='image-3', target_image='target-image')
        _write_components('three_four/three_four.py',
                          [component_three, component_four])

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 1)

    def test_component_filepattern_can_be_used_to_restrict_discovery(self):
        component = _make_component(
            func_name='preprocess', target_image='custom-image')
        _write_components('preprocess/preprocess_component.py', component)

        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('train/train_component.py', component)

        result = self.runner.invoke(
            self.cli,
            [
                'build',
                str(self._working_dir), '--component-filepattern=train/*'
            ],
        )
        self.assertEqual(result.exit_code, 0)

        self.assert_file_exists_and_contains(
            'kfp_config.ini',
            textwrap.dedent('''\
            [Components]
            train = train/train_component.py

            '''))

    def test_nested_module_imports(self):
        module_two = 'two = 2'
        _write_file('module_two.py', module_two)

        module_one = 'from module_two import two\none = 1'
        _write_file('module_one.py', module_one)

        component = _make_component(
            func_name='comp', target_image='custom-image')
        component = 'from module_one import one\n' + component
        _write_components('component.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

    def test_emptry_requirements_txt_file_is_generated(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(self.cli, ['build', str(self._working_dir)])
        self.assertEqual(result.exit_code, 0)
        self.assert_file_exists_and_contains('runtime-requirements.txt',
                                             '# Generated by KFP.\n')

    def test_existing_requirements_txt_file_is_updated(self):
        component = _make_component(
            func_name='train',
            target_image='custom-image',
            packages_to_install=[
                'tensorflow==2.10.0', 'google-cloud-pipeline-components'
            ])
        _write_components('components.py', component)

        _write_file('runtime-requirements.txt', 'Some pre-existing content')

        result = self.runner.invoke(self.cli, ['build', str(self._working_dir)])
        self.assertEqual(result.exit_code, 0)
        self.assert_file_exists_and_contains(
            'runtime-requirements.txt',
            '# Generated by KFP.\ngoogle-cloud-pipeline-components\ntensorflow==2.10.0'
        )

    def test_user_requirements_txt_file_is_unchanged(self):
        component = _make_component(
            func_name='train',
            target_image='custom-image',
            packages_to_install=['tensorflow==2.10.0'])
        _write_components('components.py', component)

        _write_file('requirements.txt', 'Some pre-existing content')

        result = self.runner.invoke(self.cli, ['build', str(self._working_dir)])
        self.assertEqual(result.exit_code, 0)
        self.assert_file_exists_and_contains('requirements.txt',
                                             'Some pre-existing content')

    def test_docker_ignore_file_is_generated(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(self.cli, ['build', str(self._working_dir)])
        self.assertEqual(result.exit_code, 0)
        self.assert_file_exists_and_contains(
            '.dockerignore',
            textwrap.dedent('''\
            # Generated by KFP.

            component_metadata/
            '''))

    def test_existing_docker_ignore_file_is_unchanged(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        _write_file('.dockerignore', 'Some pre-existing content')

        result = self.runner.invoke(self.cli, ['build', str(self._working_dir)])
        self.assertEqual(result.exit_code, 0)
        self.assert_file_exists_and_contains('.dockerignore',
                                             'Some pre-existing content')

    def test_docker_engine_is_supported(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)
        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir), '--engine=docker'],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self._docker_client.images.push.assert_called_once_with(
            'custom-image', stream=True, decode=True)

    def test_kaniko_engine_is_not_supported(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)
        with self.assertWarnsRegex(DeprecationWarning,
                                   r'The --engine option is deprecated'):
            result = self.runner.invoke(
                self.cli,
                ['build', str(self._working_dir), '--engine=kaniko'],
            )
        self.assertEqual(result.exit_code, 1)
        self._docker_client.api.build.assert_not_called()
        self._docker_client.images.push.assert_not_called()

    def test_cloud_build_engine_is_not_supported(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        with self.assertWarnsRegex(DeprecationWarning,
                                   r'The --engine option is deprecated'):
            result = self.runner.invoke(
                self.cli,
                ['build',
                 str(self._working_dir), '--engine=cloudbuild'],
            )

        self.assertEqual(result.exit_code, 1)
        self._docker_client.api.build.assert_not_called()
        self._docker_client.images.push.assert_not_called()

    def test_docker_client_is_called_to_build_and_push_by_default(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)

        self._docker_client.api.build.assert_called_once_with(
            path=mock.ANY,
            dockerfile='Dockerfile',
            tag=mock.ANY,
            decode=True,
            platform='linux/amd64',
        )
        self._docker_client.images.push.assert_called_once_with(
            'custom-image', stream=True, decode=True)

    def test_docker_client_is_not_called_to_build_or_push(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir), '--no-build-image'],
        )
        self.assertEqual(result.exit_code, 0)

        self._docker_client.api.build.assert_not_called()
        self._docker_client.images.push.assert_not_called()

    def test_docker_client_is_called_to_build_but_skips_pushing(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir), '--no-push-image'],
        )
        self.assertEqual(result.exit_code, 0)

        self._docker_client.api.build.assert_called_once()
        self._docker_client.images.push.assert_not_called()

    @mock.patch('kfp.__version__', '1.2.3')
    def test_docker_file_is_created_correctly(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists_and_contains(
            'Dockerfile',
            textwrap.dedent('''\
                # Generated by KFP.

                FROM python:3.7

                WORKDIR /usr/local/src/kfp/components
                COPY runtime-requirements.txt runtime-requirements.txt
                RUN pip install --no-cache-dir -r runtime-requirements.txt

                RUN pip install --no-cache-dir kfp==1.2.3
                COPY . .
                '''))

    @mock.patch('kfp.__version__', '1.2.3')
    def test_docker_file_is_created_correctly_with_one_url(self):
        component = _make_component(
            func_name='train',
            target_image='custom-image',
            pip_index_urls=['https://pypi.org/simple'])
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists_and_contains(
            'Dockerfile',
            textwrap.dedent('''\
                # Generated by KFP.

                FROM python:3.7

                WORKDIR /usr/local/src/kfp/components
                COPY runtime-requirements.txt runtime-requirements.txt
                RUN pip install --index-url https://pypi.org/simple --trusted-host https://pypi.org/simple --no-cache-dir -r runtime-requirements.txt

                RUN pip install --index-url https://pypi.org/simple --trusted-host https://pypi.org/simple --no-cache-dir kfp==1.2.3
                COPY . .
                '''))

    @mock.patch('kfp.__version__', '1.2.3')
    def test_docker_file_is_created_correctly_with_two_urls(self):
        component = _make_component(
            func_name='train',
            target_image='custom-image',
            pip_index_urls=[
                'https://pypi.org/simple', 'https://example.com/pypi/simple'
            ])
        _write_components('components.py', component)

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists_and_contains(
            'Dockerfile',
            textwrap.dedent('''\
                # Generated by KFP.

                FROM python:3.7

                WORKDIR /usr/local/src/kfp/components
                COPY runtime-requirements.txt runtime-requirements.txt
                RUN pip install --index-url https://pypi.org/simple --trusted-host https://pypi.org/simple --extra-index-url https://example.com/pypi/simple --trusted-host https://example.com/pypi/simple --no-cache-dir -r runtime-requirements.txt

                RUN pip install --index-url https://pypi.org/simple --trusted-host https://pypi.org/simple --extra-index-url https://example.com/pypi/simple --trusted-host https://example.com/pypi/simple --no-cache-dir kfp==1.2.3
                COPY . .
                '''))

    def test_existing_dockerfile_is_unchanged_by_default(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)
        _write_file('Dockerfile', 'Existing Dockerfile contents')

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir)],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists_and_contains('Dockerfile',
                                             'Existing Dockerfile contents')

    @mock.patch('kfp.__version__', '1.2.3')
    def test_existing_dockerfile_can_be_overwritten(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)
        _write_file('Dockerfile', 'Existing Dockerfile contents')

        result = self.runner.invoke(
            self.cli,
            ['build', str(self._working_dir), '--overwrite-dockerfile'],
        )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists_and_contains(
            'Dockerfile',
            textwrap.dedent('''\
                # Generated by KFP.

                FROM python:3.7

                WORKDIR /usr/local/src/kfp/components
                COPY runtime-requirements.txt runtime-requirements.txt
                RUN pip install --no-cache-dir -r runtime-requirements.txt

                RUN pip install --no-cache-dir kfp==1.2.3
                COPY . .
                '''))

    def test_dockerfile_can_contain_custom_kfp_package(self):
        component = _make_component(
            func_name='train', target_image='custom-image')
        _write_components('components.py', component)
        current_dir = os.path.dirname(os.path.abspath(__file__))
        package_dir = os.path.dirname(os.path.dirname(current_dir))

        # suppresses large stdout from subprocess that builds kfp package
        with mock.patch.object(
                subprocess,
                'run',
                new=functools.partial(subprocess.run, capture_output=True)):
            result = self.runner.invoke(
                self.cli,
                [
                    'build',
                    str(self._working_dir), f'--kfp-package-path={package_dir}'
                ],
            )
        self.assertEqual(result.exit_code, 0)
        self._docker_client.api.build.assert_called_once()
        self.assert_file_exists('Dockerfile')
        with open('Dockerfile', 'r') as f:
            contents = f.read()
        file_start = textwrap.dedent('''\
                # Generated by KFP.

                FROM python:3.7

                WORKDIR /usr/local/src/kfp/components
                COPY runtime-requirements.txt runtime-requirements.txt
                RUN pip install --no-cache-dir -r runtime-requirements.txt
                ''')
        self.assertTrue(contents.startswith(file_start))
        self.assertRegex(contents, 'RUN pip install --no-cache-dir kfp-*')


if __name__ == '__main__':
    unittest.main()
