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

import unittest
from unittest.mock import MagicMock

import metadata_helpers
from ml_metadata.errors import ResourceExhaustedError
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
            store, "run-1", type_name="KfpRun"
        )

        self.assertEqual(result.id, 99)
        store.get_context_by_type_and_name.assert_called_once_with(
            type_name="KfpRun", context_name="run-1"
        )

    def test_raises_value_error_when_not_found(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None

        with self.assertRaises(ValueError):
            metadata_helpers.get_context_by_name(
                store, "nonexistent", type_name="KfpRun"
            )

    def test_propagates_transport_error(self):
        store = MagicMock()
        store.get_context_by_type_and_name.side_effect = ResourceExhaustedError(
            "too big"
        )

        with self.assertRaises(ResourceExhaustedError):
            metadata_helpers.get_context_by_name(store, "run-1", type_name="KfpRun")


class GetOrCreateContextWithTypeTest(unittest.TestCase):
    def setUp(self):
        # clear the lru cache between tests.
        metadata_helpers.get_context_by_name.cache_clear()

    def test_returns_existing_context_when_found(self):
        store = MagicMock()
        existing = _make_context(context_id=7, context_name="run-1", type_id=1)
        store.get_context_by_type_and_name.return_value = existing
        store.get_context_types_by_id.return_value = [_make_context_type()]

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun"
        )

        self.assertEqual(result.id, 7)
        store.get_contexts.assert_not_called()
        store.put_contexts.assert_not_called()

    def test_creates_context_when_it_does_not_exist(self):
        store = MagicMock()
        store.get_context_by_type_and_name.return_value = None  # not found
        store.get_context_type.return_value = _make_context_type()
        store.put_contexts.return_value = [42]

        result = metadata_helpers.get_or_create_context_with_type(
            store=store, context_name="run-1", type_name="KfpRun"
        )

        self.assertEqual(result.id, 42)
        self.assertEqual(result.name, "run-1")
        store.get_contexts.assert_not_called()
        store.put_contexts.assert_called_once()

    def test_propagates_resource_exhausted_error(self):
        store = MagicMock()
        store.get_context_by_type_and_name.side_effect = ResourceExhaustedError(
            "Received message larger than max (6146499 vs. 4194304)"
        )

        with self.assertRaises(ResourceExhaustedError):
            metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun"
            )

        # the transport error should not trigger the create path
        store.put_contexts.assert_not_called()

    def test_raises_runtime_error_when_context_has_wrong_type(self):
        store = MagicMock()
        existing = _make_context(context_id=1, context_name="run-1", type_id=2)
        store.get_context_by_type_and_name.return_value = existing
        wrong_type = _make_context_type(type_id=2, type_name="OtherType")
        store.get_context_types_by_id.return_value = [wrong_type]

        with self.assertRaises(RuntimeError):
            metadata_helpers.get_or_create_context_with_type(
                store=store, context_name="run-1", type_name="KfpRun"
            )


if __name__ == "__main__":
    unittest.main()
