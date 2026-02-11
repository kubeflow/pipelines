# Copyright 2022 The Kubeflow Authors
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
import unittest

from kfp import client
from kfp.cli.utils import parsing


class MyClass:
    """Class init.

    Args:
        a (str): String a.
        b (str): String b.
    """

    def __init__(self, a: str, b: str):
        """Class init.

        Args:
            a (str): String a.
            b (str): String b.
        """
        self.a = a
        self.b = b

    def method(self, a: str, b: str):
        """My method.

        Args:
            a (str): String a.
            b (str): String b.

        Returns:
            str: Concat str.
        """
        return a + b


class TestGetParamDescr(unittest.TestCase):

    def test_function(self):

        def fn(a: str, b: str):
            """My fn.

            Args:
                a (str): String a.
                b (str): String b.

            Returns:
                str: Concat str.
            """
            return a + b

        a_descr = parsing.get_param_descr(fn, 'a')
        self.assertEqual(a_descr, 'String a.')
        b_descr = parsing.get_param_descr(fn, 'b')
        self.assertEqual(b_descr, 'String b.')

    def test_function_without_type_info(self):

        def fn(a: str, b: str):
            """My fn.

            Args:
                a: String a.
                b: String b.

            Returns:
                str: Concat str.
            """
            return a + b

        a_descr = parsing.get_param_descr(fn, 'a')
        self.assertEqual(a_descr, 'String a.')
        b_descr = parsing.get_param_descr(fn, 'b')
        self.assertEqual(b_descr, 'String b.')

    def test_class(self):

        class_doc_a = parsing.get_param_descr(MyClass, 'a')
        self.assertEqual(class_doc_a, 'String a.')
        class_doc_b = parsing.get_param_descr(MyClass, 'b')
        self.assertEqual(class_doc_b, 'String b.')

        init_a = parsing.get_param_descr(MyClass.__init__, 'a')
        self.assertEqual(init_a, 'String a.')
        init_b = parsing.get_param_descr(MyClass.__init__, 'b')
        self.assertEqual(init_b, 'String b.')

        method_a = parsing.get_param_descr(MyClass.method, 'a')
        self.assertEqual(method_a, 'String a.')
        method_b = parsing.get_param_descr(MyClass.method, 'b')
        self.assertEqual(method_b, 'String b.')

    def test_instance(self):
        instance = MyClass('a', 'b')

        inst_doc_a = parsing.get_param_descr(instance, 'a')
        self.assertEqual(inst_doc_a, 'String a.')
        inst_doc_b = parsing.get_param_descr(instance, 'b')
        self.assertEqual(inst_doc_b, 'String b.')

        init_a = parsing.get_param_descr(instance.__init__, 'a')
        self.assertEqual(init_a, 'String a.')
        init_b = parsing.get_param_descr(instance.__init__, 'b')
        self.assertEqual(init_b, 'String b.')

        method_a = parsing.get_param_descr(instance.method, 'a')
        self.assertEqual(method_a, 'String a.')
        method_b = parsing.get_param_descr(instance.method, 'b')
        self.assertEqual(method_b, 'String b.')

    def test_multiline(self):
        host_descr = parsing.get_param_descr(client.Client, 'host')

        self.assertEqual(
            host_descr,
            "Host name to use to talk to Kubeflow Pipelines. If not set, the in-cluster service DNS name will be used, which only works if the current environment is a pod in the same cluster (such as a Jupyter instance spawned by Kubeflow's JupyterHub). (`More information on connecting. <https://www.kubeflow.org/docs/components/pipelines/user-guides/core-functions/connect-api/>`_)"
        )


class TestParseParameterValue(unittest.TestCase):
    """Tests for parse_parameter_value function."""

    def test_integer_positive(self):
        self.assertEqual(parsing.parse_parameter_value('123'), 123)

    def test_integer_negative(self):
        self.assertEqual(parsing.parse_parameter_value('-456'), -456)

    def test_integer_zero(self):
        self.assertEqual(parsing.parse_parameter_value('0'), 0)

    def test_float_positive(self):
        self.assertEqual(parsing.parse_parameter_value('12.5'), 12.5)

    def test_float_negative(self):
        self.assertEqual(parsing.parse_parameter_value('-3.14'), -3.14)

    def test_float_scientific(self):
        self.assertEqual(parsing.parse_parameter_value('1e10'), 1e10)

    def test_boolean_true_lowercase(self):
        self.assertEqual(parsing.parse_parameter_value('true'), True)

    def test_boolean_false_lowercase(self):
        self.assertEqual(parsing.parse_parameter_value('false'), False)

    def test_boolean_true_capitalized(self):
        self.assertEqual(parsing.parse_parameter_value('True'), True)

    def test_boolean_false_capitalized(self):
        self.assertEqual(parsing.parse_parameter_value('False'), False)

    def test_list_integers(self):
        self.assertEqual(parsing.parse_parameter_value('[1, 2, 3]'), [1, 2, 3])

    def test_list_strings(self):
        self.assertEqual(
            parsing.parse_parameter_value('["a", "b", "c"]'), ['a', 'b', 'c'])

    def test_list_empty(self):
        self.assertEqual(parsing.parse_parameter_value('[]'), [])

    def test_dict_simple(self):
        self.assertEqual(
            parsing.parse_parameter_value('{"key": "value"}'), {'key': 'value'})

    def test_dict_nested(self):
        self.assertEqual(
            parsing.parse_parameter_value('{"a": {"b": 1}}'), {'a': {
                'b': 1
            }})

    def test_dict_empty(self):
        self.assertEqual(parsing.parse_parameter_value('{}'), {})

    def test_null_value(self):
        self.assertIsNone(parsing.parse_parameter_value('null'))

    def test_string_simple(self):
        self.assertEqual(parsing.parse_parameter_value('hello'), 'hello')

    def test_string_with_spaces(self):
        self.assertEqual(
            parsing.parse_parameter_value('hello world'), 'hello world')

    def test_json_quoted_string_preserves_value(self):
        self.assertEqual(parsing.parse_parameter_value('"007"'), '007')
        self.assertEqual(parsing.parse_parameter_value('"hello"'), 'hello')

    def test_string_that_looks_like_number(self):
        self.assertEqual(parsing.parse_parameter_value('007'), 7)
        self.assertEqual(
            parsing.parse_parameter_value('+1-555-1234'), '+1-555-1234')

    def test_string_preserves_value(self):
        self.assertEqual(
            parsing.parse_parameter_value('some-pipeline-name'),
            'some-pipeline-name')


if __name__ == '__main__':
    unittest.main()
