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
import time

import googleapiclient.discovery as discovery
from ..common import wait_operation_done

class DataprocClient:
    """ Internal client for calling Dataproc APIs.
    """
    def __init__(self):
        self._dataproc = discovery.build('dataproc', 'v1', cache_discovery=False)

    def create_cluster(self, project_id, region, cluster, request_id):
        """Creates a new dataproc cluster.
        """
        return self._dataproc.projects().regions().clusters().create(
            projectId = project_id,
            region = region,
            requestId = request_id,
            body = cluster
        ).execute()

    def get_cluster(self, project_id, region, name):
        """Gets the resource representation for a cluster in a project.
        """
        return self._dataproc.projects().regions().clusters().get(
            projectId = project_id,
            region = region,
            clusterName = name
        ).execute()

    def delete_cluster(self, project_id, region, name, request_id):
        """Deletes a cluster in a project.
        """
        return self._dataproc.projects().regions().clusters().delete(
            projectId = project_id,
            region = region,
            clusterName = name,
            requestId = request_id
        ).execute()

    def submit_job(self, project_id, region, job, request_id):
        """Submits a job to a cluster.
        """
        return self._dataproc.projects().regions().jobs().submit(
            projectId = project_id,
            region = region,
            body = {
                'job': job,
                'requestId': request_id
            }
        ).execute()

    def get_job(self, project_id, region, job_id):
        """Gets a job details
        """
        return self._dataproc.projects().regions().jobs().get(
            projectId = project_id,
            region = region,
            jobId = job_id
        ).execute()

    def cancel_job(self, project_id, region, job_id):
        """Cancels a job
        """
        return self._dataproc.projects().regions().jobs().cancel(
            projectId = project_id,
            region = region,
            jobId = job_id
        ).execute()

    def get_operation(self, operation_name):
        """Gets a operation by name.
        """
        return self._dataproc.projects().regions().operations().get(
            name = operation_name
        ).execute()

    def wait_for_operation_done(self, operation_name, wait_interval):
        """Waits for an operation to be done.

        Args:
            operation_name: the name of the operation.
            wait_interval: the wait interview between pulling job
                status.

        Returns:
            The completed operation.
        """
        return wait_operation_done(
            lambda: self.get_operation(operation_name), wait_interval)

    def cancel_operation(self, operation_name):
        """Cancels an operation.

        Args:
            operation_name: the name of the operation.
        """
        if not operation_name:
            return

        self._dataproc.projects().regions().operations().cancel(
            name = operation_name
        ).execute()
