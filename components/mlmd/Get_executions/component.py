import kfp

from kfp.components import create_component_from_func

def get_executions_from_mlmd(
    type_name: str = None,
    context_id: int = None,
    context_name: str = None,
) -> list:
    """Gets executions from MLMD

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
        executions = mlmd_store.get_executions_by_context(context_id=context_id)
        if type_name:
            all_execution_types = mlmd_store.get_execution_types()
            execution_types = [
                execution_type
                for execution_type in all_execution_types
                if execution_type.name == type_name
            ]
            if len(execution_types) == 0:
                raise ValueError(f'Execution type "{type_name}" was not found.')
            if len(execution_types) == 2:
                raise ValueError(f'Found multiple execution types with name "{type_name}": {execution_types}.')
            execution_type = execution_types[0]
            executions = [
                execution
                for execution in executions
                if execution.type_id == execution_type.id
            ]
    elif type_name:
        executions = mlmd_store.get_executions_by_type(type_name=type_name)
    else:
        executions = mlmd_store.get_executions()
    execution_dicts = [MessageToDict(execution) for execution in executions]
    return execution_dicts


if __name__ == '__main__':
    get_executions_from_mlmd_op = create_component_from_func(
        get_executions_from_mlmd,
        packages_to_install=['ml-metadata==0.25.0'],
        output_component_file='component.yaml',
    )
