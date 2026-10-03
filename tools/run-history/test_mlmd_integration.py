# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Exercise real MLMD protobuf validation and SQLite storage in release CI."""

import os
import unittest

import history
import mlmd
from test_history import graph

try:
    from google.protobuf import json_format
    import grpc
    from ml_metadata import errors
    from ml_metadata import metadata_store
    from ml_metadata.proto import metadata_store_pb2 as pb
    from ml_metadata.proto import metadata_store_service_pb2 as service
    AVAILABLE = True
except ImportError:
    if os.environ.get('KFP_HISTORY_REQUIRE_MLMD') == '1':
        raise
    AVAILABLE = False


class EmbeddedRPC(mlmd.RPC):
    """The pinned MLMD test dependency executes the same requests without a
    server."""

    def __init__(self):
        config = pb.ConnectionConfig()
        config.fake_database.SetInParent()
        self.store = metadata_store.MetadataStore(config)
        self.grpc, self.json_format, self.messages = grpc, json_format, service
        self.timeout = 60
        self.stub = EmbeddedStub(self.store)


class MissingNode(grpc.RpcError if AVAILABLE else Exception):

    def code(self):
        return grpc.StatusCode.NOT_FOUND


class EmbeddedStub:

    def __init__(self, store):
        self.store = store

    def __getattr__(self, method):

        def invoke(request, timeout):
            del timeout
            response = getattr(service, method + 'Response')()
            try:
                # Test-only seam in pinned ml-metadata 1.21.0. The production
                # RPC.call parser, request field names, and response serializer
                # are exercised unmodified against the real MLMD store.
                self.store._call(method, request, response)
            except errors.NotFoundError as error:
                raise MissingNode() from error
            return response

        return invoke


@unittest.skipUnless(AVAILABLE,
                     'optional real MLMD dependency is not installed')
class MLMDIntegrationTest(unittest.TestCase):

    def test_graph_roundtrip_and_repeat_against_real_mlmd(self):
        rpc = EmbeddedRPC()
        metadata = mlmd.Metadata(rpc)
        original = graph()
        mapping = metadata.import_graph(original, 'cluster-a')
        repeated = metadata.import_graph(original, 'cluster-a')
        self.assertEqual(mapping, repeated)
        response = rpc.call(
            'GetContextByTypeAndName',
            type_name='system.PipelineRun',
            context_name='run-a')
        self.assertEqual(response['context']['id'], mapping['contexts']['2'])
        response = rpc.call(
            'GetExecutionsByID', execution_ids=[mapping['executions']['11']])
        cached = response['executions'][0]
        self.assertEqual(
            cached['custom_properties']['cached_execution_id']['string_value'],
            mapping['executions']['10'])
        response = rpc.call(
            'GetEventsByExecutionIDs',
            execution_ids=[mapping['executions']['10']])
        self.assertEqual(len(response['events']), 1)
        self.assertEqual(response['events'][0]['artifact_id'],
                         mapping['artifacts']['20'])
        # Source timestamps are provenance; MLMD owns native creation timestamps.
        execution = rpc.call(
            'GetExecutionsByID',
            execution_ids=[mapping['executions']['10']])['executions'][0]
        self.assertIn(
            '1000', execution['custom_properties']['kfp_history_original']
            ['string_value'])

    def test_existing_unrelated_run_context_is_rejected(self):
        rpc = EmbeddedRPC()
        result = rpc.call(
            'PutContextType', context_type={'name': 'system.PipelineRun'})
        rpc.call(
            'PutContexts',
            contexts=[{
                'type_id': result['type_id'],
                'name': 'run-a'
            }])
        with self.assertRaises(history.HistoryError):
            mlmd.Metadata(rpc).import_graph(graph(), 'cluster-a')


if __name__ == '__main__':
    unittest.main()
