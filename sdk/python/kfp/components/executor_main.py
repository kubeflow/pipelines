# Copyright 2021 The Kubeflow Authors
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
import argparse
import json
import importlib
import os
import sys

from kfp.components import executor as component_executor


def _load_module(module_name: str, module_directory: str):
    """Dynamically imports the Python module with the given name and package
    path.

    E.g., Assuming there is a file called `my_module.py` under
    `/some/directory/my_module`, we can use::

        _load_module('my_module', '/some/directory')

    to effectively `import mymodule`.

    Args:
        module_name: The name of the module.
        package_path: The package under which the specified module resides.
    """
    module_spec = importlib.util.spec_from_file_location(
        name=module_name,
        location=os.path.join(module_directory, f'{module_name}.py'))
    module = importlib.util.module_from_spec(module_spec)
    sys.modules[module_spec.name] = module
    module_spec.loader.exec_module(module)
    return module


def executor_main():
    parser = argparse.ArgumentParser(description='KFP Component Executor.')
    parser.add_argument('--component_module_path',
                        type=str,
                        help='Path to a module containing the KFP component.')
    parser.add_argument('--function_to_execute',
                        type=str,
                        help='The name of the component function in '
                        '--component_module_path file that is to be executed.')
    parser.add_argument(
        '--executor_input',
        type=str,
        help='JSON-serialized ExecutorInput from the orchestrator. '
        'This should contain inputs and placeholders for outputs.')

    args, _ = parser.parse_known_args()

    module_directory = os.path.dirname(args.component_module_path)
    module_name = os.path.basename(args.component_module_path)[:-len('.py')]
    print('Loading KFP component module {} from dir {}'.format(
        module_name, module_directory))

    module = _load_module(module_name=module_name,
                          module_directory=module_directory)

    executor_input = json.loads(args.executor_input)
    function_to_execute = getattr(module, args.function_to_execute)

    executor = component_executor.Executor(
        executor_input=executor_input, function_to_execute=function_to_execute)

    executor.execute()


if __name__ == '__main__':
    executor_main()