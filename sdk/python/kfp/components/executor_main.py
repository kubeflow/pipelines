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
import importlib
from importlib import util
import sys

from kfp.components.executor import Executor

def _load_module(module_path: str):
  module_spec = importlib.util.spec_from_file_location('mycomponent', module_path + '/mycomponent.py')
  module = importlib.util.module_from_spec(module_spec)
  sys.modules[module_spec.name] = module
  module_spec.loader.exec_module(module)
  return module

def executor_main():
  import argparse
  import json

  parser = argparse.ArgumentParser(description='KFP Component Executor.')
  parser.add_argument('--component_module_path', type=str)
  parser.add_argument('--executor_input', type=str)
  parser.add_argument('--function_to_execute', type=str)

  args, _ = parser.parse_known_args()

  print('Using program path: {}'.format(args.component_module_path))
  with open(args.component_module_path + '/mycomponent.py', 'r') as f:
    print('Program contents: {}'.format(f.read()))

  module = _load_module(args.component_module_path)


  executor_input = json.loads(args.executor_input)
  function_to_execute = getattr(module, args.function_to_execute)

  executor = Executor(executor_input=executor_input,
                      function_to_execute=function_to_execute)

  executor.execute()

if __name__ == '__main__':
  executor_main()