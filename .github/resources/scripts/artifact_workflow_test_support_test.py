# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest

from artifact_workflow_test_support import evaluate
from artifact_workflow_test_support import render


class ArtifactExpressionTest(unittest.TestCase):

    def test_reference_and_literal_values_are_never_reparsed(self):
        value = "!*.dockerbuild && 'quoted' || ${{ inputs.other }}"
        context = {'inputs': {'pattern': value}}
        self.assertEqual(render('${{ inputs.pattern }}', context), value)
        self.assertEqual(
            render("${{ 'It''s ! unchanged' }}", {}), "It's ! unchanged")

    def test_actions_string_equality_ignores_case(self):
        self.assertTrue(
            evaluate("inputs.result == 'success'",
                     {'inputs': {
                         'result': 'SUCCESS'
                     }}))
        self.assertFalse(evaluate("'FALSE' != 'false'", {}))

    def test_boolean_precedence_and_fallback_preserve_values(self):
        self.assertFalse(evaluate('!true == false && false', {}))
        self.assertTrue(evaluate('true || false && false', {}))
        self.assertFalse(evaluate('(true || false) && false', {}))
        self.assertEqual(
            evaluate("steps.retry.outputs.path || 'fallback'", {}), 'fallback')
        self.assertEqual(evaluate("'false' && 'selected'", {}), 'selected')

    def test_booleans_and_null_render_as_actions_strings(self):
        self.assertEqual(
            render('${{ true }}/${{ false }}/${{ null }}', {}), 'true/false/')

    def test_status_functions_are_explicitly_supplied(self):
        self.assertTrue(
            evaluate('!cancelled() && success()', {}, {
                'cancelled': lambda: False,
                'success': lambda: True
            }))
        with self.assertRaises(AssertionError):
            evaluate('always()', {})

    def test_unsupported_syntax_and_coercion_fail_instead_of_guessing(self):
        for expression in ("true == '1'", 'true + false', 'inputs.x[0]',
                           'fromJSON(inputs.x)', 'true and false', '42',
                           'true false', '(!true', '!'):
            with self.subTest(expression=expression):
                with self.assertRaises(AssertionError):
                    evaluate(expression, {'inputs': {'x': 'value'}})


if __name__ == '__main__':
    unittest.main()
