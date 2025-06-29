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
import logging
import os
import sys

from kfp.dsl import executor as component_executor
from kfp.dsl import kfp_config
from kfp.dsl import utils


def _setup_logging():
    logging_format = '[KFP Executor %(asctime)s %(levelname)s]: %(message)s'
    logging.basicConfig(
        stream=sys.stdout, format=logging_format, level=logging.INFO)


def executor_main():
    _setup_logging()
    parser = argparse.ArgumentParser(description='KFP Component Executor.')

    parser.add_argument(
        '--component_module_path',
        type=str,
        help='Path to a module containing the KFP component.')

    parser.add_argument(
        '--function_to_execute',
        type=str,
        required=True,
        help='The name of the component function in '
        '--component_module_path file that is to be executed.')

    parser.add_argument(
        '--executor_input',
        type=str,
        help='JSON-serialized ExecutorInput from the orchestrator. '
        'This should contain inputs and placeholders for outputs.')

    args, _ = parser.parse_known_args()

    func_name = args.function_to_execute
    module_path = None
    module_directory = None
    module_name = None

    if args.component_module_path is not None:
        logging.info(
            f'Looking for component `{func_name}` in --component_module_path `{args.component_module_path}`'
        )
        module_path = args.component_module_path
        module_directory = os.path.dirname(args.component_module_path)
        module_name = os.path.basename(args.component_module_path)[:-len('.py')]
    else:
        # Look for module directory using kfp_config.ini
        logging.info(
            f'--component_module_path is not specified. Looking for component `{func_name}` in config file `kfp_config.ini` instead'
        )
        config = kfp_config.KFPConfig()
        components = config.get_components()
        if not components:
            raise RuntimeError('No components found in `kfp_config.ini`')
        try:
            module_path = components[func_name]
        except KeyError:
            raise RuntimeError(
                f'Could not find component `{func_name}` in `kfp_config.ini`. Found the  following components instead:\n{components}'
            )

        module_directory = str(module_path.parent)
        module_name = str(module_path.name)[:-len('.py')]

    logging.info(
        f'Loading KFP component "{func_name}" from {module_path} (directory "{module_directory}" and module name "{module_name}")'
    )

    module = utils.load_module(
        module_name=module_name, module_directory=module_directory)

    executor_input = json.loads(args.executor_input)
    function_to_execute = getattr(module, func_name)

    logging.info(f'Got executor_input:\n{json.dumps(executor_input, indent=4)}')

    executor = component_executor.Executor(
        executor_input=executor_input, function_to_execute=function_to_execute)

    output_file = executor.execute()
    if output_file is None:
        logging.info('Did not write output file.')
    else:
        logging.info(f'Wrote executor output file to {output_file}.')


if __name__ == '__main__':
    executor_main()
