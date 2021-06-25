from typing import NamedTuple

from kfp.components import create_component_from_func, InputPath, OutputPath

def automl_create_tables_dataset_from_csv(
    data_path: InputPath('CSV'),
    target_column_name: str = None,
    column_nullability: dict = {},
    column_types: dict = {},
    gcs_staging_uri: str = None,  # Currently AutoML Tables only supports regional buckets in "us-central1".
    gcp_project_id: str = None,
    gcp_region: str = 'us-central1',  # Currently "us-central1" is the only region supported by AutoML tables.
) -> NamedTuple('Outputs', [
    ('dataset_name', str),
    ('dataset_url', 'URI'),
]):
    '''Creates Google Cloud AutoML Tables Dataset from CSV data.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>

    Args:
        data_path: Data in CSV format that will be imported to the dataset.
        target_column_name: Name of the target column for training.
        column_nullability: Maps column name to boolean specifying whether the column should be marked as nullable.
        column_types: Maps column name to column type. Supported types: FLOAT64, CATEGORY, STRING.
        gcs_staging_uri: URI of the data staging location in Google Cloud Storage. The bucket must have the us-central1 region. If not specified, a new staging bucket will be created.
        gcp_project_id: Google Cloud project ID. If not set, the default one will be used.
        gcp_region: Google Cloud region. AutoML Tables only supports us-central1.
    Returns:
        dataset_name: AutoML dataset name (fully-qualified)
    '''

    import logging
    import random

    import google.auth
    from google.cloud import automl_v1beta1 as automl
    from google.cloud import storage

    logging.getLogger().setLevel(logging.INFO)

    # Validating and inferring the arguments

    if not gcp_project_id:
        _, gcp_project_id = google.auth.default()

    if not gcp_region:
        gcp_region = 'us-central1'
    if gcp_region != 'us-central1':
        logging.warn('AutoML only supports the us-central1 region')

    dataset_display_name = 'Dataset'  # Allowed characters for displayName are ASCII Latin letters A-Z and a-z, an underscore (_), and ASCII digits 0-9

    column_nullability = column_nullability or {}
    for name, nullability in column_nullability.items():
        assert isinstance(name, str)
        assert isinstance(nullability, bool)

    column_types = column_types or {}
    for name, data_type in column_types.items():
        assert isinstance(name, str)
        if not hasattr(automl.TypeCode, data_type):
            supported_types = [type_name for type_name in dir(automl.TypeCode) if type_name[0] != '_']
            raise ValueError(f'Unknow column type "{data_type}". Supported types: {supported_types}')

    # Generating execution ID for data staging
    random_integer = random.SystemRandom().getrandbits(256)
    execution_id = '{:064x}'.format(random_integer)
    logging.info(f'Execution ID: {execution_id}')

    logging.info('Uploading the data to storage')
    # TODO: Split table into < 100MB chunks as required by AutoML Tables
    storage_client = storage.Client()
    if gcs_staging_uri:
        if not gcs_staging_uri.startswith('gs://'):
            raise ValueError(f"Invalid staging storage URI: {gcs_staging_uri}")
        (bucket_name, blob_prefix) = gcs_staging_uri[5:].split('/', 1)
        bucket = storage_client.get_bucket(bucket_name)
    else:
        bucket_name = gcp_project_id + '_staging_' + gcp_region
        try:
            bucket = storage_client.get_bucket(bucket_name)
        except Exception as ex:
            logging.info(f'Creating Storage bucket {bucket_name}')
            bucket = storage_client.create_bucket(
                bucket_or_name=bucket_name,
                project=gcp_project_id,
                location=gcp_region,
            )
            logging.info(f'Created Storage bucket {bucket.name}')
        blob_prefix = 'google.cloud.automl_tmp'

    # AutoML Tables import data requires that "the file name must have a (case-insensitive) '.CSV' file extension"
    training_data_blob_name = blob_prefix.rstrip('/') + '/' + execution_id + '/' + 'training_data.csv'
    training_data_blob_uri = f'gs://{bucket.name}/{training_data_blob_name}'
    training_data_blob = bucket.blob(training_data_blob_name)
    logging.info(f'Uploading training data to {training_data_blob_uri}')
    training_data_blob.upload_from_filename(data_path)

    logging.info(f'Creating AutoML Tables dataset.')
    automl_client = automl.AutoMlClient()

    project_location_path = f'projects/{gcp_project_id}/locations/{gcp_region}'

    dataset = automl.Dataset(
        display_name=dataset_display_name,
        tables_dataset_metadata=automl.TablesDatasetMetadata(),
        # labels={},
    )
    dataset = automl_client.create_dataset(
        dataset=dataset,
        parent=project_location_path,
    )
    dataset_id = dataset.name.split('/')[-1]
    dataset_web_url = f'https://console.cloud.google.com/automl-tables/locations/{gcp_region}/datasets/{dataset_id}'
    logging.info(f'Created dataset {dataset.name}. Link: {dataset_web_url}')

    logging.info(f'Importing data to the dataset: {dataset.name}.')
    import_data_input_config = automl.InputConfig(
        gcs_source=automl.GcsSource(
            input_uris=[training_data_blob_uri],
        )
    )
    import_data_response = automl_client.import_data(
        name=dataset.name,
        input_config=import_data_input_config,
    )
    import_data_response.result()
    dataset = automl_client.get_dataset(
        name=dataset.name,
    )
    logging.info(f'Finished importing data.')

    logging.info('Updating column specs')
    target_column_spec = None
    primary_table_spec_name = dataset.name + '/tableSpecs/' + dataset.tables_dataset_metadata.primary_table_spec_id
    table_specs_list = list(automl_client.list_table_specs(
        parent=dataset.name,
    ))
    for table_spec in table_specs_list:
        table_spec_id = table_spec.name.split('/')[-1]
        column_specs_list = list(automl_client.list_column_specs(
            parent=table_spec.name,
        ))
        is_primary_table = table_spec.name == primary_table_spec_name
        for column_spec in column_specs_list:
            if column_spec.display_name == target_column_name and is_primary_table:
                target_column_spec = column_spec
            column_updated = False
            if column_spec.display_name in column_nullability:
                column_spec.data_type.nullable = column_nullability[column_spec.display_name]
                column_updated = True
            if column_spec.display_name in column_types:
                new_column_type = column_types[column_spec.display_name]
                column_spec.data_type.type_code = getattr(automl.TypeCode, new_column_type)
                column_updated = True
            if column_updated:
                automl_client.update_column_spec(column_spec=column_spec)

    if target_column_name:
        logging.info('Setting target column')
        if not target_column_spec:
            raise ValueError(f'Primary table does not have column "{target_column_name}"')
        target_column_spec_id = target_column_spec.name.split('/')[-1]
        dataset.tables_dataset_metadata.target_column_spec_id = target_column_spec_id
        dataset = automl_client.update_dataset(dataset=dataset)

    return (dataset.name, dataset_web_url)


if __name__ == '__main__':
    automl_create_tables_dataset_from_csv_op = create_component_from_func(
        automl_create_tables_dataset_from_csv,
        base_image='python:3.8',
        packages_to_install=['google-cloud-automl==2.0.0', 'google-cloud-storage==1.31.2', 'google-auth==1.21.3'],
        output_component_file='component.yaml',
    )
