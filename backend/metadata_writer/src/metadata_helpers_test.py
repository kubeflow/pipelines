# Copyright 2026 The Kubeflow Authors
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

import os
import shutil
import tempfile
import unittest
from unittest import mock
from unittest.mock import MagicMock

import metadata_helpers
from ml_metadata.errors import AlreadyExistsError
from ml_metadata.errors import NotFoundError
from ml_metadata.errors import ResourceExhaustedError
from ml_metadata.metadata_store import metadata_store
from ml_metadata.proto import metadata_store_pb2


def _make_context_type(type_id=1, type_name="KfpRun"):
    return metadata_store_pb2.ContextType(id=type_id, name=type_name)


def _make_context(context_id=1, context_name="run-1", type_id=1):
    return metadata_store_pb2.Context(
        id=context_id,
        name=context_name,
        type_id=type_id,
    )


class GetContextByNameTest(unittest.TestCase):

    def setUp(self):
        # clear the lru cache between tests.
        metadata_helpers.get_context_by_name.cache_clear()

    def test_returns_context_when_found(self):
        store = MagicMock()
        ctx = _make_context(context_id=99, context_name="run-1")
        store.get_context_by_type_and_name.return_value = ctx

        result = metadata_helpers.get_context_by_name(
            store, "run-1", type_name="KfpRun")

        self.assertEqual(result.id, 99)
        store.get_context_by_type_and_name.assert_called_once_with(
            type_name="KfpRun", context_name="run-1")

    def test_raises_value_error_when_not_found(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None

        with self.assertRaises(ValueError):
            metadata_helpers.get_context_by_name(
                store, "nonexistent", type_name="KfpRun")

    def test_propagates_transport_error(self):
        store = MagicMock()
        store.get_context_by_type_and_name.side_effect = ResourceExhaustedError(
            "too big")

        with self.assertRaises(ResourceExhaustedError):
            metadata_helpers.get_context_by_name(
                store, "run-1", type_name="KfpRun")


class GetOrCreateContextWithTypeTest(unittest.TestCase):

    def setUp(self):
        # clear the lru cache between tests.
        metadata_helpers.get_context_by_name.cache_clear()

    def test_returns_existing_context_when_found(self):
        store = MagicMock()
        existing = _make_context(context_id=7, context_name="run-1", type_id=1)
        store.get_context_by_type_and_name.return_value = existing

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun")

        self.assertEqual(result.id, 7)
        store.get_contexts.assert_not_called()
        store.put_contexts.assert_not_called()

    def test_creates_context_when_it_does_not_exist(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None  # not found
        store.get_context_type.return_value = _make_context_type()
        store.put_contexts.return_value = [42]

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun")

        self.assertEqual(result.id, 42)
        self.assertEqual(result.name, "run-1")
        store.get_contexts.assert_not_called()
        store.put_contexts.assert_called_once()

    def test_propagates_resource_exhausted_error(self):
        store = MagicMock()
        store.get_context_by_type_and_name.side_effect = ResourceExhaustedError(
            "Received message larger than max (6146499 vs. 4194304)")

        with self.assertRaises(ResourceExhaustedError):
            metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun")

        # the transport error should not trigger the create path
        store.put_contexts.assert_not_called()

    def test_recovers_from_already_exists_error(self):
        store = MagicMock()
        existing = _make_context(context_id=7, context_name="run-1", type_id=1)
        # First lookup misses, create races and fails, recovery lookup finds it.
        store.get_context_by_type_and_name.side_effect = [None, existing]
        store.get_context_type.return_value = _make_context_type()
        store.put_contexts.side_effect = AlreadyExistsError("already exists")

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun")

        self.assertEqual(result.id, 7)
        self.assertEqual(store.get_context_by_type_and_name.call_count, 2)
        store.put_contexts.assert_called_once()

    def test_propagates_context_type_schema_error(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None  # not found
        store.get_context_type.side_effect = NotFoundError("type not found")
        store.put_context_type.side_effect = AlreadyExistsError(
            "incompatible type schema")

        with self.assertRaises(AlreadyExistsError):
            metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun")

        # Check that the type creation failure doesn't trigger the
        # duplicate-context recovery: no context insert is attempted.
        store.put_contexts.assert_not_called()

    def test_propagates_transport_error_from_type_lookup(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None  # not found
        store.get_context_type.side_effect = ResourceExhaustedError("too big")

        with self.assertRaises(ResourceExhaustedError):
            metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun")

        # Check that the transport error is not mistaken for a missing type.
        store.put_context_type.assert_not_called()
        store.put_contexts.assert_not_called()


class GetOrCreateContextWithTypeWithMlMDStoreTest(unittest.TestCase):
    """Exercise the create and recovery paths against a live sqlite MLMD."""

    def setUp(self):
        # Clear the lru cache between tests.
        metadata_helpers.get_context_by_name.cache_clear()
        self._tmp_dir = tempfile.mkdtemp()
        self._db_path = os.path.join(self._tmp_dir, "mlmd.sqlite")

    def tearDown(self):
        shutil.rmtree(self._tmp_dir)

    def _make_store(self):
        return metadata_store.MetadataStore(
            metadata_store_pb2.ConnectionConfig(
                sqlite=metadata_store_pb2.SqliteMetadataSourceConfig(
                    filename_uri=self._db_path)))

    def test_same_name_context_of_different_type_can_be_created(self):
        store = self._make_store()
        other_type_id = store.put_context_type(
            metadata_store_pb2.ContextType(name="OtherType"))
        other_context_id = store.put_contexts(
            [metadata_store_pb2.Context(name="run-1",
                                        type_id=other_type_id)])[0]

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun")

        kfp_run_type_id = store.get_context_type("KfpRun").id
        self.assertEqual(result.name, "run-1")
        self.assertEqual(result.type_id, kfp_run_type_id)
        self.assertNotEqual(result.id, other_context_id)
        # MLMD names are unique per ContextType, so both contexts coexist.
        self.assertIsNotNone(
            store.get_context_by_type_and_name("KfpRun", "run-1"))
        self.assertIsNotNone(
            store.get_context_by_type_and_name("OtherType", "run-1"))
        # The typed lookup returns the original context for the other type.
        existing = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="OtherType")
        self.assertEqual(existing.id, other_context_id)
        self.assertEqual(existing.type_id, other_type_id)

    def test_recovers_from_duplicate_insert_on_real_store(self):
        store = self._make_store()
        run_type_id = store.put_context_type(
            metadata_store_pb2.ContextType(name="KfpRun"))
        run_context_id = store.put_contexts(
            [metadata_store_pb2.Context(name="run-1", type_id=run_type_id)])[0]

        real_lookup = metadata_helpers.get_context_by_name
        lookup_calls = []

        def race_lookup(store, context_name, type_name):
            # Simulate the race: the first lookup misses, but by the time
            # the writer creates the context, another writer landed.
            lookup_calls.append((context_name, type_name))
            if len(lookup_calls) == 1:
                raise ValueError("context not found")
            return real_lookup(store, context_name, type_name)

        with mock.patch.object(
                metadata_helpers, "get_context_by_name",
                side_effect=race_lookup):
            result = metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun")

        self.assertEqual(result.id, run_context_id)
        self.assertEqual(len(lookup_calls), 2)
        self.assertEqual(len(store.get_contexts()), 1)


if __name__ == "__main__":
    unittest.main()
