# Copyright 2019 The Kubeflow Authors
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
from unittest.mock import MagicMock

from ml_metadata.proto import metadata_store_pb2
from ml_metadata.errors import AlreadyExistsError

import metadata_helpers


class GetOrCreateContextWithTypeTest(unittest.TestCase):

    def setUp(self):
        # get_context_by_name is decorated with functools.lru_cache, which
        # caches across tests since it's a module-level function. Clear it
        # before every test so tests don't leak state into each other.
        metadata_helpers.get_context_by_name.cache_clear()

    def _make_context_type(self, type_id=1, type_name='KfpRun'):
        return metadata_store_pb2.ContextType(id=type_id, name=type_name)

    def test_creates_context_when_it_does_not_exist(self):
        store = MagicMock()
        store.get_contexts.return_value = []  # no existing contexts
        store.get_context_type.return_value = self._make_context_type()
        store.put_contexts.return_value = [42]

        context = metadata_helpers.get_or_create_context_with_type(
            store=store,
            context_name='run-1',
            type_name='KfpRun',
        )

        self.assertEqual(context.id, 42)
        self.assertEqual(context.name, 'run-1')
        store.put_contexts.assert_called_once()

    def test_recovers_when_context_already_exists(self):
        """
        Simulates the race from issue #13713: get_context_by_name finds
        nothing (e.g. because the lru_cache or in-memory state was reset
        after a restart), so we attempt to create the context, but MLMD
        rejects it because some other event already created it. The
        function should recover by fetching and returning the existing
        context instead of raising.
        """
        store = MagicMock()
        existing_context = metadata_store_pb2.Context(
            id=99,
            name='run-1',
            type_id=1,
        )

        store.get_contexts.side_effect = [[], [existing_context]]
        store.get_context_type.return_value = self._make_context_type()
        store.put_contexts.side_effect = AlreadyExistsError('already exists')
        store.get_context_types_by_id.return_value = [self._make_context_type()]

        context = metadata_helpers.get_or_create_context_with_type(
            store=store,
            context_name='run-1',
            type_name='KfpRun',
        )

        self.assertEqual(context.id, 99)
        self.assertEqual(context.name, 'run-1')
        store.put_contexts.assert_called_once()

    def test_returns_existing_context_when_already_cached(self):
        store = MagicMock()
        existing_context = metadata_store_pb2.Context(
            id=7,
            name='run-2',
            type_id=1,
        )
        store.get_contexts.return_value = [existing_context]
        store.get_context_types_by_id.return_value = [self._make_context_type()]

        context = metadata_helpers.get_or_create_context_with_type(
            store=store,
            context_name='run-2',
            type_name='KfpRun',
        )

        self.assertEqual(context.id, 7)
        store.put_contexts.assert_not_called()


if __name__ == '__main__':
    unittest.main()
