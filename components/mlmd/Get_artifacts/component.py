from kfp.components import create_component_from_func

def get_artifacts_from_mlmd(
    type_name: str = None,
    context_id: int = None,
    context_name: str = None,
) -> list:
    """Gets artifacts from MLMD

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

    if context_name:
        all_contexts = mlmd_store.get_contexts()
        contexts = [
            context
            for context in all_contexts
            if context.name == context_name
        ]
        if len(contexts) == 0:
            raise ValueError(f'Context "{context_name}" was not found.')
        if len(contexts) == 2:
            raise ValueError(f'Found multiple contexts with name "{context_name}": {contexts}.')
        context = contexts[0]
        context_id = context.id

    if context_id:
        artifacts = mlmd_store.get_artifacts_by_context(context_id=context_id)
        if type_name:
            all_artifact_types = mlmd_store.get_artifact_types()
            artifact_types = [
                artifact_type
                for artifact_type in all_artifact_types
                if artifact_type.name == type_name
            ]
            if len(artifact_types) == 0:
                raise ValueError(f'Artifact type "{type_name}" was not found.')
            if len(artifact_types) == 2:
                raise ValueError(f'Found multiple artifact types with name "{type_name}": {artifact_types}.')
            artifact_type = artifact_types[0]
            artifacts = [
                artifact
                for artifact in artifacts
                if artifact.type_id == artifact_type.id
            ]
    elif type_name:
        artifacts = mlmd_store.get_artifacts_by_type(type_name=type_name)
    else:
        artifacts = mlmd_store.get_artifacts()

    artifact_dicts = [MessageToDict(artifact) for artifact in artifacts]
    return artifact_dicts


if __name__ == '__main__':
    get_artifacts_from_mlmd_op = create_component_from_func(
        get_artifacts_from_mlmd,
        packages_to_install=['ml-metadata==0.25.0'],
        output_component_file='component.yaml',
    )
