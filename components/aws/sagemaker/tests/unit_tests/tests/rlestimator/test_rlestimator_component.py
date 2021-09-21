from common.sagemaker_component import SageMakerJobStatus
from rlestimator.src.sagemaker_rlestimator_spec import SageMakerRLEstimatorSpec
from rlestimator.src.sagemaker_rlestimator_component import (
    SageMakerRLEstimatorComponent,
    DebugRulesStatus,
)
from tests.unit_tests.tests.rlestimator.test_rlestimator_spec import (
    RLEstimatorSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock

HAS_ATTR_MESSAGE = "{} should have an attribute {}"
HAS_NOT_ATTR_MESSAGE = "{} should not have an attribute {}"
ATTR_NOT_NONE_MESSAGE = "{} attribute {} should be None"


class BaseTestCase(unittest.TestCase):
    def assertHasAttr(self, obj, attrname, message=None):
        if not hasattr(obj, attrname):
            if message is not None:
                self.fail(message)
            else:
                self.fail(HAS_ATTR_MESSAGE.format(obj, attrname))

    def assertHasNotAttr(self, obj, attrname, message=None):
        if hasattr(obj, attrname):
            if message is not None:
                self.fail(message)
            else:
                self.fail(HAS_NOT_ATTR_MESSAGE.format(obj, attrname))

    def assertAttrNone(self, obj, attrname, message=None):
        if not hasattr(obj, attrname):
            if message is not None:
                self.fail(message)
            else:
                self.fail(HAS_NOT_ATTR_MESSAGE.format(obj, attrname))
        if getattr(obj, attrname) is not None:
            if message is not None:
                self.fail(message)
            else:
                self.fail(ATTR_NOT_NONE_MESSAGE.format(obj, attrname))


class RLEstimatorComponentTestCase(BaseTestCase):

    CUSTOM_IMAGE_ARGS = RLEstimatorSpecTestCase.CUSTOM_IMAGE_ARGS
    TOOLKIT_IMAGE_ARGS = RLEstimatorSpecTestCase.TOOLKIT_IMAGE_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerRLEstimatorComponent()
        # Instantiate without calling Do()
        cls.component._rlestimator_job_name = "test-job"
        cls.component._sagemaker_session = MagicMock()

    @patch("rlestimator.src.sagemaker_rlestimator_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS + ["--job_name", "job-name"]
        )
        unnamed_spec = SageMakerRLEstimatorSpec(self.CUSTOM_IMAGE_ARGS)

        self.component.Do(named_spec)
        self.assertEqual("job-name", self.component._rlestimator_job_name)

        with patch(
            "rlestimator.src.sagemaker_rlestimator_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="unique"),
        ):
            self.component.Do(unnamed_spec)
            self.assertEqual("unique", self.component._rlestimator_job_name)

    def test_create_rlestimator_custom_job(self):
        spec = SageMakerRLEstimatorSpec(self.CUSTOM_IMAGE_ARGS)
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertHasAttr(rlestimator, "image_uri")
        self.assertHasAttr(rlestimator, "role")
        self.assertHasAttr(rlestimator, "source_dir")
        self.assertHasAttr(rlestimator, "entry_point")
        self.assertHasNotAttr(rlestimator, "toolkit")
        self.assertHasNotAttr(rlestimator, "toolkit_version")
        self.assertHasNotAttr(rlestimator, "framework")

    def test_create_rlestimator_toolkit_job(self):
        spec = SageMakerRLEstimatorSpec(self.TOOLKIT_IMAGE_ARGS)
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertHasAttr(rlestimator, "role")
        self.assertHasAttr(rlestimator, "source_dir")
        self.assertHasAttr(rlestimator, "entry_point")
        self.assertHasAttr(rlestimator, "toolkit")
        self.assertHasAttr(rlestimator, "toolkit_version")
        self.assertHasAttr(rlestimator, "framework")
        self.assertAttrNone(rlestimator, "image_uri")

    def test_get_job_status(self):
        self.component._sm_client = mock_client = MagicMock()
        self.component._get_debug_rule_status = MagicMock(
            return_value=SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status="Completed"
            )
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Failed",
            "FailureReason": "lolidk",
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="Failed",
                has_error=True,
                error_message="lolidk",
            ),
        )

    def test_after_job_completed(self):
        self.component._get_model_artifacts_from_job = MagicMock(return_value="model")
        self.component._get_image_from_job = MagicMock(return_value="image")

        spec = SageMakerRLEstimatorSpec(self.CUSTOM_IMAGE_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.job_name, "test-job")
        self.assertEqual(spec.outputs.model_artifact_url, "model")
        self.assertEqual(spec.outputs.training_image, "image")

    def test_metric_definitions(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + [
                "--metric_definitions",
                '[ {"Name": "metric1", "Regex": "regexval1"},{"Name": "metric2", "Regex": "regexval2"},]',
            ]
        )

        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertEqual(
            getattr(rlestimator, "metric_definitions"),
            [
                {"Name": "metric1", "Regex": "regexval1"},
                {"Name": "metric2", "Regex": "regexval2"},
            ],
        )

    def test_no_defined_image(self):
        # Pass the image to pass the parser
        no_image_args = self.CUSTOM_IMAGE_ARGS.copy()
        image_index = no_image_args.index("--image")
        # Cut out --image and it's associated value
        no_image_args = no_image_args[:image_index] + no_image_args[image_index + 2 :]

        spec = SageMakerRLEstimatorSpec(no_image_args)

        with self.assertRaises(Exception):
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_valid_hyperparameters(self):
        hyperparameters_str = '{"hp1": "val1", "hp2": "val2", "hp3": "val3"}'

        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertIn("hp1", getattr(rlestimator, "_hyperparameters"))
        self.assertIn("hp2", getattr(rlestimator, "_hyperparameters"))
        self.assertIn("hp3", getattr(rlestimator, "_hyperparameters"))
        self.assertEqual(getattr(rlestimator, "_hyperparameters")["hp1"], "val1")
        self.assertEqual(getattr(rlestimator, "_hyperparameters")["hp2"], "val2")
        self.assertEqual(getattr(rlestimator, "_hyperparameters")["hp3"], "val3")

    def test_empty_hyperparameters(self):
        hyperparameters_str = "{}"

        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(getattr(rlestimator, "_hyperparameters"), {})

    def test_object_hyperparameters(self):
        hyperparameters_str = '{"hp1": {"innerkey": "innerval"}}'

        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        with self.assertRaises(Exception):
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_vpc_configuration(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + [
                "--vpc_security_group_ids",
                '["sg1", "sg2"]',
                "--vpc_subnets",
                '["subnet1", "subnet2"]',
            ]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertHasAttr(rlestimator, "subnets")
        self.assertHasAttr(rlestimator, "security_group_ids")
        self.assertIn("sg1", getattr(rlestimator, "security_group_ids"))
        self.assertIn("sg2", getattr(rlestimator, "security_group_ids"))
        self.assertIn("subnet1", getattr(rlestimator, "subnets"))
        self.assertIn("subnet2", getattr(rlestimator, "subnets"))

    def test_training_mode(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS + ["--training_input_mode", "Pipe"]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(getattr(rlestimator, "input_mode"), "Pipe")

    def test_wait_for_debug_rules(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.side_effect = [
            {
                "DebugRuleEvaluationStatuses": [
                    {
                        "RuleConfigurationName": "rule1",
                        "RuleEvaluationStatus": "InProgress",
                    },
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "InProgress",
                    },
                ]
            },
            {
                "DebugRuleEvaluationStatuses": [
                    {
                        "RuleConfigurationName": "rule1",
                        "RuleEvaluationStatus": "NoIssuesFound",
                    },
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "InProgress",
                    },
                ]
            },
            {
                "DebugRuleEvaluationStatuses": [
                    {
                        "RuleConfigurationName": "rule1",
                        "RuleEvaluationStatus": "NoIssuesFound",
                    },
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "IssuesFound",
                    },
                ]
            },
        ]
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=False,
                has_error=False,
                raw_status=DebugRulesStatus.INPROGRESS,
            ),
        )
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=False,
                has_error=False,
                raw_status=DebugRulesStatus.INPROGRESS,
            ),
        )
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=True,
                has_error=False,
                raw_status=DebugRulesStatus.COMPLETED,
            ),
        )

    def test_wait_for_errored_rule(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.side_effect = [
            {
                "DebugRuleEvaluationStatuses": [
                    {
                        "RuleConfigurationName": "rule1",
                        "RuleEvaluationStatus": "InProgress",
                    },
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "InProgress",
                    },
                ]
            },
            {
                "DebugRuleEvaluationStatuses": [
                    {"RuleConfigurationName": "rule1", "RuleEvaluationStatus": "Error"},
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "InProgress",
                    },
                ]
            },
            {
                "DebugRuleEvaluationStatuses": [
                    {"RuleConfigurationName": "rule1", "RuleEvaluationStatus": "Error"},
                    {
                        "RuleConfigurationName": "rule2",
                        "RuleEvaluationStatus": "NoIssuesFound",
                    },
                ]
            },
        ]
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=False,
                has_error=False,
                raw_status=DebugRulesStatus.INPROGRESS,
            ),
        )
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=False,
                has_error=False,
                raw_status=DebugRulesStatus.INPROGRESS,
            ),
        )
        self.assertEqual(
            self.component._get_debug_rule_status(),
            SageMakerJobStatus(
                is_completed=True, has_error=True, raw_status=DebugRulesStatus.ERRORED
            ),
        )

    def test_hook_min_args(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + ["--debug_hook_config", '{"S3OutputPath": "s3://fake-uri/"}']
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertEqual(
            getattr(rlestimator, "debugger_hook_config")["S3OutputPath"],
            "s3://fake-uri/",
        )

    def test_hook_max_args(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + [
                "--debug_hook_config",
                '{"S3OutputPath": "s3://fake-uri/", "LocalPath": "/local/path/", "HookParameters": {"key": "value"}, "CollectionConfigurations": [{"CollectionName": "collection1", "CollectionParameters": {"key1": "value1"}}, {"CollectionName": "collection2", "CollectionParameters": {"key2": "value2", "key3": "value3"}}]}',
            ]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertEqual(
            getattr(rlestimator, "debugger_hook_config")["S3OutputPath"],
            "s3://fake-uri/",
        )
        self.assertEqual(
            getattr(rlestimator, "debugger_hook_config")["LocalPath"], "/local/path/"
        )
        self.assertEqual(
            getattr(rlestimator, "debugger_hook_config")["HookParameters"],
            {"key": "value"},
        )
        self.assertEqual(
            getattr(rlestimator, "debugger_hook_config")["CollectionConfigurations"],
            [
                {
                    "CollectionName": "collection1",
                    "CollectionParameters": {"key1": "value1"},
                },
                {
                    "CollectionName": "collection2",
                    "CollectionParameters": {"key2": "value2", "key3": "value3"},
                },
            ],
        )

    def test_rule_max_args(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + [
                "--debug_rule_config",
                '[{"InstanceType": "ml.m4.xlarge", "LocalPath": "/local/path/", "RuleConfigurationName": "rule_name", "RuleEvaluatorImage": "test-image", "RuleParameters": {"key1": "value1"}, "S3OutputPath": "s3://fake-uri/", "VolumeSizeInGB": 1}]',
            ]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)
        attrs = vars(rlestimator)
        print(", ".join("%s: %s" % item for item in attrs.items()))
        print(getattr(rlestimator, "debugger_rule_configs"))
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["InstanceType"], "ml.m4.xlarge"
        )
        self.assertEqual(getattr(rlestimator, "rules")[0]["LocalPath"], "/local/path/")
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["RuleConfigurationName"], "rule_name"
        )
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["RuleEvaluatorImage"], "test-image"
        )
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["RuleParameters"], {"key1": "value1"}
        )
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["S3OutputPath"], "s3://fake-uri/"
        )
        self.assertEqual(getattr(rlestimator, "rules")[0]["VolumeSizeInGB"], 1)

    def test_rule_min_good_args(self):
        spec = SageMakerRLEstimatorSpec(
            self.CUSTOM_IMAGE_ARGS
            + [
                "--debug_rule_config",
                '[{"RuleConfigurationName": "rule_name", "RuleEvaluatorImage": "test-image"}]',
            ]
        )
        rlestimator = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            getattr(rlestimator, "rules")[0]["RuleConfigurationName"], "rule_name"
        )
        self.assertEqual(
            getattr(rlestimator, "rules")[0]["RuleEvaluatorImage"], "test-image"
        )
