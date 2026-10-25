"""Run in Python 3.10 with ml-metadata==1.14.0 on the database network."""
import sys

import grpc
from ml_metadata import metadata_store
from ml_metadata.proto import metadata_store_pb2

config = metadata_store_pb2.MetadataStoreClientConfig(
    host="127.0.0.1", port=15433)
with grpc.insecure_channel("127.0.0.1:15433") as channel:
    grpc.channel_ready_future(channel).result(timeout=60)
store = metadata_store.MetadataStore(config)
if sys.argv[1] == "seed":
    type_id = store.put_artifact_type(
        metadata_store_pb2.ArtifactType(name="PostgresUpgradeArtifact"))
    store.put_artifacts([
        metadata_store_pb2.Artifact(
            type_id=type_id,
            uri="s3://acceptance/before-upgrade",
            name="preserved-before-upgrade")
    ])
artifacts = store.get_artifacts_by_type("PostgresUpgradeArtifact")
assert len(artifacts) == 1, artifacts
assert artifacts[0].uri == "s3://acceptance/before-upgrade", artifacts
assert artifacts[0].name == "preserved-before-upgrade", artifacts
artifacts[0].custom_properties["last_validation_phase"].string_value = sys.argv[
    1]
store.put_artifacts(artifacts)
assert store.get_artifacts_by_id([
    artifacts[0].id
])[0].custom_properties["last_validation_phase"].string_value == sys.argv[1]
print("PASS MLMD gRPC persisted artifact read/write:", sys.argv[1])
