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
from typing import NamedTuple
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


@pytest.fixture(autouse=True)
def set_packages_for_test_classes(monkeypatch, request):
    if request.cls.__name__ in {
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
        local.init(local.SubprocessRunner(), pipeline_root=ROOT_FOR_TESTING)

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

    def test_parallel_for_not_supported(self):
        local.init(local.SubprocessRunner())

        @dsl.component
        def pass_op():
            pass

        @dsl.pipeline
        def my_pipeline():
            with dsl.ParallelFor([1, 2, 3]):
                pass_op()

        with self.assertRaisesRegex(
                NotImplementedError,
                r"'dsl\.ParallelFor' is not supported by local pipeline execution\."
        ):
            my_pipeline()

    def test_condition_not_supported(self):
        local.init(local.SubprocessRunner())

        @dsl.component
        def pass_op():
            pass

        @dsl.pipeline
        def my_pipeline(x: str):
            with dsl.Condition(x == 'foo'):
                pass_op()

        with self.assertRaisesRegex(
                NotImplementedError,
                r"'dsl\.Condition' is not supported by local pipeline execution\."
        ):
            my_pipeline(x='bar')

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
