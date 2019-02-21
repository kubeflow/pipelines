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

from ._client import DataprocClient
from kfp_component.core import KfpExecutionContext

def delete_cluster(project_id, region, name, wait_interval=30):
    client = DataprocClient()
    operation_name = None
    with KfpExecutionContext(
        on_cancel=lambda: client.cancel_operation(operation_name)) as ctx:
        operation = client.delete_cluster(project_id, region, name, 
            request_id=ctx.context_id())
        operation_name = operation.get('name')
        return client.wait_for_operation_done(operation_name, 
            wait_interval)