# Copyright 2020-2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This sample shows how components can output directories
# Outputting a directory is performed the same way as outputting a file:
# component receives an output path, writes data at that path and the system takes that data and makes it available for the downstream components.
# To output a file, create a new file at the output path location.
# To output a directory, create a new directory at the output path location.
from kfp.components import load_component_from_text
from kfp import compiler, dsl
from kfp.dsl import Input, InputPath, Output, OutputPath, Artifact

# Outputting directories from general command-line based components:

produce_dir_with_files_general_op = load_component_from_text('''
name: Produce directory
inputs:
- {name: num_files, type: Integer}
outputs:
- {name: output_dir}
implementation:
  container:
    image: alpine
    command:
    - sh
    - -ecx
    - |
      num_files="$0"
      output_path="$1"
      mkdir -p "$output_path"
      for i in $(seq "$num_files"); do
        echo "$i" > "$output_path/${i}.txt"
      done
    - {inputValue: num_files}
    - {outputPath: output_dir}
''')

list_dir_files_general_op = load_component_from_text('''
name: List dir files
inputs:
- {name: input_dir}
implementation:
  container:
    image: alpine
    command:
    - ls
    - {inputPath: input_dir}
''')




@dsl.component
def list_dir_files_python_op(input_dir: Input[Artifact],
                             subdir: str = 'texts'):
    import os
    dir_items = os.listdir(os.path.join(input_dir.path, subdir))
    for dir_item in dir_items:
        print(dir_item)


@dsl.component
def produce_dir_with_files_python_op(output_dir: Output[Artifact],
                                     num_files: int = 10,
                                     subdir: str = 'texts'):
    import os
    subdir_path = os.path.join(output_dir.path, subdir)
    os.makedirs(subdir_path, exist_ok=True)
    for i in range(num_files):
        file_path = os.path.join(subdir_path, str(i) + '.txt')
        with open(file_path, 'w') as f:
            f.write(str(i))


@dsl.pipeline(name='dir-pipeline')
def dir_pipeline(subdir: str = 'texts'):
    produce_dir_python_task = produce_dir_with_files_python_op(
        num_files=15,
        subdir=subdir,
    )
    list_dir_files_python_op(
        input_dir=produce_dir_python_task.output,
        subdir=subdir,
    )


if __name__ == '__main__':
  compiler.Compiler().compile(dir_pipeline, __file__ + '.yaml')
