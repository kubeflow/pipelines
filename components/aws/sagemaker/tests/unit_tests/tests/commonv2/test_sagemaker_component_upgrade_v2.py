import os
import unittest
from unittest.mock import patch, MagicMock

from commonv2.sagemaker_component import (
    SageMakerComponent,
)
from kubernetes.client.rest import ApiException


class SageMakerComponentUpgradeTest(unittest.TestCase):
    @classmethod
    def setUp(cls):
        """Bootstrap Unit test resources for testing runtime."""
        cls.component = SageMakerComponent()
        # Turn off polling interval for instant tests
        cls.component.STATUS_POLL_INTERVAL = 0

        # set up mock job_request_outline_location
        test_files_dir = os.path.join(
            os.path.dirname(os.path.relpath(__file__)), "files"
        )
        test_job_request_outline_location = os.path.join(
            test_files_dir, "test_job_request_outline.yaml.tpl"
        )
        cls.component.job_request_outline_location = test_job_request_outline_location
        cls.component.spaced_out_resource_name = "Test Resource"

    def test_is_upgrade(self):
        self.component._get_resource = MagicMock(return_value={"status": "Active"})
        assert self.component._is_upgrade()
        self.component._get_resource = MagicMock(side_effect=ApiException(status=404))
        assert self.component._is_upgrade() == False

    def test_verify_resource_creation_upgrade(self):

        # Time change
        self.component._check_resource_conditions = MagicMock(return_value=None)
        initial_resource =  {
                "status": {
                    "ackResourceMetadata": {
                        "arn": "arn:aws:sagemaker:eu-west-3:123456789103:stack/sample-endpoint"
                    }
                }
        }
        updated_resource = {
                "status": {
                    "ackResourceMetadata": {
                        "arn": "arn:aws:sagemaker:eu-west-3:123456789103:stack/sample-endpoint"
                    },
                    "test": "t1"
                }
        }
        self.component._get_resource = MagicMock(
            return_value=updated_resource
        )
        self.component.initial_status = initial_resource["status"]
        self.component.resource_upgrade = True
        assert self.component._verify_resource_consumption() == True

        # Upgrade Case - same time initially
        with patch("time.sleep", return_value=None):
            self.component._get_resource = MagicMock(
                side_effect=[initial_resource,updated_resource]
            )
            assert self.component._verify_resource_consumption() == True

