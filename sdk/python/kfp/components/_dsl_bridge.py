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

_dummy_pipeline=None

def _create_task_object(name:str, container_image:str, command=None, arguments=None, file_inputs=None, file_outputs=None):
    import kfp.dsl as dsl
    global _dummy_pipeline
    need_dummy = dsl.Pipeline._default_pipeline is None
    if need_dummy:
        if _dummy_pipeline == None:
            _dummy_pipeline = dsl.Pipeline('dummy pipeline')
        _dummy_pipeline.__enter__()

    task = dsl.ContainerOp(
        name=name,
        image=container_image,
        command=command,
        arguments=arguments,
        file_inputs=file_inputs,
        file_outputs=file_outputs,
    )

    if need_dummy:
        _dummy_pipeline.__exit__()

    return task


_task_object_factory=_create_task_object
