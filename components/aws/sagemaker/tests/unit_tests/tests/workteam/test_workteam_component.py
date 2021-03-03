from common.sagemaker_component import SageMakerJobStatus
from workteam.src.sagemaker_workteam_spec import SageMakerWorkteamSpec
from workteam.src.sagemaker_workteam_component import SageMakerWorkteamComponent
from tests.unit_tests.tests.workteam.test_workteam_spec import WorkteamSpecTestCase
import unittest
import json

from unittest.mock import patch, MagicMock, ANY


class WorkteamComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = WorkteamSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerWorkteamComponent()
        # Instantiate without calling Do()
        cls.component._workteam_name = "test-team"

    @patch("workteam.src.sagemaker_workteam_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerWorkteamSpec(
            self.REQUIRED_ARGS + ["--team_name", "my-team-name"]
        )

        self.component.Do(named_spec)
        self.assertEqual("my-team-name", self.component._workteam_name)

    def test_create_workteam_job(self):
        spec = SageMakerWorkteamSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "WorkteamName": "test-team",
                "MemberDefinitions": [
                    {
                        "CognitoMemberDefinition": {
                            "UserPool": None,
                            "UserGroup": "",
                            "ClientId": None,
                        }
                    }
                ],
                "Description": "fake team",
                "Tags": [],
            },
        )

    def test_create_workteam_job_most_arguments(self):
        spec = SageMakerWorkteamSpec(
            self.REQUIRED_ARGS
            + ["--sns_topic", "fake-topic", "--tags", '{"fake_key": "fake_value"}',]
        )
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "WorkteamName": "test-team",
                "MemberDefinitions": [
                    {
                        "CognitoMemberDefinition": {
                            "UserPool": None,
                            "UserGroup": "",
                            "ClientId": None,
                        }
                    }
                ],
                "Description": "fake team",
                "NotificationConfiguration": {"NotificationTopicArn": "fake-topic"},
                "Tags": [{"Key": "fake_key", "Value": "fake_value"}],
            },
        )

    def test_get_job_status(self):
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="",),
        )

    @patch("workteam.src.sagemaker_workteam_component.logging")
    def test_after_submit_job_request(self, mock_logging):
        spec = SageMakerWorkteamSpec(self.REQUIRED_ARGS)
        self.component._get_portal_domain = MagicMock(return_value="portal-domain")

        self.component._after_submit_job_request({}, {}, spec.inputs, spec.outputs)

        self.component._get_portal_domain.assert_called_once()
        mock_logging.info.assert_called_once()

    def test_after_job_completed(self):
        spec = SageMakerWorkteamSpec(self.REQUIRED_ARGS)

        mock_job_response = {"WorkteamArn": "my-arn"}
        self.component._after_job_complete(
            mock_job_response, {}, spec.inputs, spec.outputs
        )

        self.assertEqual(spec.outputs.workteam_arn, "my-arn")

    def test_get_portal_domain(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_workteam.return_value = {
            "Workteam": {"SubDomain": "portal-domain"}
        }

        self.assertEqual(self.component._get_portal_domain(), "portal-domain")
