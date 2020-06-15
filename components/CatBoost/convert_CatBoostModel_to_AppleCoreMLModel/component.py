from kfp.components import InputPath, OutputPath, create_component_from_func

def convert_CatBoostModel_to_AppleCoreMLModel(
    model_path: InputPath('CatBoostModel'),
    converted_model_path: OutputPath('AppleCoreMLModel'),
):
    '''Convert CatBoost model to Apple CoreML format.

    Args:
        model_path: Path of a trained model in binary CatBoost model format.
        converted_model_path: Output path for the converted model.

    Outputs:
        converted_model: Model in Apple CoreML format.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from catboost import CatBoost

    model = CatBoost()
    model.load_model(model_path)
    model.save_model(
        converted_model_path,
        format="coreml",
        # export_parameters={'prediction_type': 'probability'},
        # export_parameters={'prediction_type': 'raw'},
    )


if __name__ == '__main__':
    create_component_from_func(
        convert_CatBoostModel_to_AppleCoreMLModel,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['catboost==0.22']
    )
