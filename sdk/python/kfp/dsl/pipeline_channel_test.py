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
"""Tests for kfp.dsl.pipeline_channel."""

from typing import List
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Output
from kfp.dsl import pipeline_channel


class PipelineChannelTest(parameterized.TestCase):

    def test_instantiate_pipline_channel(self):
        with self.assertRaisesRegex(
                TypeError, "Can't instantiate abstract class PipelineChannel"):
            p = pipeline_channel.PipelineChannel(
                name='channel',
                channel_type='String',
            )

    def test_invalid_name(self):
        with self.assertRaisesRegex(
                ValueError,
                'Only letters, numbers, spaces, "_", and "-" are allowed in the '
                'name. Must begin with a letter. Got name: 123_abc'):
            p = pipeline_channel.create_pipeline_channel(
                name='123_abc',
                channel_type='String',
            )

    def test_task_name_and_value_both_set(self):
        with self.assertRaisesRegex(ValueError,
                                    'task_name and value cannot be both set.'):
            p = pipeline_channel.create_pipeline_channel(
                name='abc',
                channel_type='Integer',
                task_name='task1',
                value=123,
            )

    def test_invalid_type(self):
        with self.assertRaisesRegex(TypeError,
                                    'Artifact is not a parameter type.'):
            p = pipeline_channel.PipelineParameterChannel(
                name='channel1',
                channel_type='Artifact',
            )

        with self.assertRaisesRegex(TypeError,
                                    'String is not an artifact type.'):
            p = pipeline_channel.PipelineArtifactChannel(
                name='channel1',
                channel_type='String',
                task_name='task1',
                is_artifact_list=False,
            )

    @parameterized.parameters(
        {
            'pipeline_channel':
                pipeline_channel.create_pipeline_channel(
                    name='channel1',
                    task_name='task1',
                    channel_type='String',
                ),
            'str_repr':
                '{{channel:task=task1;name=channel1;type=String;}}',
        },
        {
            'pipeline_channel':
                pipeline_channel.create_pipeline_channel(
                    name='channel2',
                    channel_type='Integer',
                ),
            'str_repr':
                '{{channel:task=;name=channel2;type=Integer;}}',
        },
        {
            'pipeline_channel':
                pipeline_channel.create_pipeline_channel(
                    name='channel3',
                    channel_type={'type_a': {
                        'property_b': 'c'
                    }},
                    task_name='task3',
                ),
            'str_repr':
                '{{channel:task=task3;name=channel3;type={"type_a": {"property_b": "c"}};}}',
        },
        {
            'pipeline_channel':
                pipeline_channel.create_pipeline_channel(
                    name='channel4',
                    channel_type='Float',
                    value=1.23,
                ),
            'str_repr':
                '{{channel:task=;name=channel4;type=Float;}}',
        },
        {
            'pipeline_channel':
                pipeline_channel.create_pipeline_channel(
                    name='channel5',
                    channel_type='system.Artifact@0.0.1',
                    task_name='task5',
                ),
            'str_repr':
                '{{channel:task=task5;name=channel5;type=system.Artifact@0.0.1;}}',
        },
    )
    def test_str_repr(self, pipeline_channel, str_repr):
        self.assertEqual(str_repr, str(pipeline_channel))

    def test_extract_pipeline_channels(self):
        p1 = pipeline_channel.create_pipeline_channel(
            name='channel1',
            channel_type='String',
            value='abc',
        )
        p2 = pipeline_channel.create_pipeline_channel(
            name='channel2',
            channel_type='customized_type_b',
            task_name='task2',
        )
        p3 = pipeline_channel.create_pipeline_channel(
            name='channel3',
            channel_type={'customized_type_c': {
                'property_c': 'value_c'
            }},
            task_name='task3',
        )
        stuff_chars = ' between '
        payload = str(p1) + stuff_chars + str(p2) + stuff_chars + str(p3)
        params = pipeline_channel.extract_pipeline_channels_from_string(payload)
        self.assertListEqual([p1, p2, p3], params)

        # Expecting the extract_pipelineparam_from_any to dedup pipeline channels
        # among all the payloads.
        payload = [
            str(p1) + stuff_chars + str(p2),
            str(p2) + stuff_chars + str(p3)
        ]
        params = pipeline_channel.extract_pipeline_channels_from_any(payload)
        self.assertListEqual([p1, p2, p3], params)


@dsl.component
def string_comp() -> str:
    return 'text'


@dsl.component
def list_comp() -> List[str]:
    return ['text']


@dsl.component
def roll_three_sided_die() -> str:
    import random
    val = random.randint(0, 2)

    if val == 0:
        return 'heads'
    elif val == 1:
        return 'tails'
    else:
        return 'draw'


@dsl.component
def print_and_return(text: str) -> str:
    print(text)
    return text


class TestCanAccessTask(unittest.TestCase):

    def test(self):

        @dsl.pipeline
        def my_pipeline():
            op1 = string_comp()
            self.assertEqual(op1.output.task, op1)


class TestOneOfAndCollectedNotComposable(unittest.TestCase):

    def test_collected_in_oneof(self):
        with self.assertRaisesRegex(
                ValueError,
                'dsl.Collected cannot be used inside of dsl.OneOf.'):

            @dsl.pipeline
            def my_pipeline(x: str):
                with dsl.If(x == 'foo'):
                    t1 = list_comp()
                with dsl.Else():
                    with dsl.ParallelFor([1, 2, 3]):
                        t2 = string_comp()
                    collected = dsl.Collected(t2.output)
                # test cases doesn't return or pass to task to ensure validation is in the OneOf
                dsl.OneOf(t1.output, collected)

    def test_oneof_in_collected(self):
        with self.assertRaisesRegex(
                ValueError,
                'dsl.OneOf cannot be used inside of dsl.Collected.'):

            @dsl.pipeline
            def my_pipeline(x: str):
                with dsl.ParallelFor([1, 2, 3]):
                    with dsl.If(x == 'foo'):
                        t1 = string_comp()
                    with dsl.Else():
                        t2 = string_comp()
                    oneof = dsl.OneOf(t1.output, t2.output)
                # test cases doesn't return or pass to task to ensure validation is in the Collected constructor
                dsl.Collected(oneof)


class TestOneOfRequiresSameType(unittest.TestCase):

    def test_same_parameter_type(self):

        @dsl.pipeline
        def my_pipeline(x: str) -> str:
            with dsl.If(x == 'foo'):
                t1 = string_comp()
            with dsl.Else():
                t2 = string_comp()
            return dsl.OneOf(t1.output, t2.output)

        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-condition-branches-1']
            .output_definitions.parameters[
                'pipelinechannel--condition-branches-1-oneof-1'].parameter_type,
            3)

    def test_different_parameter_types(self):

        with self.assertRaisesRegex(
                TypeError,
                r'Task outputs passed to dsl\.OneOf must be the same type. Got two channels with different types: String at index 0 and typing\.List\[str\] at index 1\.'
        ):

            @dsl.pipeline
            def my_pipeline(x: str) -> str:
                with dsl.If(x == 'foo'):
                    t1 = string_comp()
                with dsl.Else():
                    t2 = list_comp()
                return dsl.OneOf(t1.output, t2.output)

    def test_same_artifact_type(self):

        @dsl.component
        def artifact_comp(out: Output[Artifact]):
            with open(out.path, 'w') as f:
                f.write('foo')

        @dsl.pipeline
        def my_pipeline(x: str) -> Artifact:
            with dsl.If(x == 'foo'):
                t1 = artifact_comp()
            with dsl.Else():
                t2 = artifact_comp()
            return dsl.OneOf(t1.outputs['out'], t2.outputs['out'])

        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-condition-branches-1']
            .output_definitions
            .artifacts['pipelinechannel--condition-branches-1-oneof-1']
            .artifact_type.schema_title,
            'system.Artifact',
        )
        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-condition-branches-1']
            .output_definitions
            .artifacts['pipelinechannel--condition-branches-1-oneof-1']
            .artifact_type.schema_version,
            '0.0.1',
        )

    def test_different_artifact_type(self):

        @dsl.component
        def artifact_comp_one(out: Output[Artifact]):
            with open(out.path, 'w') as f:
                f.write('foo')

        @dsl.component
        def artifact_comp_two(out: Output[Dataset]):
            with open(out.path, 'w') as f:
                f.write('foo')

        with self.assertRaisesRegex(
                TypeError,
                r'Task outputs passed to dsl\.OneOf must be the same type. Got two channels with different types: system.Artifact@0.0.1 at index 0 and system.Dataset@0.0.1 at index 1\.'
        ):

            @dsl.pipeline
            def my_pipeline(x: str) -> Artifact:
                with dsl.If(x == 'foo'):
                    t1 = artifact_comp_one()
                with dsl.Else():
                    t2 = artifact_comp_two()
                return dsl.OneOf(t1.outputs['out'], t2.outputs['out'])

    def test_different_artifact_type_due_to_list(self):
        # if we ever support list of artifact outputs from components, this test will fail, which is good because it needs to be changed

        with self.assertRaisesRegex(
                ValueError,
                r"Output lists of artifacts are only supported for pipelines\. Got output list of artifacts for output parameter 'out' of component 'artifact-comp-two'\."
        ):

            @dsl.component
            def artifact_comp_one(out: Output[Artifact]):
                with open(out.path, 'w') as f:
                    f.write('foo')

            @dsl.component
            def artifact_comp_two(out: Output[List[Artifact]]):
                with open(out.path, 'w') as f:
                    f.write('foo')

            @dsl.pipeline
            def my_pipeline(x: str) -> Artifact:
                with dsl.If(x == 'foo'):
                    t1 = artifact_comp_one()
                with dsl.Else():
                    t2 = artifact_comp_two()
                return dsl.OneOf(t1.outputs['out'], t2.outputs['out'])

    def test_parameters_mixed_with_artifacts(self):

        @dsl.component
        def artifact_comp(out: Output[Artifact]):
            with open(out.path, 'w') as f:
                f.write('foo')

        with self.assertRaisesRegex(
                TypeError,
                r'Task outputs passed to dsl\.OneOf must be the same type\. Found a mix of parameters and artifacts passed to dsl\.OneOf\.'
        ):

            @dsl.pipeline
            def my_pipeline(x: str) -> str:
                with dsl.If(x == 'foo'):
                    t1 = artifact_comp()
                with dsl.Else():
                    t2 = string_comp()
                return dsl.OneOf(t1.output, t2.output)

    def test_no_else_raises(self):
        with self.assertRaisesRegex(
                ValueError,
                r'dsl\.OneOf must include an output from a task in a dsl\.Else group to ensure at least one output is available at runtime\.'
        ):

            @dsl.pipeline
            def roll_die_pipeline():
                flip_coin_task = roll_three_sided_die()
                with dsl.If(flip_coin_task.output == 'heads'):
                    t1 = print_and_return(text='Got heads!')
                with dsl.Elif(flip_coin_task.output == 'tails'):
                    t2 = print_and_return(text='Got tails!')
                print_and_return(text=dsl.OneOf(t1.output, t2.output))


class DictSubvariableTest(unittest.TestCase):
    """Tests for DictSubvariable class."""

    def test_single_level_access(self):
        """Test extracting a single value from a dict parameter."""
        config = pipeline_channel.PipelineParameterChannel(
            name='config',
            channel_type='Dict',
        )
        
        db_host = config['db_host']
        
        self.assertIsInstance(db_host, pipeline_channel.DictSubvariable)
        self.assertEqual(db_host.parent_channel, config)
        self.assertEqual(db_host.key, 'db_host')
        self.assertIn('config-subvar-db_host', db_host.name)

    def test_nested_access(self):
        """Test extracting nested values from a dict parameter."""
        config = pipeline_channel.PipelineParameterChannel(
            name='config',
            channel_type='Dict',
        )
        
        # Test 2-level nesting
        host = config['database']['host']
        
        self.assertIsInstance(host, pipeline_channel.DictSubvariable)
        self.assertEqual(host.key, 'host')
        self.assertIsInstance(host.parent_channel, pipeline_channel.DictSubvariable)
        self.assertEqual(host.parent_channel.key, 'database')
        
        # Test 3-level nesting
        username = config['database']['credentials']['username']
        
        self.assertIsInstance(username, pipeline_channel.DictSubvariable)
        self.assertEqual(username.key, 'username')
        self.assertIsInstance(username.parent_channel, pipeline_channel.DictSubvariable)
        self.assertEqual(username.parent_channel.key, 'credentials')

    def test_parent_must_be_pipeline_parameter_channel(self):
        """Test that parent_channel must be a PipelineParameterChannel."""
        with self.assertRaisesRegex(
                TypeError,
                'parent_channel must be a PipelineParameterChannel'):
            pipeline_channel.DictSubvariable(
                parent_channel='not_a_channel',
                key='test_key'
            )

    def test_subvar_name_in_channel_name(self):
        """Test that the subvar name is included in the channel name."""
        config = pipeline_channel.PipelineParameterChannel(
            name='my_config',
            channel_type='Dict',
        )
        
        value = config['my_key']
        
        self.assertIn('my_config', value.name)
        self.assertIn('subvar', value.name)
        self.assertIn('my_key', value.name)

    def test_chained_access_creates_nested_subvariables(self):
        """Test that chaining creates proper nested structure."""
        config = pipeline_channel.PipelineParameterChannel(
            name='config',
            channel_type='Dict',
        )
        
        # Create chained access
        result = config['level1']['level2']['level3']
        
        # Walk up the chain and verify structure
        self.assertEqual(result.key, 'level3')
        self.assertEqual(result.parent_channel.key, 'level2')
        self.assertEqual(result.parent_channel.parent_channel.key, 'level1')
        self.assertEqual(result.parent_channel.parent_channel.parent_channel.name, 'config')

    def test_getitem_returns_new_dictsubvariable(self):
        """Test that __getitem__ returns a new DictSubvariable instance."""
        config = pipeline_channel.PipelineParameterChannel(
            name='config',
            channel_type='Dict',
        )
        
        first = config['key1']
        second = config['key2']
        
        # Different keys should create different instances
        self.assertIsNot(first, second)
        self.assertEqual(first.key, 'key1')
        self.assertEqual(second.key, 'key2')


if __name__ == '__main__':
    unittest.main()
