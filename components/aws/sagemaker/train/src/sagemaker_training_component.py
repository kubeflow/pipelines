"""SageMaker component for training."""
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

import logging
from typing import Dict, Type
from sagemaker.amazon.amazon_estimator import get_image_uri

from train.src.sagemaker_training_spec import (
    SageMakerTrainingSpec,
    SageMakerTrainingInputs,
    SageMakerTrainingOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Training Job",
    description="Train Machine Learning and Deep Learning Models using SageMaker",
    spec=SageMakerTrainingSpec,
)
class SageMakerTrainingComponent(SageMakerComponent):
    """SageMaker component for training."""

    BUILT_IN_ALGOS = {
        "blazingtext": "blazingtext",
        "deepar forecasting": "forecasting-deepar",
        "factorization machines": "factorization-machines",
        "image classification": "image-classification",
        "ip insights": "ipinsights",
        "k-means": "kmeans",
        "k-nearest neighbors": "knn",
        "k-nn": "knn",
        "lda": "lda",
        "linear learner": "linear-learner",
        "neural topic model": "ntm",
        "object2vec": "object2vec",
        "object detection": "object-detection",
        "pca": "pca",
        "random cut forest": "randomcutforest",
        "semantic segmentation": "semantic-segmentation",
        "sequence to sequence": "seq2seq",
        "seq2seq modeling": "seq2seq",
        "xgboost": "xgboost",
    }

    def Do(self, spec: SageMakerTrainingSpec):
        self._training_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="TrainingJob"
            )
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_training_job(
            TrainingJobName=self._training_job_name
        )
        status = response["TrainingJobStatus"]

        if status == "Completed":
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=status
            )
        if status == "Failed":
            message = response["FailureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTrainingInputs,
        outputs: SageMakerTrainingOutputs,
    ):
        outputs.job_name = self._training_job_name
        outputs.model_artifact_url = self._get_model_artifacts_from_job()
        outputs.training_image = self._get_image_from_job()

    def _on_job_terminated(self):
        self._sm_client.stop_training_job(TrainingJobName=self._training_job_name)

    def _print_logs_for_job(self):
        self._print_cloudwatch_logs(
            "/aws/sagemaker/TrainingJobs", self._training_job_name
        )

    def _create_job_request(
        self, inputs: SageMakerTrainingInputs, outputs: SageMakerTrainingOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_training_job
        request = self._get_request_template("train")

        request["TrainingJobName"] = self._training_job_name
        request["RoleArn"] = inputs.role
        request["HyperParameters"] = self._create_hyperparameters(
            inputs.hyperparameters
        )
        request["AlgorithmSpecification"][
            "TrainingInputMode"
        ] = inputs.training_input_mode

        ### Update training image (for BYOC and built-in algorithms) or algorithm resource name
        if not inputs.image and not inputs.algorithm_name:
            logging.error("Please specify training image or algorithm name.")
            raise Exception("Could not create job request")
        if inputs.image and inputs.algorithm_name:
            logging.error(
                "Both image and algorithm name inputted, only one should be specified. Proceeding with image."
            )

        if inputs.image:
            request["AlgorithmSpecification"]["TrainingImage"] = inputs.image
            request["AlgorithmSpecification"].pop("AlgorithmName")
        else:
            # TODO: Adjust this implementation to account for custom algorithm resources names that are the same as built-in algorithm names
            algo_name = inputs.algorithm_name.lower().strip()
            if algo_name in SageMakerTrainingComponent.BUILT_IN_ALGOS.keys():
                request["AlgorithmSpecification"]["TrainingImage"] = get_image_uri(
                    inputs.region, SageMakerTrainingComponent.BUILT_IN_ALGOS[algo_name],
                )
                request["AlgorithmSpecification"].pop("AlgorithmName")
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            # Just to give the user more leeway for built-in algorithm name inputs
            elif algo_name in SageMakerTrainingComponent.BUILT_IN_ALGOS.values():
                request["AlgorithmSpecification"]["TrainingImage"] = get_image_uri(
                    inputs.region, algo_name
                )
                request["AlgorithmSpecification"].pop("AlgorithmName")
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            else:
                request["AlgorithmSpecification"][
                    "AlgorithmName"
                ] = inputs.algorithm_name
                request["AlgorithmSpecification"].pop("TrainingImage")

        ### Update metric definitions
        if inputs.metric_definitions:
            for key, val in inputs.metric_definitions.items():
                request["AlgorithmSpecification"]["MetricDefinitions"].append(
                    {"Name": key, "Regex": val}
                )
        else:
            request["AlgorithmSpecification"].pop("MetricDefinitions")

        ### Update or pop VPC configs
        if inputs.vpc_security_group_ids and inputs.vpc_subnets:
            request["VpcConfig"][
                "SecurityGroupIds"
            ] = inputs.vpc_security_group_ids.split(",")
            request["VpcConfig"]["Subnets"] = inputs.vpc_subnets.split(",")
        else:
            request.pop("VpcConfig")

        ### Update input channels, must have at least one specified
        if len(inputs.channels) > 0:
            request["InputDataConfig"] = inputs.channels
        else:
            logging.error("Must specify at least one input channel.")
            raise Exception("Could not create job request")

        request["OutputDataConfig"]["S3OutputPath"] = inputs.model_artifact_path
        request["OutputDataConfig"]["KmsKeyId"] = inputs.output_encryption_key
        request["ResourceConfig"]["InstanceType"] = inputs.instance_type
        request["ResourceConfig"]["VolumeKmsKeyId"] = inputs.resource_encryption_key
        request["EnableNetworkIsolation"] = inputs.network_isolation
        request["EnableInterContainerTrafficEncryption"] = inputs.traffic_encryption

        ### Update InstanceCount, VolumeSizeInGB, and MaxRuntimeInSeconds if input is non-empty and > 0, otherwise use default values
        if inputs.instance_count:
            request["ResourceConfig"]["InstanceCount"] = inputs.instance_count

        if inputs.volume_size:
            request["ResourceConfig"]["VolumeSizeInGB"] = inputs.volume_size

        self._enable_spot_instance_support(request, inputs)
        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict):
        self._sm_client.create_training_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        inputs: SageMakerTrainingInputs,
        outputs: SageMakerTrainingOutputs,
    ):
        logging.info(f"Created Training Job with name: {self._training_job_name}")
        logging.info(
            "Training job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}".format(
                inputs.region, inputs.region, self._training_job_name,
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._training_job_name,
            )
        )

    def _get_model_artifacts_from_job(self):
        info = self._sm_client.describe_training_job(
            TrainingJobName=self._training_job_name
        )
        model_artifact_url = info["ModelArtifacts"]["S3ModelArtifacts"]
        return model_artifact_url

    def _get_image_from_job(self):
        info = self._sm_client.describe_training_job(
            TrainingJobName=self._training_job_name
        )
        if "TrainingImage" in info["AlgorithmSpecification"]:
            image = info["AlgorithmSpecification"]["TrainingImage"]
        else:
            algorithm_name = info["AlgorithmSpecification"]["AlgorithmName"]
            image = self._sm_client.describe_algorithm(AlgorithmName=algorithm_name)[
                "TrainingSpecification"
            ]["TrainingImage"]

        return image


if __name__ == "__main__":
    import sys

    spec = SageMakerTrainingSpec(sys.argv[1:])

    component = SageMakerTrainingComponent()
    component.Do(spec)
