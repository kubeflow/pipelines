"""Custom argument parser validators for SageMaker components"""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from argparse import ArgumentTypeError

class SpecValidators:
    @staticmethod
    def nullable_string_argument(value):
        """Strips strings and returns None if they are empty"""
        value = value.strip()
        if not value:
            return None
        return value

    @staticmethod
    def _yaml_or_json_str(str):
        if str == "" or str == None:
            return None
        try:
            return json.loads(str)
        except:
            return yaml.safe_load(str)

    @staticmethod
    def yaml_or_json_list(str):
        """Parses a YAML or JSON list to a Python list"""
        parsed = SpecValidators._yaml_or_json_str
        if not isinstance(parsed, list):
            raise ArgumentTypeError(f"{str} is not a list")
        return parsed

    @staticmethod
    def yaml_or_json_dict(str):
        """Parses a YAML or JSON dictionary to a Python dictionary"""
        parsed = SpecValidators._yaml_or_json_str
        if not isinstance(parsed, dict):
            raise ArgumentTypeError(f"{str} is not a dictionary")
        return parsed

    @staticmethod
    def str_to_bool(str):
        """Converts a string interpretation of a boolean to a Python bool"""
        # This distutils function returns an integer representation of the boolean
        # rather than a True/False value. This simply hard casts it.
        return bool(strtobool(str))
