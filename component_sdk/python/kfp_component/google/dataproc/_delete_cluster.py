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
from googleapiclient import errors
from ._client import DataprocClient
from kfp_component.core import KfpExecutionContext

def delete_cluster(project_id, region, name, wait_interval=30):
    """Deletes a DataProc cluster.
    
    Args:
        project_id (str): Required. The ID of the Google Cloud Platform project 
            that the cluster belongs to.
        region (str): Required. The Cloud Dataproc region in which to handle the 
            request.
        name (str): Required. The cluster name to delete.
        wait_interval (int): The wait seconds between polling the operation. 
            Defaults to 30s.

    """
    client = DataprocClient()
    operation_name = None
    with KfpExecutionContext(
        on_cancel=lambda: client.cancel_operation(operation_name)) as ctx:
        try:
            operation = client.delete_cluster(project_id, region, name, 
                request_id=ctx.context_id())
        except errors.HttpError as e:
            if e.resp.status == 404:
                logging.info('Cluster {} is not found.'.format(name))
                return
            raise e
        operation_name = operation.get('name')
        return client.wait_for_operation_done(operation_name, 
            wait_interval)