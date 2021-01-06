# Copyright 2021 Google LLC
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


def main(**kwargs):
  """Container entrypoint used by KFP Python function based component.

  This function has a dynamic signature, which will be interpreted according to
  the I/O and data-passing contract of KFP Python function components. The
  parameter will be received from command line interface.

  For each declared parameter input of the user function, three command line
  arguments will be recognized:
  1. {name of the parameter}_input_metadata_file: The metadata JSON file path
     output by the producer.
  2. {name of the parameter}_input_field_name: The output name of the parameter,
     by which the parameter can be found in the producer metadata JSON file.
  3. {name of the parameter}_input_argo_param: The actual runtime value of the
     input parameter.
  When the producer is a new-styled KFP Python component, 1 and 2 will be
  populated, and when it's a conventional KFP Python component, 3 will be in
  use.

  For each declared artifact input of the user function, two command line args
  will be recognized:
  1. {name of the artifact}_input_path: The actual path, or uri, of the input
     artifact.
  2. {name of the artifact}_input_metadata_file: The metadata JSON file path
     output by the producer.
  If the producer is a new-styled KFP Python component, 2 will be used to give
  user code access to MLMD (custom) properties associated with this artifact;
  if the producer is a conventional KFP Python component, 1 will be used to
  access only the content of the artifact.

  For each declared artifact or parameter output of the user function, a command
  line arg, namely, `{name of the artifact|parameter}_output_path`, will be
  passed to specify the location where the output content is written to.

  In addition, `executor_metadata_json_file` specifies the location where the
  output metadata JSON file will be written.
  """
  print(kwargs)


if __name__ == '__main__':
  fire.Fire(main)