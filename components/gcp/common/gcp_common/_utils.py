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