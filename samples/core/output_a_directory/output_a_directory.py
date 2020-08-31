import kfp
from kfp.components import create_component_from_func, InputPath, OutputPath

@create_component_from_func
def produce_dir_with_files_op(output_dir_path: OutputPath('Directory'), num_files: int = 10):
    import os
    os.makedirs(output_dir_path, exist_ok=True)
    for i in range(num_files):
        file_path = os.path.join(output_dir_path, str(i) + '.txt')
        with open(file_path, 'w') as f:
            f.write(str(i))


@create_component_from_func
def list_dir_files_op(input_dir_path: InputPath('Directory')):
    import os
    dir_items = os.listdir(input_dir_path)
    for dir_item in dir_items:
        print(dir_item)


def dir_pipeline():
    produce_dir_task = produce_dir_with_files_op(num_files=15)
    list_dir_files_op(input_dir=produce_dir_task.output)


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(dir_pipeline, arguments={})
