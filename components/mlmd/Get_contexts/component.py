from kfp.components import create_component_from_func

def get_contexts_from_mlmd(
    type_name: str = None,
) -> list:
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

    contexts = mlmd_store.get_contexts()
    if type_name:
        all_context_types = mlmd_store.get_context_types()
        context_types = [
            context_type
            for context_type in all_context_types
            if context_type.name == type_name
        ]
        if len(context_types) == 0:
            raise ValueError(f'Context type "{type_name}" was not found.')
        if len(context_types) == 2:
            raise ValueError(f'Found multiple context types with name "{type_name}": {context_types}.')
        context_type = context_types[0]
        contexts = [
            context
            for context in contexts
            if context.type_id == context_type.id
        ]

    context_dicts = [MessageToDict(context) for context in contexts]
    return context_dicts


if __name__ == '__main__':
    get_contexts_from_mlmd_op = create_component_from_func(
        get_contexts_from_mlmd,
        packages_to_install=['ml-metadata==0.25.0'],
        output_component_file='component.yaml',
    )
