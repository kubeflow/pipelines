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
from googleapiclient import errors
from ..common import wait_operation_done

class MLEngineClient:
    """ Client for calling MLEngine APIs.
    """
    def __init__(self):
        self._ml_client = discovery.build('ml', 'v1', cache_discovery=False)

    def create_job(self, project_id, job):
        """Create a new job.

        Args:
            project_id: the ID of the parent project.
            job: the payload of the job.

        Returns:
            The created job.
        """
        return self._ml_client.projects().jobs().create(
            parent = 'projects/{}'.format(project_id),
            body = job
        ).execute()

    def cancel_job(self, project_id, job_id):
        """Cancel the specified job.

        Args:
            project_id: the parent project ID of the job.
            job_id: the ID of the job.
        """
        job_name = 'projects/{}/jobs/{}'.format(project_id, job_id)
        self._ml_client.projects().jobs().cancel(
            name = job_name,
            body = {
                'name': job_name
            },
        ).execute()

    def get_job(self, project_id, job_id):
        """Gets the job by ID.

        Args:
            project_id: the ID of the parent project.
            job_id: the ID of the job to retrieve.
        Returns:
            The retrieved job payload.
        """
        job_name = 'projects/{}/jobs/{}'.format(project_id, job_id)
        return self._ml_client.projects().jobs().get(
            name=job_name).execute()

    def create_model(self, project_id, model):
        """Creates a new model.

        Args:
            project_id: the ID of the parent project.
            model: the payload of the model.
        Returns:
            The created model.
        """
        return self._ml_client.projects().models().create(
            parent = 'projects/{}'.format(project_id),
            body = model
        ).execute()

    def get_model(self, model_name):
        """Gets a model.

        Args:
            model_name: the name of the model.
        Returns:
            The retrieved model.
        """
        return self._ml_client.projects().models().get(
            name = model_name
        ).execute()

    def create_version(self, model_name, version):
        """Creates a new version.

        Args:
            model_name: the name of the parent model.
            version: the payload of the version.

        Returns:
            The created version.
        """
        return self._ml_client.projects().models().versions().create(
            parent = model_name,
            body = version
        ).execute()

    def get_version(self, version_name):
        """Gets a version.

        Args:
            version_name: the name of the version.

        Returns:
            The retrieved version. None if the version is not found.
        """
        try:
            return self._ml_client.projects().models().versions().get(
                name = version_name
            ).execute()
        except errors.HttpError as e:
            if e.resp.status == 404:
                return None
            raise

    def delete_version(self, version_name):
        """Deletes a version.

        Args:
            version_name: the name of the version.

        Returns:
            The delete operation. None if the version is not found.
        """
        try:
            return self._ml_client.projects().models().versions().delete(
                name = version_name
            ).execute()
        except errors.HttpError as e:
            if e.resp.status == 404:
                logging.info('The version has already been deleted.')
                return None
            raise

    def set_default_version(self, version_name):
        return self._ml_client.projects().models().versions().setDefault(
            name = version_name
        ).execute()

    def get_operation(self, operation_name):
        """Gets an operation.

        Args:
            operation_name: the name of the operation.

        Returns:
            The retrieved operation.
        """
        return self._ml_client.projects().operations().get(
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
        self._ml_client.projects().operations().cancel(
            name = operation_name
        ).execute()
