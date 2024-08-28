# Copyright 2020-2023 The Kubeflow Authors
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
import os

from kfp import client, dsl
from kfp.dsl import Input, Output, Artifact
# Outputting directories from Python-based components:

# In tests, we install a KFP package from the PR under test. Users should not
# normally need to specify `kfp_package_path` in their component definitions.
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
ddef list_dir_files_python(input_dir: Input[Artifact],
                                subdir: str = 'texts'):
    import os
    dir_items = os.listdir(os.path.join(input_dir.path, subdir))
    for dir_item in dir_items:
        print(dir_item)


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
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

@kfp.dsl.pipeline(name='dir-pipeline')
def dir_pipeline(subdir: str = 'texts'):
    produce_dir_python_task = produce_dir_with_files_python_op(
        num_files=15,
        subdir=subdir,
    )
    list_dir_files_python(
        input_dir=produce_dir_python_task.output,
        subdir=subdir,
    )


if __name__ == '__main__':
    kfp_endpoint = None
    kfp_client = client.Client(host=kfp_endpoint)
    run = kfp_client.create_run_from_pipeline_func(
            dir_pipeline,
            arguments={
            },
         )

   
