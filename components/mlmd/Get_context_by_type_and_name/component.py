from typing import NamedTuple
from kfp.components import create_component_from_func

def get_context_by_type_and_name_from_mlmd(
    type_name: str,
    context_name: str,
) -> NamedTuple('Outputs', [
    ('context', dict),
    ('context_id', int),
]):
    """Gets contexts from MLMD

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

    context = mlmd_store.get_context_by_type_and_name(
        type_name=type_name,
        context_name=context_name,
    )

    context_dict = MessageToDict(context)
    return (context_dict, context.id)


if __name__ == '__main__':
    get_context_by_type_and_name_from_mlmd_op = create_component_from_func(
        get_context_by_type_and_name_from_mlmd,
        packages_to_install=['ml-metadata==0.25.0'],
        output_component_file='component.yaml',
    )
