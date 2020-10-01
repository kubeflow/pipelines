import os
from pathlib import Path
import kfp
import kfp.v2.compiler as compiler
from kfp.v2 import dsl


test_data_dir = os.path.join(Path(__file__).parent, 'test_data')
print('test_data_dir', test_data_dir)
trainer_op = kfp.components.load_component_from_file(os.path.join(test_data_dir, 'trainer_component.yaml'))
serving_op = kfp.components.load_component_from_file(os.path.join(test_data_dir,'serving_component.yaml'))

@dsl.pipeline(
    name='simple-pipeline',
    description='A linear two-step pipeline.')
def simple_pipeline(
    input_gcs = 'gs://test-bucket/pipeline_root',
    optimizer:str = 'sgd',
    epochs:int = 200
):
    trainer = trainer_op(input_location=input_gcs, train_optimizer=optimizer, num_epochs=epochs)
    serving = serving_op(model=trainer.outputs['model_output'], model_cfg=trainer.outputs['model_config'])

if __name__ == '__main__':
    compiler.Compiler().compile(simple_pipeline, __file__ + '.json')
