# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import List

from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath


@container_component
def hyperparameter_tuning_job(
    project: str,
    display_name: str,
    base_output_directory: str,
    worker_pool_specs: List[str],
    study_spec_metrics: List[str],
    study_spec_parameters: List[str],
    max_trial_count: int,
    parallel_trial_count: int,
    gcp_resources: OutputPath(str),
    max_failed_trial_count: int = 0,
    location: str = 'us-central1',
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    encryption_spec_key_name: str = '',
    service_account: str = '',
    network: str = '',
):
  """Creates a Google Cloud AI Platform HyperparameterTuning Job and waits for it to complete.

    For example usage, see
    https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/experimental/hyperparameter_tuning_job/hp_tuning_job_sample.ipynb.

    For more information on using hyperparameter tuning, please visit:
    https://cloud.google.com/vertex-ai/docs/training/using-hyperparameter-tuning

    Args:
        display_name (str): Required. The user-defined name of the
          HyperparameterTuningJob. The name can be up to 128 characters long and
          can be consist of any UTF-8 characters.
        project (str): Required. Project to run the HyperparameterTuningJob in.
        base_output_directory (str): Required. The Cloud Storage location to
          store the output of this HyperparameterTuningJob. The
          base_output_directory of each child CustomJob backing a Trial is set
          to a subdirectory with name as the trial id under its parent
          HyperparameterTuningJob's base_output_directory. The following Vertex
          AI environment variables will be passed to containers or python
          modules when this field is set: For CustomJob backing a Trial of
          HyperparameterTuningJob: * AIP_MODEL_DIR = `\/\/model\/` *
            AIP_CHECKPOINT_DIR = `\/\/checkpoints\/` * AIP_TENSORBOARD_LOG_DIR =
            `\/\/logs\/`
        worker_pool_specs (List[Dict]): Required. The spec of the worker pools
          including machine type and Docker image. All worker pools except the
          first one are optional and can be skipped by providing an empty value.
        study_spec_metrics: (List[Dict]): Required. List serialized from
          dictionary representing the metrics to optimize. The dictionary key is
          the metric_id, which is reported by your training job, and the
          dictionary value is the optimization goal of the metric ('minimize' or
          'maximize'). example: metrics =
          hyperparameter_tuning_job.serialize_metrics( {'loss': 'minimize',
          'accuracy': 'maximize'})
        study_spec_parameters (list[str]): Required. List serialized from the
          parameter dictionary. The dictionary represents parameters to
          optimize. The dictionary key is the parameter_id, which is passed into
          your training job as a command line key word argument, and the
          dictionary value is the parameter specification of the metric. from
          google.cloud.aiplatform.aiplatform import hyperparameter_tuning as hpt
          from
          google_cloud_pipeline_components.google_cloud_pipeline_components.v1
          import hyperparameter_tuning_job parameters =
          hyperparameter_tuning_job.serialize_parameters({ 'lr':
          hpt.DoubleParameterSpec(min=0.001, max=0.1, scale='log'), 'units':
          hpt.IntegerParameterSpec(min=4, max=128, scale='linear'),
          'activation': hpt.CategoricalParameterSpec(values=['relu', 'selu']),
          'batch_size': hpt.DiscreteParameterSpec(values=[128, 256],
          scale='linear') }) Supported parameter specifications can be found in
          aiplatform.hyperparameter_tuning. These parameter specification are
          currently supported: DoubleParameterSpec, IntegerParameterSpec,
          CategoricalParameterSpace, DiscreteParameterSpec
        max_trial_count (int): Required. The desired total number of Trials.
        parallel_trial_count (int): Required. The desired number of Trials to
          run in parallel.
        max_failed_trial_count (Optional[int]): The number of failed Trials that
          need to be seen before failing the HyperparameterTuningJob. If set to
          0, Vertex AI decides how many Trials must fail before the whole job
          fails.
        location (Optional[str]): Location to run the HyperparameterTuningJob
          in, defaults to "us-central1"
        study_spec_algorithm (Optional[str]): The search algorithm specified for
          the Study. Accepts one of the following: * `ALGORITHM_UNSPECIFIED` -
          If you do not specify an algorithm, your job uses the default Vertex
          AI algorithm. The default algorithm applies Bayesian optimization to
          arrive at the optimal solution with a more effective search over the
          parameter space. * 'GRID_SEARCH' - A simple grid search within the
          feasible space. This option is particularly useful if you want to
          specify a quantity of trials that is greater than the number of points
          in the feasible space. In such cases, if you do not specify a grid
          search, the Vertex AI default algorithm may generate duplicate
          suggestions. To use grid search, all parameter specs must be of type
          `IntegerParameterSpec`, `CategoricalParameterSpace`, or
          `DiscreteParameterSpec`. * 'RANDOM_SEARCH' - A simple random search
          within the feasible space.
        study_spec_measurement_selection_type (Optional[str]): This indicates
          which measurement to use if/when the service automatically selects the
          final measurement from previously reported intermediate measurements.
            Accepts: 'BEST_MEASUREMENT', 'LAST_MEASUREMENT' Choose this based on
              two considerations: A) Do you expect your measurements to
              monotonically improve? If so, choose 'LAST_MEASUREMENT'. On the
              other hand, if you're in a situation where your system can
              "over-train" and you expect the performance to get better for a
              while but then start declining, choose 'BEST_MEASUREMENT'. B) Are
              your measurements significantly noisy and/or irreproducible? If
              so, 'BEST_MEASUREMENT' will tend to be over-optimistic, and it may
              be better to choose 'LAST_MEASUREMENT'. If both or neither of (A)
              and (B) apply, it doesn't matter which selection type is chosen.
        encryption_spec_key_name (Optional[str]): Customer-managed encryption
          key options for a HyperparameterTuningJob. If this is set, then all
          resources created by the HyperparameterTuningJob will be encrypted
          with the provided encryption key.  Has the form:
          ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
          The key needs to be in the same region as where the compute resource
          is created.
        service_account (Optional[str]): Specifies the service account for
          workload run-as account. Users submitting jobs must have act-as
          permission on this run-as account.
        network (Optional[str]): The full name of the Compute Engine network to
          which the job should be peered. For example,
          projects/12345/global/networks/myVPC. Private services access must
          already be configured for the network. If left unspecified, the job is
          not peered with any network.

    Returns:
        Serialized gcp_resources proto tracking the Hyperparameter Tuning job.
            For more details, see
            https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-u', '-m',
          'google_cloud_pipeline_components.container.v1.hyperparameter_tuning_job.launcher'
      ],
      args=[
          '--type',
          'HyperparameterTuningJob',
          '--payload',
          ConcatPlaceholder([
              '{', '"display_name": "', display_name, '"', ', "study_spec": {',
              '"metrics": ', study_spec_metrics, ', "parameters": ',
              study_spec_parameters, ', "algorithm": "', study_spec_algorithm,
              '"', ', "measurement_selection_type": "',
              study_spec_measurement_selection_type, '"', '}',
              ', "max_trial_count": ', max_trial_count,
              ', "parallel_trial_count": ', parallel_trial_count,
              ', "max_failed_trial_count": ', max_failed_trial_count,
              ', "trial_job_spec": {', '"worker_pool_specs": ',
              worker_pool_specs, ', "service_account": "', service_account, '"',
              ', "network": "', network, '"', ', "base_output_directory": {',
              '"output_uri_prefix": "', base_output_directory, '"}', '}',
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name, '"}', '}'
          ]),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
      ])
