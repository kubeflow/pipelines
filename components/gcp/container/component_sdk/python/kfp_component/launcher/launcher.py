# Copyright 2018 Google LLC
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

import fire
import importlib
import sys
import logging

def launch(file_or_module, args):
    """Launches a python file or module as a command entrypoint.

    Args:
        file_or_module: it is either a file path to python file
            a module path.
        args: the args passed to the entrypoint function.

    Returns:
        The return value from the launched function.
    """
    try:
        module = importlib.import_module(file_or_module)
    except Exception:
        try:
            if sys.version_info.major > 2:
                spec = importlib.util.spec_from_file_location('module', file_or_module)
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
            else:
                import imp
                module = imp.load_source('module', file_or_module)
        except Exception:
            logging.error('Failed to find the module or file: {}'.format(file_or_module))
            sys.exit(1)
    return fire.Fire(module, command=args, name=module.__name__)