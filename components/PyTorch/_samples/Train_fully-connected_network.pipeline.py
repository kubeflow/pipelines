from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/6162d55998b176b50267d351241100bb0ee715bc/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')
download_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/240543e483076ae718f82c6f280441daa2f041fd/components/web/Download/component.yaml')

create_fully_connected_pytorch_network_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/4e1facea1a270535b515a9e8cc59422d1ad76a9e/components/PyTorch/Create_fully_connected_network/component.yaml')
train_pytorch_model_from_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/603342c4b88fe2d69ff07682f702cd3601e883bb/components/PyTorch/Train_PyTorch_model/from_CSV/component.yaml')
convert_to_onnx_from_pytorch_script_module_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e011e4affa85542ef2b24d63fdac27f8d939bbee/components/PyTorch/Convert_to_OnnxModel_from_PyTorchScriptModule/component.yaml')
create_pytorch_model_archive_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/abc180be2b2b5538d19eb87124684629ec45e620/components/PyTorch/Create_PyTorch_Model_Archive/component.yaml')


def pytorch_pipeline():
    feature_columns = ['trip_seconds', 'trip_miles', 'pickup_community_area', 'dropoff_community_area', 'fare', 'tolls', 'extras']  # Excluded 'trip_total'
    label_column = 'tips'
    network = create_fully_connected_pytorch_network_op(
        layer_sizes=[len(feature_columns), 100, 10, 1],
        activation_name='elu',
    ).output

    training_data = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select=','.join([label_column] + feature_columns),
        limit=10000,
    ).output

    training_data = pandas_transform_csv_op(
        table=training_data,
        transform_code='''df = df.fillna({'tolls': 0.0, 'extras': 0.0}); df = df.dropna(axis='index')''',
    ).output

    trained_model = train_pytorch_model_from_csv_op(
        model=network,
        training_data=training_data,
        label_column_name=label_column,
        loss_function_name='mse_loss',
        # Optional:
        batch_size=32,
        number_of_epochs=2,
        random_seed=0,
        learning_rate=0.1,
        optimizer_name='Adadelta',
        optimizer_parameters={},
    ).outputs['trained_model']

    convert_to_onnx_from_pytorch_script_module_op(
        model=trained_model,
        list_of_input_shapes=[[len(feature_columns)]],
    )

    # TODO: Use a real working regression handler here. See https://github.com/pytorch/serve/issues/987
    serving_handler = download_op('https://raw.githubusercontent.com/pytorch/serve/5c03e711a401387a1d42fc01072fcc38b4995b66/ts/torch_handler/base_handler.py').output
    
    model_archive = create_pytorch_model_archive_op(
        model=trained_model,
        handler=serving_handler,
        # model_name="model",  # Optional
        # model_version="1.0",  # Optional
    ).output

if __name__ == '__main__':
    import kfp
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
        pytorch_pipeline,
        arguments={},
    )
