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

import logging
import re
import os
import time

def normalize_name(name,
              valid_first_char_pattern='a-zA-Z',
              valid_char_pattern='0-9a-zA-Z_',
              invalid_char_placeholder='_',
              prefix_placeholder='x_'):
        """Normalize a name to a valid resource name.

        Uses ``valid_first_char_pattern`` and ``valid_char_pattern`` regex pattern
        to find invalid characters from ``name`` and replaces them with 
        ``invalid_char_placeholder`` or prefix the name with ``prefix_placeholder``.

        Args:
             name: The name to be normalized.
             valid_first_char_pattern: The regex pattern for the first character.
             valid_char_pattern: The regex pattern for all the characters in the name.
             invalid_char_placeholder: The placeholder to replace invalid characters.
             prefix_placeholder: The placeholder to prefix the name if the first char 
                is invalid.
        
        Returns:
            The normalized name. Unchanged if all characters are valid.
        """
        if not name:
            return name
        normalized_name = re.sub('[^{}]+'.format(valid_char_pattern), 
            invalid_char_placeholder, name)
        if not re.match('[{}]'.format(valid_first_char_pattern), 
            normalized_name[0]):
            normalized_name = prefix_placeholder + normalized_name
        if name != normalized_name:
            logging.info('Normalize name from "{}" to "{}".'.format(
                name, normalized_name))
        return normalized_name

def dump_file(path, content):
    """Dumps string into local file.

    Args:
        path: the local path to the file.
        content: the string content to dump.
    """
    directory = os.path.dirname(path)
    if not os.path.exists(directory):
        os.makedirs(directory)
    elif os.path.exists(path):
        logging.warning('The file {} will be overwritten.'.format(path))
    with open(path, 'w') as f:
            f.write(content)

def check_resource_changed(requested_resource, 
    existing_resource, property_names):
    """Check if a resource has been changed.

    The function checks requested resource with existing resource
    by comparing specified property names. Check fails if any property
    name in the list is in ``requested_resource`` but its value is
    different with the value in ``existing_resource``.

    Args:
        requested_resource: the user requested resource paylod.
        existing_resource: the existing resource payload from data storage.
        property_names: a list of property names.

    Return:
        True if ``requested_resource`` has been changed.
    """
    for property_name in property_names:
        if not property_name in requested_resource:
            continue
        existing_value = existing_resource.get(property_name, None)
        if requested_resource[property_name] != existing_value:
            return True
    return False

def wait_operation_done(get_operation, wait_interval):
    """Waits for an operation to be done.

    Args:
        get_operation: the name of the operation.
        wait_interval: the wait interview between pulling job
            status.

    Returns:
        The completed operation.
    """
    while True:
        operation = get_operation()
        operation_name = operation.get('name')
        done = operation.get('done', False)
        if not done:
            logging.info('Operation {} is not done. Wait for {}s.'.format(
                operation_name, wait_interval))
            time.sleep(wait_interval)
            continue
        error = operation.get('error', None)
        if error:
            raise RuntimeError('Failed to complete operation {}: {} {}'.format(
                operation_name,
                error.get('code', 'Unknown code'),
                error.get('message', 'Unknown message'),
            ))
        return operation

