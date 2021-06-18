#!/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Minio Executor Module."""
import os
import urllib3
from minio import Minio  #pylint: disable=no-name-in-module
from pytorch_kfp_components.components.base.base_executor import BaseExecutor
from pytorch_kfp_components.types import standard_component_specs


class Executor(BaseExecutor):
    """Minio Executor Class."""
    def __init__(self):  #pylint: disable=useless-super-delegation
        super(Executor, self).__init__()  #pylint: disable=super-with-arguments

    def _initiate_minio_client(self, minio_config: dict):  #pylint: disable=no-self-use
        """Initializes the minio client.

        Args:
            minio_config : a dict for minio configuration.
        Returns:
            client : the minio server client
        """
        minio_host = minio_config["HOST"]
        access_key = minio_config["ACCESS_KEY"]
        secret_key = minio_config["SECRET_KEY"]
        client = Minio(
            minio_host,
            access_key=access_key,
            secret_key=secret_key,
            secure=False,
        )
        return client

    def _read_minio_creds(self, endpoint: str):  #pylint: disable=no-self-use
        """Reads the minio credentials.

        Args:
            endpoint : minio endpoint url
        Raises:
            ValueError : if minio access key and secret keys are missing
        Returns:
            minio_config : a dict for minio configuration.
        """
        if "MINIO_ACCESS_KEY" not in os.environ:
            raise ValueError("Environment variable MINIO_ACCESS_KEY not found")

        if "MINIO_SECRET_KEY" not in os.environ:
            raise ValueError("Environment variable MINIO_SECRET_KEY not found")

        minio_config = {
            "HOST": endpoint,
            "ACCESS_KEY": os.environ["MINIO_ACCESS_KEY"],
            "SECRET_KEY": os.environ["MINIO_SECRET_KEY"],
        }

        return minio_config

    def upload_artifacts_to_minio(  #pylint: disable=no-self-use,too-many-arguments
        self,
        client: Minio,
        source: str,
        destination: str,
        bucket_name: str,
        output_dict: dict,
    ):
        """Uploads artifacts to minio server.

        Args:
            client : Minio client
            source : source path of artifacts.
            destination : destination path of artifacts
            bucket_name : minio bucket name.
            output_dict : dict of output containing destination paths,
                          source and bucket names
        Raises:
            Exception : on MaxRetryError, NewConnectionError,
                        ConnectionError.
        Returns:
            output_dict : dict of output containing destination paths,
                          source and bucket names
        """
        print(f"source {source} destination {destination}")
        try:
            client.fput_object(
                bucket_name=bucket_name,
                file_path=source,
                object_name=destination,
            )
            output_dict[destination] = {
                "bucket_name": bucket_name,
                "source": source,
            }
        except (
                urllib3.exceptions.MaxRetryError,
                urllib3.exceptions.NewConnectionError,
                urllib3.exceptions.ConnectionError,
                RuntimeError,
        ) as expection_raised:
            print(str(expection_raised))
            raise Exception(expection_raised)  #pylint: disable=raise-missing-from

        return output_dict

    def get_fn_args(self, input_dict: dict, exec_properties: dict):  #pylint: disable=no-self-use
        """Extracts the source, bucket_name, folder_name from the input_dict
        and endpoint from exec_properties.

        Args:
            input_dict : a dict of inputs having source,
                        destination etc.
            exec_properties : a dict of execution properties,
                                having minio endpoint.
        Returns:
            source : source path of artifacts.
            bucket_name : name of minio bucket
            folder_name : name of folder in which artifacts are uploaded.
            endpoint : minio endpoint url.
        """
        source = input_dict.get(standard_component_specs.MINIO_SOURCE)
        bucket_name = input_dict.get(
            standard_component_specs.MINIO_BUCKET_NAME)
        folder_name = input_dict.get(
            standard_component_specs.MINIO_DESTINATION)
        endpoint = exec_properties.get(standard_component_specs.MINIO_ENDPOINT)
        return source, bucket_name, folder_name, endpoint

    def Do(self, input_dict: dict, output_dict: dict, exec_properties: dict):  #pylint: disable=too-many-locals
        """Executes the minio upload process.

        Args:
            input_dict : a dict of inputs having source, destination etc.
            output_dict : dict of output containing destination paths,
                          source and bucket names
            exec_properties : a dict of execution properties,
                                having minio endpoint.

        Raises:
            ValueError : for invalid/unknonwn source path
        """

        source, bucket_name, folder_name, endpoint = self.get_fn_args(
            input_dict=input_dict, exec_properties=exec_properties)

        minio_config = self._read_minio_creds(endpoint=endpoint)

        client = self._initiate_minio_client(minio_config=minio_config)

        if not os.path.exists(source):
            raise ValueError("Input path - {} does not exists".format(source))

        if os.path.isfile(source):
            artifact_name = source.split("/")[-1]
            destination = os.path.join(folder_name, artifact_name)
            self.upload_artifacts_to_minio(
                client=client,
                source=source,
                destination=destination,
                bucket_name=bucket_name,
                output_dict=output_dict,
            )
        elif os.path.isdir(source):
            for root, dirs, files in os.walk(source):  #pylint: disable=unused-variable
                for file in files:
                    source = os.path.join(root, file)
                    artifact_name = source.split("/")[-1]
                    destination = os.path.join(folder_name, artifact_name)
                    self.upload_artifacts_to_minio(
                        client=client,
                        source=source,
                        destination=destination,
                        bucket_name=bucket_name,
                        output_dict=output_dict,
                    )
        else:
            raise ValueError("Unknown source: {} ".format(source))
