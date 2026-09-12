import unittest
from unittest import mock

from ml_metadata.proto import metadata_store_pb2

from metadata_helpers import (
    get_or_create_artifact_type,
    get_or_create_execution_type,
    get_or_create_context_type,
    get_or_create_context_with_type,
)


class TestGetOrCreateArtifactType(unittest.TestCase):
    def setUp(self):
        self.store = mock.MagicMock()

    def test_get_path_returns_existing(self):
        existing = metadata_store_pb2.ArtifactType(name="test-type")
        self.store.get_artifact_type.return_value = existing

        result = get_or_create_artifact_type(self.store, "test-type")

        self.store.get_artifact_type.assert_called_once_with(type_name="test-type")
        self.store.put_artifact_type.assert_not_called()
        self.assertEqual(result, existing)

    def test_create_path_puts_new_type(self):
        self.store.get_artifact_type.side_effect = ValueError("not found")
        self.store.put_artifact_type.return_value = 42

        result = get_or_create_artifact_type(self.store, "test-type")

        self.store.get_artifact_type.assert_called_once_with(type_name="test-type")
        self.store.put_artifact_type.assert_called_once()
        self.assertEqual(result.id, 42)
        self.assertEqual(result.name, "test-type")

    def test_unrelated_exception_propagates(self):
        self.store.get_artifact_type.side_effect = RuntimeError("something else")

        with self.assertRaises(RuntimeError):
            get_or_create_artifact_type(self.store, "test-type")

        self.store.put_artifact_type.assert_not_called()


class TestGetOrCreateExecutionType(unittest.TestCase):
    def setUp(self):
        self.store = mock.MagicMock()

    def test_get_path_returns_existing(self):
        existing = metadata_store_pb2.ExecutionType(name="test-type")
        self.store.get_execution_type.return_value = existing

        result = get_or_create_execution_type(self.store, "test-type")

        self.store.get_execution_type.assert_called_once_with(type_name="test-type")
        self.store.put_execution_type.assert_not_called()
        self.assertEqual(result, existing)

    def test_create_path_puts_new_type(self):
        self.store.get_execution_type.side_effect = ValueError("not found")
        self.store.put_execution_type.return_value = 42

        result = get_or_create_execution_type(self.store, "test-type")

        self.store.get_execution_type.assert_called_once_with(type_name="test-type")
        self.store.put_execution_type.assert_called_once()
        self.assertEqual(result.id, 42)
        self.assertEqual(result.name, "test-type")

    def test_unrelated_exception_propagates(self):
        self.store.get_execution_type.side_effect = RuntimeError("something else")

        with self.assertRaises(RuntimeError):
            get_or_create_execution_type(self.store, "test-type")

        self.store.put_execution_type.assert_not_called()


class TestGetOrCreateContextType(unittest.TestCase):
    def setUp(self):
        self.store = mock.MagicMock()

    def test_get_path_returns_existing(self):
        existing = metadata_store_pb2.ContextType(name="test-type")
        self.store.get_context_type.return_value = existing

        result = get_or_create_context_type(self.store, "test-type")

        self.store.get_context_type.assert_called_once_with(type_name="test-type")
        self.store.put_context_type.assert_not_called()
        self.assertEqual(result, existing)

    def test_create_path_puts_new_type(self):
        self.store.get_context_type.side_effect = ValueError("not found")
        self.store.put_context_type.return_value = 42

        result = get_or_create_context_type(self.store, "test-type")

        self.store.get_context_type.assert_called_once_with(type_name="test-type")
        self.store.put_context_type.assert_called_once()
        self.assertEqual(result.id, 42)
        self.assertEqual(result.name, "test-type")

    def test_unrelated_exception_propagates(self):
        self.store.get_context_type.side_effect = RuntimeError("something else")

        with self.assertRaises(RuntimeError):
            get_or_create_context_type(self.store, "test-type")

        self.store.put_context_type.assert_not_called()


class TestGetOrCreateContextWithType(unittest.TestCase):
    def setUp(self):
        self.store = mock.MagicMock()

    @mock.patch("metadata_helpers.get_context_by_name")
    def test_get_path_returns_existing_and_verifies_type(self, mock_get_context):
        existing_context = metadata_store_pb2.Context(name="test-context", type_id=1)
        mock_get_context.return_value = existing_context
        existing_context_type = metadata_store_pb2.ContextType(
            name="test-type-name"
        )
        self.store.get_context_types_by_id.return_value = [existing_context_type]

        result = get_or_create_context_with_type(
            self.store,
            context_name="test-context",
            type_name="test-type-name",
        )

        mock_get_context.assert_called_once_with(self.store, "test-context")
        self.store.get_context_types_by_id.assert_called_once_with([1])
        self.assertEqual(result, existing_context)

    @mock.patch("metadata_helpers.create_context_with_type")
    @mock.patch("metadata_helpers.get_context_by_name")
    def test_create_path_calls_create_and_returns_new_context(
        self, mock_get_context, mock_create_context
    ):
        mock_get_context.side_effect = ValueError("not found")
        new_context = metadata_store_pb2.Context(name="test-context", type_id=1)
        mock_create_context.return_value = new_context

        result = get_or_create_context_with_type(
            self.store,
            context_name="test-context",
            type_name="test-type",
            properties={"key": "value"},
            type_properties={"tkey": "tval"},
            custom_properties={"ckey": "cval"},
        )

        mock_get_context.assert_called_once_with(self.store, "test-context")
        mock_create_context.assert_called_once_with(
            store=self.store,
            context_name="test-context",
            type_name="test-type",
            properties={"key": "value"},
            type_properties={"tkey": "tval"},
            custom_properties={"ckey": "cval"},
        )
        self.assertEqual(result, new_context)

    @mock.patch("metadata_helpers.create_context_with_type")
    @mock.patch("metadata_helpers.get_context_by_name")
    def test_unrelated_exception_propagates(
        self, mock_get_context, mock_create_context
    ):
        mock_get_context.side_effect = RuntimeError("something else")

        with self.assertRaises(RuntimeError):
            get_or_create_context_with_type(
                self.store,
                context_name="test-context",
                type_name="test-type",
            )

        mock_get_context.assert_called_once_with(self.store, "test-context")
        mock_create_context.assert_not_called()
