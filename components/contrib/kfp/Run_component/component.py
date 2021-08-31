from typing import NamedTuple


def run_component_or_pipeline(
    component_url: 'Url',
    arguments: dict,
    endpoint: str = None,
    wait_timeout_seconds: float = None,
) -> NamedTuple('Outputs', [
    ('run_id', str),
    ('run_object', 'JsonObject'),  # kfp.ApiRunDetails
]):
    import json
    import os
    import kfp
    from kfp_server_api import ApiClient
    print('Loading component...')
    op = kfp.components.load_component_from_url(component_url)
    print('Loading component done.')
    print('Submitting run...')
    if not endpoint:
        endpoint = 'http://' + os.environ['ML_PIPELINE_SERVICE_HOST'] + ':' + os.environ['ML_PIPELINE_SERVICE_PORT']
    create_run_result = kfp.Client(host=endpoint).create_run_from_pipeline_func(op, arguments=arguments)
    run_id = str(create_run_result.run_id)
    print('Submitted run: ' + run_id)
    run_url = f'{endpoint.rstrip("/")}/#/runs/details/{run_id}'
    print(run_url)
    print('Waiting for the run to finish...')
    run_object = create_run_result.wait_for_run_completion(wait_timeout_seconds)
    print('Run has finished.')
    # sanitize_for_serialization uses correct field names and properly converts datetime values
    run_dict = ApiClient().sanitize_for_serialization(run_object)
    return (
        run_id,
        json.dumps(run_dict, indent=4),
    )


if __name__ == '__main__':
    from kfp.components import create_component_from_func
    run_component_or_pipeline_op = create_component_from_func(
        run_component_or_pipeline,
        base_image='python:3.9',
        packages_to_install=['kfp==1.4.0'],
        output_component_file='component.yaml',
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/kfp/Run_component/component.yaml",
        },
    )
