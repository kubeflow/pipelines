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
            "Host name to use to talk to Kubeflow Pipelines. If not set, the in-cluster service DNS name will be used, which only works if the current environment is a pod in the same cluster (such as a Jupyter instance spawned by Kubeflow's JupyterHub). (`More information on connecting. <https://www.kubeflow.org/docs/components/pipelines/sdk/connect-api/>`_)"
        )


if __name__ == '__main__':
    unittest.main()
