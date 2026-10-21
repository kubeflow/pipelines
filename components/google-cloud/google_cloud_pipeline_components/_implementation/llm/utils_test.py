"""Unit test for utils."""

from google_cloud_pipeline_components._implementation.llm import utils
import unittest


class UtilsTest(unittest.TestCase):

  def test_build_payload_basic_success(self):
    machine_type = "n1-standard-1"
    image_uri = "fake_image_uri"
    args = ["--foo=bar"]

    expected_payload = {
        "display_name": "test_with_encryption_spec_key_name",
        "job_spec": {
            "worker_pool_specs": [{
                "replica_count": "1",
                "machine_spec": {"machine_type": machine_type},
                "container_spec": {"image_uri": image_uri, "args": args},
            }]
        },
    }

    actual_payload = utils.build_payload(
        display_name="test_with_encryption_spec_key_name",
        machine_type=machine_type,
        image_uri=image_uri,
        args=args,
    )
    self.assertDictEqual(expected_payload, actual_payload)

  def test_build_payload_with_encryption_spec_key_name(self):
    machine_type = "n1-standard-1"
    image_uri = "fake_image_uri"
    args = ["--foo=bar"]
    encryption_spec_key_name = "fake_cmek_key"

    expected_payload = {
        "display_name": "test_with_encryption_spec_key_name",
        "job_spec": {
            "worker_pool_specs": [{
                "replica_count": "1",
                "machine_spec": {"machine_type": machine_type},
                "container_spec": {"image_uri": image_uri, "args": args},
            }]
        },
        "encryption_spec": {"kms_key_name": encryption_spec_key_name},
    }

    actual_payload = utils.build_payload(
        display_name="test_with_encryption_spec_key_name",
        machine_type=machine_type,
        image_uri=image_uri,
        args=args,
        encryption_spec_key_name=encryption_spec_key_name,
    )
    self.assertDictEqual(expected_payload, actual_payload)

  def test_build_payload_with_labels_and_scheduling(self):
    machine_type = "n1-standard-1"
    image_uri = "fake_image_uri"
    args = ["--foo=bar"]
    labels = {"vertex-internal-enable-custom-job-retries": ""}
    scheduling = {"disable_retries": False}

    expected_payload = {
        "display_name": "test_with_encryption_spec_key_name",
        "job_spec": {
            "worker_pool_specs": [{
                "replica_count": "1",
                "machine_spec": {"machine_type": machine_type},
                "container_spec": {"image_uri": image_uri, "args": args},
            }],
            "scheduling": scheduling,
        },
        "labels": labels,
    }

    actual_payload = utils.build_payload(
        display_name="test_with_encryption_spec_key_name",
        machine_type=machine_type,
        image_uri=image_uri,
        args=args,
        labels=labels,
        scheduling=scheduling,
    )
    self.assertDictEqual(expected_payload, actual_payload)


if __name__ == "__main__":
  unittest.main()
