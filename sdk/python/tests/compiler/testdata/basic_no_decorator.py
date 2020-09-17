# Copyright 2019 Google LLC
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


import kfp.dsl as dsl
import kfp.gcp as gcp


message_param = dsl.PipelineParam(name='message')
output_path_param = dsl.PipelineParam(name='outputpath', value='default_output')

class GetFrequentWordOp(dsl.ContainerOp):
  """A get frequent word class representing a component in ML Pipelines.

  The class provides a nice interface to users by hiding details such as container,
  command, arguments.
  """
  def __init__(self, name, message):
    """Args:
         name: An identifier of the step which needs to be unique within a pipeline.
         message: a dsl.PipelineParam object representing an input message.
    """
    super(GetFrequentWordOp, self).__init__(
        name=name,
        image='python:3.5-jessie',
        command=['sh', '-c'],
        arguments=['python -c "from collections import Counter; '
                   'words = Counter(\'%s\'.split()); print(max(words, key=words.get))" '
                   '| tee /tmp/message.txt' % message],
        file_outputs={'word': '/tmp/message.txt'})


class SaveMessageOp(dsl.ContainerOp):
  """A class representing a component in ML Pipelines.

  It saves a message to a given output_path.
  """
  def __init__(self, name, message, output_path):
    """Args:
         name: An identifier of the step which needs to be unique within a pipeline.
         message: a dsl.PipelineParam object representing the message to be saved.
         output_path: a dsl.PipelineParam object representing the GCS path for output file.
    """
    super(SaveMessageOp, self).__init__(
        name=name,
        image='google/cloud-sdk',
        command=['sh', '-c'],
        arguments=['echo %s | tee /tmp/results.txt | gsutil cp /tmp/results.txt %s'
                   % (message, output_path)])


class ExitHandlerOp(dsl.ContainerOp):
  """A class representing a component in ML Pipelines.
  """
  def __init__(self, name):
    super(ExitHandlerOp, self).__init__(
        name=name,
        image='python:3.5-jessie',
        command=['sh', '-c'],
        arguments=['echo exit!'])

def save_most_frequent_word():
  exit_op = ExitHandlerOp('exiting')
  with dsl.ExitHandler(exit_op):
    counter = GetFrequentWordOp(
        name='get-Frequent',
        message=message_param)
    counter.container.set_memory_request('200M')

    saver = SaveMessageOp(
        name='save',
        message=counter.output,
        output_path=output_path_param)
    saver.container.set_cpu_limit('0.5')
    saver.container.set_gpu_limit('2')
    saver.add_node_selector_constraint(
        'cloud.google.com/gke-accelerator',
        'nvidia-tesla-k80')
    saver.apply(
        gcp.use_tpu(tpu_cores = 8, tpu_resource = 'v2', tf_version = '1.12'))