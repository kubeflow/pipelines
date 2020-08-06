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

from boto3.session import Session
from botocore.config import Config
from botocore.credentials import (
    AssumeRoleCredentialFetcher,
    CredentialResolver,
    DeferredRefreshableCredentials,
    JSONFileCache,
)
from botocore.session import Session as BotocoreSession


class Boto3Manager(object):
    class AssumeRoleProvider(object):
        METHOD = "assume-role"

        def __init__(self, fetcher):
            self._fetcher = fetcher

        def load(self):
            return DeferredRefreshableCredentials(
                self._fetcher.fetch_credentials, self.METHOD
            )

    @staticmethod
    def _get_boto3_session(region, role_arn=None):
        """Creates a boto3 session, optionally assuming a role.

        Args:
            region: The AWS region for the session.
            role_arn: The ARN to assume for the session.

        Returns:
            object: A boto3 Session.
        """

        # By default return a basic session
        if not role_arn:
            return Session(region_name=region)

        # The following assume role example was taken from
        # https://github.com/boto/botocore/issues/761#issuecomment-426037853

        # Create a session used to assume role
        assume_session = BotocoreSession()
        fetcher = AssumeRoleCredentialFetcher(
            assume_session.create_client,
            assume_session.get_credentials(),
            role_arn,
            extra_args={"DurationSeconds": 3600,},  # 1 hour assume assume by default
            cache=JSONFileCache(),
        )
        role_session = BotocoreSession()
        role_session.register_component(
            "credential_provider",
            CredentialResolver([Boto3Manager.AssumeRoleProvider(fetcher)]),
        )
        return Session(region_name=region, botocore_session=role_session)

    @staticmethod
    def get_sagemaker_client(
        component_version: str,
        region: str,
        endpoint_url: str = None,
        assume_role_arn: str = None,
    ):
        """Builds a client to the AWS SageMaker API.

        Args:
            component_version: The version of the component to include in
                the user agent.
            region: The AWS region for the SageMaker client.
            endpoint_url: A private link endpoint for SageMaker.
            assume_role_arn: The ARN of a role for the boto3 client to assume.

        Returns:
            object: A SageMaker boto3 client.
        """
        session = Boto3Manager._get_boto3_session(region, assume_role_arn)
        session_config = Config(
            user_agent="sagemaker-on-kubeflow-pipelines-v{}".format(component_version)
        )
        client = session.client(
            "sagemaker",
            region_name=region,
            endpoint_url=endpoint_url,
            config=session_config,
        )
        return client

    @staticmethod
    def get_cloudwatch_client(region: str, assume_role_arn: str = None):
        """Builds a client to the AWS CloudWatch API.

        Args:
            region: The AWS region for the CloudWatch client.

        Returns:
            object: A CloudWatch boto3 client.
        """
        session = Boto3Manager._get_boto3_session(region, assume_role_arn)
        client = session.client("logs")
        return client
