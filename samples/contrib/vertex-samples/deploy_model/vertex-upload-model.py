from kfp.v2 import dsl

@dsl.component(base_image='python:3.8',packages_to_install=['google-cloud-aiplatform==1.36.0'])
def vertex_deploy_model_to_endpoint(
    model_id: str,           # '/projects/{PROJECT_ID}/locations/{region}/models/{MODEL_ID}'
    endpoint_id: str,        # '/projects/{PROJECT_ID}/locations/{region}/endpoints/{ENDPOINT_ID}'
    machine_type: str,
    min_replica_count: int,
    max_replica_count: int,
):
    import json
    from google.cloud import aiplatform

    model = aiplatform.Model(model_id)
    endpoint = aiplatform.Endpoint(endpoint_id)

    endpoint = model.deploy(
        endpoint=endpoint,
        machine_type=machine_type,
        min_replica_count=min_replica_count,
        max_replica_count=max_replica_count,
    )


@dsl.pipeline(name='upload-model')
def pipeline_upload_model():
    vertex_deploy_model_to_endpoint(
        'projects/123456789012/locations/us-west1/models/9876543210987654321',
        'projects/123456789012/locations/us-west1/endpoints/0123456789876543210', 
        'n1-standard-2', 1, 1)


if __name__ == "__main__":
    from kfp.v2 import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_upload_model,
        package_path='vertex_upload_model.json')