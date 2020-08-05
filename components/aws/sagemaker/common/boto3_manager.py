"""Class for managing boto3 sessions."""
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

import boto3
import botocore
from botocore.exceptions import ClientError


class Boto3Manager(object):
    @staticmethod
    def get_sagemaker_client(component_version, region, endpoint_url=None):
        """Builds a client to the AWS SageMaker API.

        Args:
            component_version: The version of the component to include in
                the user agent.
            region: The AWS region for the SageMaker client.
            endpoint_url: A private link endpoint for SageMaker.

        Returns:
            object: A SageMaker boto3 client.
        """
        session_config = botocore.config.Config(
            user_agent="sagemaker-on-kubeflow-pipelines-v{}".format(component_version)
        )
        client = boto3.client(
            "sagemaker",
            region_name=region,
            endpoint_url=endpoint_url,
            config=session_config,
        )
        return client

    @staticmethod
    def get_cloudwatch_client(region):
        """Builds a client to the AWS CloudWatch API.

        Args:
            region: The AWS region for the CloudWatch client.

        Returns:
            object: A CloudWatch boto3 client.
        """
        client = boto3.client("logs", region_name=region)
        return client
