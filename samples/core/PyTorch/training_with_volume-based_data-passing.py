kfp_endpoint=None


from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/1902306d9ea9ffce00c8c1e3920a8efac816bdd6/components/datasets/Chicago_Taxi_Trips/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/6162d55998b176b50267d351241100bb0ee715bc/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')

create_fully_connected_pytorch_network_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/4e1facea1a270535b515a9e8cc59422d1ad76a9e/components/PyTorch/Create_fully_connected_network/component.yaml')
train_pytorch_model_from_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/603342c4b88fe2d69ff07682f702cd3601e883bb/components/PyTorch/Train_PyTorch_model/from_CSV/component.yaml')
convert_to_onnx_from_pytorch_script_module_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e011e4affa85542ef2b24d63fdac27f8d939bbee/components/PyTorch/Convert_to_OnnxModel_from_PyTorchScriptModule/component.yaml')

kubernetes_create_pvc_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/deece918fe2be5ba738d036ae6d9fd58b6e6f097/components/kubernetes/Create_PersistentVolumeClaim/component.yaml')

volume_name_for_data_passing = 'my-pvc'

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
    
    if volume_name_for_data_passing:
        from kfp.dsl import PipelineConf, data_passing_methods
        from kubernetes.client.models import V1Volume, V1PersistentVolumeClaimVolumeSource
        kfp.dsl.get_pipeline_conf().data_passing_method = data_passing_methods.KubernetesVolume(
            volume=V1Volume(
                name='data',
                persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(volume_name_for_data_passing),
            ),
            path_prefix='artifact_data/',
        )

if __name__ == '__main__':
    import kfp

    if volume_name_for_data_passing:
        def create_volume_pipeline():
            kubernetes_create_pvc_op(name=volume_name_for_data_passing)

        create_volume_run = kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(create_volume_pipeline, arguments={})
        create_volume_run.wait_for_run_completion()
    
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
        pytorch_pipeline,
        arguments={},
    )
