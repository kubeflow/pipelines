from pathlib import Path

import kfp.dsl
from kfp.components import ComponentStore, create_component_from_func, OutputPath, load_component_from_file


store = ComponentStore.default_store
chicago_taxi_dataset_op = store.load_component('datasets/Chicago_Taxi_Trips')


def save_text_as_file(text: str,
                      output_path: OutputPath('Text')):
    with open(output_path, 'w') as file:
        file.write(text)


save_text_as_file_op = create_component_from_func(
    func=save_text_as_file,
    base_image='python:3.7',
)

CURRENT_FOLDER = Path(__file__).parent
with open(CURRENT_FOLDER / 'expectation_suite.json') as file:
    expectation_suite = file.read()

validate_csv_op = load_component_from_file(
    str(CURRENT_FOLDER.parent / 'CSV' / 'component.yaml')
)


@kfp.dsl.pipeline(name='Great Expectations')
def great_expectations_sample_pipeline():
    features = ['trip_seconds', 'trip_miles', 'pickup_community_area', 'dropoff_community_area',
                'fare', 'tolls', 'extras', 'trip_total']

    csv_path = chicago_taxi_dataset_op(
        select=','.join(features),
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        limit=1000,
    ).output

    expectation_suite_path = save_text_as_file_op(text=expectation_suite).output

    validate_csv_op(csv=csv_path,
                    expectation_suite=expectation_suite_path)


if __name__ == '__main__':
    kfp_endpoint = None

    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(great_expectations_sample_pipeline, arguments={})
