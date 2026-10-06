# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
import unittest
from unittest import mock

import live_pagination as fixture


def response(values, token=''):
    return {
        'experiments': [{
            'experiment_id': value
        } for value in values],
        'next_page_token': token
    }


class PaginationTest(unittest.TestCase):

    def test_alternating_readers_and_exact_sequence(self):
        with mock.patch.object(
                fixture,
                'page',
                side_effect=[
                    response(['a'], 'one'),
                    response(['b'], 'two'),
                    response(['c'])
                ]) as page:
            result = fixture.walk([8889, 8888], 'criteria', ['a', 'b', 'c'])
        self.assertEqual(result['ports'], [8889, 8888, 8889])
        self.assertEqual(page.call_args_list, [
            mock.call(8889, 'criteria', '', True),
            mock.call(8888, 'criteria', 'one', True),
            mock.call(8889, 'criteria', 'two', True)
        ])

    def test_saved_page_is_not_refetched(self):
        with mock.patch.object(
                fixture, 'page', return_value=response(['b'])) as page:
            fixture.walk([8888],
                         'criteria', ['a', 'b'],
                         response(['a'], 'old-token'),
                         repeat=False)
        page.assert_called_once_with(8888, 'criteria', 'old-token', False)

    def test_duplicate_missing_extra_and_reordered_rows_fail(self):
        for values in (['a', 'a'], ['a'], ['a', 'b', 'c'], ['b', 'a']):
            with self.subTest(values=values), mock.patch.object(
                    fixture, 'page', return_value=response(values)):
                with self.assertRaises(ValueError):
                    fixture.walk([8888], 'criteria', ['a', 'b'])

    def test_saved_source_trace_is_explicit(self):
        result = fixture.walk([8888], 'criteria', ['a'], response(['a']))
        self.assertEqual(result['ports'], ['saved_source_page'])

    def test_source_image_pin_is_required(self):
        with self.assertRaisesRegex(ValueError, 'pinned'):
            fixture.source(None, 'example/api:latest')

    def test_token_cycle_fails(self):
        with mock.patch.object(
                fixture, 'page', return_value=response(['a'], 'same')):
            with self.assertRaisesRegex(ValueError, 'repeated page token'):
                fixture.walk([8888], 'criteria', ['a'])

    def test_page_budget(self):
        with mock.patch.object(
                fixture,
                'page',
                side_effect=[response([str(i)], str(i)) for i in range(20)]):
            with self.assertRaisesRegex(ValueError, 'page budget'):
                fixture.walk([8888], 'criteria', [])


if __name__ == '__main__':
    unittest.main()
