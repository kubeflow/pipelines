# Copyright 2020 Google LLC
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
"""The entrypoint binary used in KFP component."""
import argparse
import sys


class ParseKwargs(argparse.Action):
  """Helper class to parse the keyword arguments.

  This Python binary expects a set of kwargs, whose keys are not predefined.
  """
  def __call__(
      self, parser, namespace, values, option_string=None):
    setattr(namespace, self.dest, dict())
    assert len(values) % 2 == 0, 'Each specified arg key must have a value.'
    current_key = None
    for idx, value in enumerate(values):
      if idx % 2 == 0:
        # Parse this into a key.
        current_key = value
      else:
        # Parse current value with the previous key.
        getattr(namespace, self.dest)[current_key] = value


def main():
  """The main program of KFP container entrypoint.

  This entrypoint should be called as follows:
  python run_container.py -k key1 value1 key2 value2 ...

  The recognized argument keys are as follows:
  - {input-parameter-name}_metadata_file
  - {input_parameter-name}_field_name
  - {}
  """
  parser = argparse.ArgumentParser()
  parser.add_argument('-k', '--kwargs', nargs='*', action=ParseKwargs)
  args = parser.parse_args(sys.argv[1:])  # Skip the file name.
  print(args.kwargs)


if __name__ == '__main__':
  main()