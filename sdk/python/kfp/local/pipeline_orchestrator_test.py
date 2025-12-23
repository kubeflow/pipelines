# Copyright 2024 The Kubeflow Authors
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
"""Tests for pipeline_orchestrator.py."""
import functools
import io as stdlib_io
import os
import sys
from typing import List, NamedTuple
import unittest
from unittest import mock

from kfp import dsl
from kfp import local
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Model
from kfp.dsl import Output
from kfp.dsl import pipeline_task
from kfp.local import testing_utilities
import pytest

ROOT_FOR_TESTING = './testing_root'

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@pytest.fixture(autouse=True)
def set_packages_for_test_classes(monkeypatch, request):
    if request.cls and request.cls.__name__ in {
            'TestRunLocalPipeline',
            'TestFstringContainerComponent',
    }:
        root_dir = os.path.dirname(
            os.path.dirname(
                os.path.dirname(os.path.dirname(os.path.dirname(__file__)))))
        kfp_pipeline_spec_path = os.path.join(root_dir, 'api', 'v2alpha1',
                                              'python')
        original_dsl_component = dsl.component
        monkeypatch.setattr(
            dsl, 'component',
            functools.partial(
                original_dsl_component,
                packages_to_install=[kfp_pipeline_spec_path]))


class TestRunLocalPipeline(testing_utilities.LocalRunnerEnvironmentTestCase):

    def assert_output_dir_contents(
        self,
        expected_dirs_in_pipeline_root: int,
        expected_files_in_pipeline_dir: int,
    ) -> None:
        # check that output files are correctly nested
        # and only one directory for the outer pipeline in pipeline root
        actual_dirs_in_pipeline_root = os.listdir(ROOT_FOR_TESTING)
        self.assertLen(
            actual_dirs_in_pipeline_root,
            expected_dirs_in_pipeline_root,
        )

        # and check that each task has a directory
        actual_contents_of_pipeline_dir = os.listdir(
            os.path.join(
                ROOT_FOR_TESTING,
                actual_dirs_in_pipeline_root[0],
            ))
        self.assertLen(
            actual_contents_of_pipeline_dir,
            expected_files_in_pipeline_dir,
        )

    def test_must_initialize(self):

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline():
            identity(string='foo')

        with self.assertRaisesRegex(
                RuntimeError,
                r"Local environment not initialized\. Please run 'kfp\.local\.init\(\)' before executing tasks locally\."
        ):
            my_pipeline()

    def test_no_io(self):
        local.init(
            local.SubprocessRunner(use_venv=False),
            pipeline_root=ROOT_FOR_TESTING)

        @dsl.component()
        def pass_op():
            pass

        @dsl.pipeline
        def my_pipeline():
            pass_op()
            pass_op()

        result = my_pipeline()
        self.assertIsInstance(result, pipeline_task.PipelineTask)
        self.assertEqual(result.outputs, {})
        self.assert_output_dir_contents(1, 2)

    def test_missing_args(self):
        local.init(local.SubprocessRunner())

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline(string: str) -> str:
            t1 = identity(string=string)
            t2 = identity(string=t1.output)
            return t2.output

        with self.assertRaisesRegex(
                TypeError,
                r'my-pipeline\(\) missing 1 required argument: string\.'):
            my_pipeline()

    def test_single_return(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline(string: str = 'text') -> str:
            t1 = identity(string=string)
            t2 = identity(string=t1.output)
            return t2.output

        task = my_pipeline()
        self.assertEqual(task.output, 'text')
        self.assert_output_dir_contents(1, 2)

    def test_can_run_loaded_pipeline(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline(string: str = 'text') -> str:
            t1 = identity(string=string)
            t2 = identity(string=t1.output)
            return t2.output

        my_pipeline_loaded = testing_utilities.compile_and_load_component(
            my_pipeline)

        task = my_pipeline_loaded(string='foo')
        self.assertEqual(task.output, 'foo')
        self.assert_output_dir_contents(1, 2)

    def test_all_param_io(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        # tests all I/O types with:
        # - use of component args/defaults
        # - use of pipeline args/defaults
        # - passing pipeline args to first run component and not first run component
        # - passing args from pipeline param, upstream output, and constant
        # - pipeline surfacing outputs from last run component and not last run component

        @dsl.component
        def many_parameter_component(
            a_float: float,
            a_boolean: bool,
            a_dict: dict,
            a_string: str = 'default',
            an_integer: int = 12,
            a_list: list = ['item1', 'item2'],
        ) -> NamedTuple(
                'outputs',
                a_string=str,
                a_float=float,
                an_integer=int,
                a_boolean=bool,
                a_list=list,
                a_dict=dict,
        ):
            outputs = NamedTuple(
                'outputs',
                a_string=str,
                a_float=float,
                an_integer=int,
                a_boolean=bool,
                a_list=list,
                a_dict=dict,
            )
            return outputs(
                a_string=a_string,
                a_float=a_float,
                an_integer=an_integer,
                a_boolean=a_boolean,
                a_list=a_list,
                a_dict=a_dict,
            )

        @dsl.pipeline
        def my_pipeline(
            flt: float,
            boolean: bool,
            dictionary: dict = {'foo': 'bar'},
        ) -> NamedTuple(
                'outputs',
                another_string=str,
                another_float=float,
                another_integer=int,
                another_boolean=bool,
                another_list=list,
                another_dict=dict,
        ):

            t1 = many_parameter_component(
                a_float=flt,
                a_boolean=True,
                a_dict={'baz': 'bat'},
                an_integer=10,
            )
            t2 = many_parameter_component(
                a_float=t1.outputs['a_float'],
                a_dict=dictionary,
                a_boolean=boolean,
            )

            outputs = NamedTuple(
                'outputs',
                another_string=str,
                another_float=float,
                another_integer=int,
                another_boolean=bool,
                another_list=list,
                another_dict=dict,
            )
            return outputs(
                another_string=t1.outputs['a_string'],
                another_float=t1.outputs['a_float'],
                another_integer=t1.outputs['an_integer'],
                another_boolean=t2.outputs['a_boolean'],
                another_list=t2.outputs['a_list'],
                another_dict=t1.outputs['a_dict'],
            )

        task = my_pipeline(
            flt=2.718,
            boolean=False,
        )
        self.assertEqual(task.outputs['another_string'], 'default')
        self.assertEqual(task.outputs['another_float'], 2.718)
        self.assertEqual(task.outputs['another_integer'], 10)
        self.assertEqual(task.outputs['another_boolean'], False)
        self.assertEqual(task.outputs['another_list'], ['item1', 'item2'])
        self.assertEqual(task.outputs['another_dict'], {'baz': 'bat'})
        self.assert_output_dir_contents(1, 2)

    def test_artifact_io(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def make_dataset(content: str) -> Dataset:
            d = Dataset(uri=dsl.get_uri(), metadata={'framework': 'pandas'})
            with open(d.path, 'w') as f:
                f.write(content)
            return d

        @dsl.component
        def make_model(dataset: Input[Dataset], model: Output[Model]):
            with open(dataset.path) as f:
                content = f.read()
            with open(model.path, 'w') as f:
                f.write(content * 2)
            model.metadata['framework'] = 'tensorflow'
            model.metadata['dataset'] = dataset.metadata

        @dsl.pipeline
        def my_pipeline(content: str = 'string') -> Model:
            t1 = make_dataset(content=content)
            t2 = make_model(dataset=t1.output)
            return t2.outputs['model']

        task = my_pipeline(content='text')
        output_model = task.output
        self.assertIsInstance(output_model, Model)
        self.assertEqual(output_model.name, 'model')
        self.assertTrue(output_model.uri.endswith('/make-model/model'))
        self.assertEqual(output_model.metadata, {
            'framework': 'tensorflow',
            'dataset': {
                'framework': 'pandas'
            }
        })
        self.assert_output_dir_contents(1, 2)

    def test_notebook_component_local_exec(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        import json as _json
        import os as _os
        import tempfile as _tempfile

        nb = {
            'cells': [
                {
                    'cell_type': 'code',
                    'execution_count': None,
                    'metadata': {
                        'tags': ['parameters']
                    },
                    'outputs': [],
                    'source': ['# parameters\n', "text='hello'\n"],
                },
                {
                    'cell_type':
                        'code',
                    'execution_count':
                        None,
                    'metadata': {},
                    'outputs': [],
                    'source': [
                        'import os\n',
                        "os.makedirs('/tmp/kfp_nb_outputs', exist_ok=True)\n",
                        "with open('/tmp/kfp_nb_outputs/log.txt','w') as f: f.write(text)\n",
                    ],
                },
            ],
            'metadata': {
                'kernelspec': {
                    'display_name': 'Python 3',
                    'language': 'python',
                    'name': 'python3',
                },
                'language_info': {
                    'name': 'python',
                    'version': '3.11'
                },
            },
            'nbformat': 4,
            'nbformat_minor': 5,
        }

        with _tempfile.TemporaryDirectory() as tmpdir:
            nb_path = _os.path.join(tmpdir, 'nb.ipynb')
            with open(nb_path, 'w', encoding='utf-8') as f:
                _json.dump(nb, f)

            @dsl.notebook_component(
                notebook_path=nb_path, kfp_package_path=_KFP_PACKAGE_PATH)
            def nb_comp(msg: str) -> str:
                dsl.run_notebook(text=msg)

                with open(
                        '/tmp/kfp_nb_outputs/log.txt', 'r',
                        encoding='utf-8') as f:
                    return f.read()

            @dsl.pipeline
            def my_pipeline() -> str:
                comp_result = nb_comp(msg='hi')
                return comp_result.output

            result = my_pipeline()
            self.assertEqual(result.output, 'hi')

        self.assert_output_dir_contents(1, 1)

    def test_embedded_artifact_local_exec(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        import os as _os
        import tempfile as _tempfile

        with _tempfile.TemporaryDirectory() as srcdir:
            with open(
                    _os.path.join(srcdir, 'data.txt'), 'w',
                    encoding='utf-8') as f:
                f.write('EMBED')

            @dsl.component(embedded_artifact_path=srcdir)
            def use_embed(cfg: dsl.EmbeddedInput[dsl.Dataset]) -> dsl.Dataset:
                out = dsl.Dataset(uri=dsl.get_uri('out'))
                import os
                import shutil
                shutil.copy(os.path.join(cfg.path, 'data.txt'), out.path)
                return out

            @dsl.pipeline
            def my_pipeline() -> dsl.Dataset:
                return use_embed().output

            task = my_pipeline()
            out_ds = task.output
            with open(out_ds.path, 'r', encoding='utf-8') as f:
                self.assertEqual(f.read(), 'EMBED')
            self.assert_output_dir_contents(1, 1)

    def test_notebook_component_invalid_notebook_raises(self):
        local.init(
            local.SubprocessRunner(),
            pipeline_root=ROOT_FOR_TESTING,
            raise_on_error=True)

        import os as _os
        import tempfile as _tempfile

        tmpdir = _tempfile.mkdtemp()
        bad_nb = _os.path.join(tmpdir, 'bad.ipynb')
        with open(bad_nb, 'w', encoding='utf-8') as f:
            f.write('not a json')

        @dsl.notebook_component(notebook_path=bad_nb,)
        def nb_comp():
            dsl.run_notebook()

        @dsl.pipeline
        def my_pipeline():
            nb_comp()

        with self.assertRaises(RuntimeError):
            my_pipeline()

    def test_input_artifact_constant_not_permitted(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def print_model(model: Input[Model]):
            print(model.name)
            print(model.uri)
            print(model.metadata)

        with self.assertRaisesRegex(
                ValueError,
                r"Input artifacts are not supported\. Got input artifact of type 'Model'\."
        ):

            @dsl.pipeline
            def my_pipeline():
                print_model(model=dsl.Model(name='model', uri='/foo/bar/model'))

    def test_importer(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def artifact_printer(a: Dataset):
            print(a)

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline(greeting: str) -> Dataset:
            world_op = identity(string='world')
            message_op = identity(string='message')
            imp_op = dsl.importer(
                artifact_uri='/local/path/to/dataset',
                artifact_class=Dataset,
                metadata={
                    message_op.output: [greeting, world_op.output],
                })
            artifact_printer(a=imp_op.outputs['artifact'])
            return imp_op.outputs['artifact']

        task = my_pipeline(greeting='hello')
        output_model = task.output
        self.assertIsInstance(output_model, Dataset)
        self.assertEqual(output_model.name, 'artifact')
        self.assertEqual(output_model.uri, '/local/path/to/dataset')
        self.assertEqual(output_model.metadata, {
            'message': ['hello', 'world'],
        })
        # importer doesn't have an output directory
        self.assert_output_dir_contents(1, 3)

    def test_pipeline_in_pipeline_simple(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def inner_pipeline() -> str:
            return identity(string='foo').output

        @dsl.pipeline
        def my_pipeline() -> str:
            return inner_pipeline().output

        task = my_pipeline()
        self.assertEqual(task.output, 'foo')
        self.assert_output_dir_contents(1, 1)

    def test_pipeline_in_pipeline_complex(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def square(x: float) -> float:
            return x**2

        @dsl.component
        def add(x: float, y: float) -> float:
            return x + y

        @dsl.component
        def square_root(x: float) -> float:
            return x**.5

        @dsl.component
        def convert_to_artifact(val: float) -> Dataset:
            dataset = Dataset(name='dataset', uri=dsl.get_uri())
            with open(dataset.path, 'w') as f:
                f.write(str(val))

        @dsl.component
        def convert_from_artifact(dataset: Dataset) -> float:
            with open(dataset.path) as f:
                return float(f.read())

        @dsl.pipeline
        def square_and_sum(a: float, b: float) -> Dataset:
            a_sq_task = square(x=a)
            b_sq_task = square(x=b)
            add_task = add(x=a_sq_task.output, y=b_sq_task.output)
            return convert_to_artifact(val=add_task.output).output

        @dsl.pipeline
        def pythagorean(a: float = 1.2, b: float = 1.2) -> float:
            sq_and_sum_task = square_and_sum(a=a, b=b)
            make_float_task = convert_from_artifact(
                dataset=sq_and_sum_task.output)
            return square_root(x=make_float_task.output).output

        @dsl.pipeline
        def pythagorean_then_add(
            side: float,
            addend: float = 42.24,
        ) -> float:
            t = pythagorean(a=side, b=1.2)
            return add(x=t.output, y=addend).output

        task = pythagorean_then_add(side=2.2)
        self.assertAlmostEqual(task.output, 44.745992817228334)
        self.assert_output_dir_contents(1, 7)

    def test_parallel_for_supported(self):
        # Use use_venv=False to avoid pip race conditions when installing
        # packages in parallel virtual environments
        local.init(
            local.SubprocessRunner(use_venv=False),
            pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def pass_op():
            pass

        @dsl.pipeline
        def my_pipeline():
            with dsl.ParallelFor([1, 2, 3]):
                pass_op()

        # Should now execute successfully with enhanced orchestrator
        task = my_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)
        # ParallelFor detection should route to enhanced orchestrator
        # and execute tasks in parallel

    def test_parallel_for_with_venv(self):
        # Use use_venv=True with pip installation serialization enabled
        # This test demonstrates the fix for pip race conditions
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=True  # Enable serialization to prevent race conditions
            ),
            pipeline_root=ROOT_FOR_TESTING)

        # Use numpy>=1.26.0 for Python 3.13 compatibility (has pre-built wheels)
        import sys
        numpy_version = 'numpy>=1.26.0' if sys.version_info >= (
            3, 13) else 'numpy==1.24.3'

        @dsl.component(packages_to_install=[numpy_version])
        def package_using_op() -> List[int]:
            import numpy as np
            return np.array([1, 2, 3]).tolist()

        @dsl.pipeline
        def my_pipeline():
            # Use more parallel tasks to increase chance of race condition
            with dsl.ParallelFor([1, 2, 3, 4, 5, 6, 7, 8]):
                package_using_op()

        # This test now passes reliably with pip installation serialization
        task = my_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

    def test_parallel_for_with_venv_race_condition(self):
        # Use use_venv=True to demonstrate pip race conditions when installing
        # packages in parallel virtual environments
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=True  # Enable serialization to prevent race conditions
            ),
            pipeline_root=ROOT_FOR_TESTING)

        # Use numpy>=1.26.0 for Python 3.13 compatibility (has pre-built wheels)
        import sys
        numpy_version = 'numpy>=1.26.0' if sys.version_info >= (
            3, 13) else 'numpy==1.24.3'

        @dsl.component(packages_to_install=[numpy_version])
        def package_using_op() -> List[int]:
            import numpy as np
            return np.array([1, 2, 3]).tolist()

        @dsl.pipeline
        def my_pipeline():
            # Use more parallel tasks to increase chance of race condition
            with dsl.ParallelFor([1, 2, 3, 4, 5, 6, 7, 8]):
                package_using_op()

        # This test now passes reliably with pip installation serialization
        task = my_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

    def test_parallel_for_with_max_concurrent_pip_installs(self):
        # Use use_venv=True with pip installation serialization enabled
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=False,  # Allow concurrent installs
                max_concurrent_pip_installs=2,  # But limit to 2 concurrent
            ),
            pipeline_root=ROOT_FOR_TESTING,
            raise_on_error=False)

        # Use packages that require installation to demonstrate serialization
        @dsl.component(packages_to_install=['requests==2.28.2'])
        def network_task(url: str) -> str:
            # Simple task that imports requests but doesn't make actual network calls
            return f'Would fetch from {url}'

        @dsl.pipeline
        def my_pipeline():
            # Use parallel tasks that will install packages concurrently
            urls = ['http://example.com/' + str(i) for i in range(4)]
            with dsl.ParallelFor(urls) as url:
                network_task(url=url)

        # This should work reliably with concurrent pip installations
        task = my_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # Check concurrent execution stats
        from kfp.local.pip_install_manager import pip_install_manager
        stats = pip_install_manager.get_stats()
        self.assertLessEqual(stats['concurrent_peak'],
                             2)  # Should respect max concurrent limit

    def test_condition_supported(self):
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

        @dsl.component
        def pass_op():
            pass

        @dsl.pipeline
        def my_pipeline(x: str):
            with dsl.Condition(x == 'foo'):
                pass_op()

        # Should now execute successfully with enhanced orchestrator
        task = my_pipeline(x='bar')
        self.assertIsInstance(task, pipeline_task.PipelineTask)
        # Condition detection should route to enhanced orchestrator
        # and evaluate the condition (false in this case, so task is skipped)

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_fails_with_raise_on_error_true(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=True)

        @dsl.component
        def raise_component():
            raise Exception('Error from raise_component.')

        @dsl.pipeline
        def my_pipeline():
            raise_component()

        with self.assertRaisesRegex(
                RuntimeError,
                r"Pipeline \x1b\[95m\'my-pipeline\'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\. Inner task failed: \x1b\[96m\'raise-component\'\x1b\[0m\.",
        ):
            my_pipeline()

        logged_output = mock_stdout.getvalue()
        # Logs should:
        # - log task failure trace
        # - log pipeline failure
        # - indicate which task the failure came from
        self.assertRegex(
            logged_output,
            r"raise Exception\('Error from raise_component\.'\)",
        )

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_single_nested_fails_with_raise_on_error_true(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=True)

        @dsl.component
        def fail():
            raise Exception('Nested failure!')

        @dsl.pipeline
        def inner_pipeline():
            fail()

        @dsl.pipeline
        def my_pipeline():
            inner_pipeline()

        with self.assertRaisesRegex(
                RuntimeError,
                r"Pipeline \x1b\[95m\'my-pipeline\'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\. Inner task failed: \x1b\[96m\'fail\'\x1b\[0m inside \x1b\[96m\'inner-pipeline\'\x1b\[0m\.",
        ):
            my_pipeline()

        logged_output = mock_stdout.getvalue()
        self.assertRegex(
            logged_output,
            r"raise Exception\('Nested failure!'\)",
        )

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_deeply_nested_fails_with_raise_on_error_true(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=True)

        @dsl.component
        def fail():
            raise Exception('Nested failure!')

        @dsl.pipeline
        def deep_pipeline():
            fail()

        @dsl.pipeline
        def mid_pipeline():
            deep_pipeline()

        @dsl.pipeline
        def outer_pipeline():
            mid_pipeline()

        with self.assertRaisesRegex(
                RuntimeError,
                r"Pipeline \x1b\[95m\'outer-pipeline\'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\. Inner task failed: \x1b\[96m\'fail\'\x1b\[0m\ inside \x1b\[96m\'deep-pipeline\'\x1b\[0m inside \x1b\[96m\'mid-pipeline\'\x1b\[0m\.",
        ):
            outer_pipeline()

        logged_output = mock_stdout.getvalue()
        self.assertRegex(
            logged_output,
            r"raise Exception\('Nested failure!'\)",
        )

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_fails_with_raise_on_error_false(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=False)

        @dsl.component
        def raise_component():
            raise Exception('Error from raise_component.')

        @dsl.pipeline
        def my_pipeline():
            raise_component()

        task = my_pipeline()
        logged_output = mock_stdout.getvalue()
        # Logs should:
        # - log task failure trace
        # - log pipeline failure
        # - indicate which task the failure came from
        self.assertRegex(
            logged_output,
            r"raise Exception\('Error from raise_component\.'\)",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Task \x1b\[96m'raise-component'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\n",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Pipeline \x1b\[95m'my-pipeline'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\. Inner task failed: \x1b\[96m'raise-component'\x1b\[0m\.\n",
        )
        self.assertEqual(task.outputs, {})

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_single_nested_fails_with_raise_on_error_false(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=False)

        @dsl.component
        def fail():
            raise Exception('Nested failure!')

        @dsl.pipeline
        def inner_pipeline():
            fail()

        @dsl.pipeline
        def my_pipeline():
            inner_pipeline()

        task = my_pipeline()
        logged_output = mock_stdout.getvalue()
        # Logs should:
        # - log task failure trace
        # - log pipeline failure
        # - indicate which task the failure came from
        self.assertRegex(
            logged_output,
            r"raise Exception\('Nested failure!'\)",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Task \x1b\[96m'fail'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\n",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Pipeline \x1b\[95m\'my-pipeline\'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m. Inner task failed: \x1b\[96m\'fail\'\x1b\[0m inside \x1b\[96m\'inner-pipeline\'\x1b\[0m.\n"
        )
        self.assertEqual(task.outputs, {})

    @mock.patch('sys.stdout', new_callable=stdlib_io.StringIO)
    def test_deeply_nested_fails_with_raise_on_error_false(self, mock_stdout):
        local.init(local.SubprocessRunner(), raise_on_error=False)

        @dsl.component
        def fail():
            raise Exception('Nested failure!')

        @dsl.pipeline
        def deep_pipeline():
            fail()

        @dsl.pipeline
        def mid_pipeline():
            deep_pipeline()

        @dsl.pipeline
        def outer_pipeline():
            mid_pipeline()

        task = outer_pipeline()
        logged_output = mock_stdout.getvalue()
        # Logs should:
        # - log task failure trace
        # - log pipeline failure
        # - indicate which task the failure came from
        self.assertRegex(
            logged_output,
            r"raise Exception\('Nested failure!'\)",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Task \x1b\[96m'fail'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\n",
        )
        self.assertRegex(
            logged_output,
            r"ERROR - Pipeline \x1b\[95m'outer-pipeline'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m\. Inner task failed: \x1b\[96m'fail'\x1b\[0m inside \x1b\[96m'deep-pipeline'\x1b\[0m inside \x1b\[96m'mid-pipeline'\x1b\[0m\.\n",
        )
        self.assertEqual(task.outputs, {})

    def test_fstring_python_component(self):
        local.init(runner=local.SubprocessRunner())

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.pipeline
        def my_pipeline(string: str = 'baz') -> str:
            op1 = identity(string=f'bar-{string}')
            op2 = identity(string=f'foo-{op1.output}')
            return op2.output

        task = my_pipeline()
        self.assertEqual(task.output, 'foo-bar-baz')

    def test_workspace_functionality(self):
        import tempfile

        # Create temporary directory for workspace
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace_root = os.path.join(temp_dir, 'workspace')
            os.makedirs(workspace_root, exist_ok=True)

            local.init(
                local.SubprocessRunner(),
                pipeline_root=ROOT_FOR_TESTING,
                workspace_root=workspace_root)

            @dsl.component
            def write_to_workspace(text: str, workspace_path: str) -> str:
                import os
                output_file = os.path.join(workspace_path, 'output.txt')
                with open(output_file, 'w') as f:
                    f.write(text)
                return output_file

            @dsl.component
            def read_from_workspace(file_path: str) -> str:
                with open(file_path, 'r') as f:
                    return f.read()

            @dsl.pipeline(
                pipeline_config=dsl.PipelineConfig(
                    workspace=dsl.WorkspaceConfig(size='1Gi')))
            def my_pipeline(text: str = 'Hello workspace!') -> str:
                # Write to workspace
                write_task = write_to_workspace(
                    text=text, workspace_path=dsl.WORKSPACE_PATH_PLACEHOLDER)

                # Read from workspace
                read_task = read_from_workspace(file_path=write_task.output)

                return read_task.output

            task = my_pipeline(text='Test workspace functionality!')
            self.assertEqual(task.output, 'Test workspace functionality!')
            self.assert_output_dir_contents(1, 2)

    def test_docker_runner_workspace_functionality(self):
        import tempfile

        # Create temporary directory for workspace
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace_root = os.path.join(temp_dir, 'workspace')
            os.makedirs(workspace_root, exist_ok=True)

            # Test that DockerRunner can be initialized with workspace
            local.init(
                local.DockerRunner(),
                pipeline_root=ROOT_FOR_TESTING,
                workspace_root=workspace_root)

            # Verify that the workspace is properly configured
            self.assertEqual(
                local.config.LocalExecutionConfig.instance.workspace_root,
                workspace_root)
            self.assertEqual(
                type(local.config.LocalExecutionConfig.instance.runner),
                local.DockerRunner)


class TestFstringContainerComponent(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    @classmethod
    def setUpClass(cls):
        from kfp.local import subprocess_task_handler

        # Temporarily removing these these validation calls is useful hack to
        # test a ContainerComponent outside of a container.
        # We do this here because we only want to test the very specific
        # f-string logic in container components without the presence of
        # Docker in the test environment.
        cls.original_validate_image = subprocess_task_handler.SubprocessTaskHandler.validate_image
        subprocess_task_handler.SubprocessTaskHandler.validate_image = lambda slf, image: None

        cls.original_validate_not_container_component = subprocess_task_handler.SubprocessTaskHandler.validate_not_container_component
        subprocess_task_handler.SubprocessTaskHandler.validate_not_container_component = lambda slf, full_command: None

        cls.original_validate_not_containerized_python_component = subprocess_task_handler.SubprocessTaskHandler.validate_not_containerized_python_component
        subprocess_task_handler.SubprocessTaskHandler.validate_not_containerized_python_component = lambda slf, full_command: None

    @classmethod
    def tearDownClass(cls):
        from kfp.local import subprocess_task_handler

        subprocess_task_handler.SubprocessTaskHandler.validate_image = cls.original_validate_image
        subprocess_task_handler.SubprocessTaskHandler.validate_not_container_component = cls.original_validate_not_container_component
        subprocess_task_handler.SubprocessTaskHandler.validate_not_containerized_python_component = cls.original_validate_not_containerized_python_component

    def test_fstring_container_component(self):
        local.init(runner=local.SubprocessRunner())

        @dsl.container_component
        def identity_container(string: str, outpath: dsl.OutputPath(str)):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'sh',
                    '-c',
                    f"""mkdir -p $(dirname {outpath}) && printf '%s' {string} > {outpath}""",
                ])

        @dsl.pipeline
        def my_pipeline(string: str = 'baz') -> str:
            op1 = identity_container(string=f'bar-{string}')
            op2 = identity_container(string=f'foo-{op1.output}')
            return op2.output

        task = my_pipeline()
        self.assertEqual(task.output, 'foo-bar-baz')


if __name__ == '__main__':
    unittest.main()
