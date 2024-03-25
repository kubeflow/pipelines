import os
import unittest
from unittest.mock import patch, MagicMock

from commonv2.sagemaker_component import (
    SageMakerComponent,
)
from tests.unit_tests.tests.commonv2.dummy_spec import (
    DummySpec,
)


class SageMakerAbstractComponentTestCase(unittest.TestCase):
    @classmethod
    def setUp(cls):
        """Bootstrap Unit test resources for testing runtime.
        """
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
    
    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_create_custom_resource(self, mock_custom_objects_api):
        """ Testing creating a custom resource.
        """

        cr_dict = {}
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"

        self.component._create_custom_resource(cr_dict)
        mock_custom_objects_api().create_cluster_custom_object.assert_called_once_with(
            "group-test", "version-test", "plural-test", {}
        )

        self.component.namespace = "namespace-test"
        self.component._create_custom_resource(cr_dict)
        mock_custom_objects_api().create_namespaced_custom_object.assert_called_once_with(
            "group-test", "version-test", "namespace-test", "plural-test", {}
        )

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_get_resource(self, mock_custom_objects_api):
        """ Testing get resource.
        """
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"
        self.component.job_name = "ack-job-name-test"

        self.component._get_resource()
        mock_custom_objects_api().get_cluster_custom_object.assert_called_once_with(
            "group-test", "version-test", "plural-test", "ack-job-name-test"
        )

        self.component.namespace = "namespace-test"
        self.component._get_resource()
        mock_custom_objects_api().get_namespaced_custom_object.assert_called_once_with(
            "group-test",
            "version-test",
            "namespace-test",
            "plural-test",
            "ack-job-name-test",
        )

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_delete_custom_resource(self, mock_custom_objects_api):
        """ Testing deletion
        """
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"
        self.component.job_name = "ack-job-name-test"

        with patch.object(
            SageMakerComponent, "_get_resource_exists", return_value=False
        ) as mock_exists:
            _, ret_val = self.component._delete_custom_resource()
            mock_custom_objects_api().delete_cluster_custom_object.assert_called_once_with(
                "group-test", "version-test", "plural-test", "ack-job-name-test"
            )
            self.assertTrue(ret_val)

    def test_create_job_yaml(self):
        """Verify that the create yaml function works as intended.
        """
        self.component.job_name = "ack-job-name-test"
        with patch(
            "builtins.open",
            MagicMock(return_value=open(self.component.job_request_outline_location)),
        ) as mock_open:
            Args = ["--input1", "abc123", "--input2", "123", "--region", "us-west-1"]
            nSpec = DummySpec(Args)

            result = self.component._create_job_yaml(nSpec._inputs, DummySpec.OUTPUTS)

            sample = {
                "apiVersion": "sagemaker.services.k8s.aws/v1alpha1",
                "kind": "TrainingJob",
                "metadata": {
                    "name": "ack-job-name-test",
                    "annotations": {"services.k8s.aws/region": "us-west-1"},
                },
                "spec": {
                    "spotInstance": False,
                    "maxWaitTime": 1,
                    "inputStr": "input",
                    "inputInt": 1,
                    "inputBool": False,
                },
            }
            print("\n\n", result, "\n\n", sample, "\n\n")

            self.assertDictEqual(result, sample)


if __name__ == "__main__":
    unittest.main()
