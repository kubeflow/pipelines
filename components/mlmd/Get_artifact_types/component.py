from kfp.components import create_component_from_func

def get_artifact_types_from_mlmd() -> list:
    """Gets artifact types from MLMD

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    import os
    from google.protobuf.json_format import MessageToDict
    from ml_metadata.proto import metadata_store_pb2
    from ml_metadata.metadata_store import metadata_store
    metadata_service_host = os.environ.get('METADATA_GRPC_SERVICE_SERVICE_HOST', 'metadata-grpc-service')
    metadata_service_port = int(os.environ.get('METADATA_GRPC_SERVICE_SERVICE_PORT', 8080))
    mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
        host=metadata_service_host,
        port=metadata_service_port,
    )
    mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)

    artifact_types = mlmd_store.get_artifact_types()

    artifact_type_dicts = [MessageToDict(artifact_type) for artifact_type in artifact_types]
    return artifact_type_dicts


if __name__ == '__main__':
    get_artifact_types_from_mlmd_op = create_component_from_func(
        get_artifact_types_from_mlmd,
        packages_to_install=['ml-metadata==0.25.0'],
        output_component_file='component.yaml',
    )
