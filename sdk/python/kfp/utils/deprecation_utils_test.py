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

from kfp.utils import deprecation_utils


@deprecation_utils._alias_deprecated_params({'new': 'old'})
def my_func(new: str, **kwargs):
    return new


class TestAliasDeprecatedParams(unittest.TestCase):

    def test_new_param(self):
        expected = 'new_param'
        actual = my_func(new='new_param')
        self.assertEqual(expected, actual)

    def test_old_param(self):
        expected = 'old_param'
        with self.assertWarnsRegex(
                DeprecationWarning,
                'old parameter is deprecated, use new instead.'):
            actual = my_func(old=expected)
        self.assertEqual(expected, actual)

    def test_positional_param(self):
        expected = 'positional'
        actual = my_func(expected)
        self.assertEqual(expected, actual)


if __name__ == "__main__":
    unittest.main()