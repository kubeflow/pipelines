from kfp.components import InputPath, create_component_from_func, OutputPath


def validate_csv_using_greatexpectations(
    csv_path: InputPath(),
    expectation_suite_path: InputPath(),
    data_doc_path: OutputPath(),
):
    """Validate a CSV dataset against a Great Expectations suite and create Data Doc (a validation report).
    This component fails if validation is not successful.

    Annotations:
        authors: Yaroslav Beshta <ybeshta@provectus.com>, Anton Kiselev <akiselev@provectus.com>

    Args:
        csv_path: Path to the CSV file with the dataset.
        expectation_suite_path: Path to Great Expectations expectation suite (in JSON format).
    """
    import json
    import os
    import sys

    import great_expectations as ge
    from great_expectations.render import DefaultJinjaPageView
    from great_expectations.render.renderer import ValidationResultsPageRenderer

    with open(expectation_suite_path, 'r') as json_file:
        expectation_suite = json.load(json_file)
    df = ge.read_csv(csv_path, expectation_suite=expectation_suite)
    result = df.validate()

    document_model = ValidationResultsPageRenderer().render(result)
    os.makedirs(os.path.dirname(data_doc_path), exist_ok=True)
    with open(data_doc_path, 'w') as writer:
        writer.write(DefaultJinjaPageView().render(document_model))

    print(f'Saved: {data_doc_path}')

    if not result.success:
        sys.exit(1)


if __name__ == '__main__':
    validate_csv_using_greatexpectations_op = create_component_from_func(
        validate_csv_using_greatexpectations,
        output_component_file='component.yaml',
        base_image='python:3.8',
        packages_to_install=['great-expectations==0.13.11']
    )
