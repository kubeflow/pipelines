import json

from kfp.v2.google.client import AIPlatformClient
from kfp.components import create_component_from_func_v2
from google.cloud import aiplatform
from google_cloud_pipeline_components.aiplatform import utils
from typing import NamedTuple, Optional, Union, Sequence, Dict, List


def hyperparameter_tuning_job(
    display_name: str,
    metric_spec: Dict[str, str],
    parameter_spec: Dict[str, Any],
    max_trial_count: int,
    parallel_trial_count: int,
    max_failed_trial_count: int = 0,
    project: str,
    location: str,
    staging_bucket: str,
    worker_pool_specs: List[Dict],
    search_algorithm: Optional[str] = None,
    measurement_selection: Optional[str] = "best",
    encryption_spec_key_name: Optional[str] = None,
    base_output_dir: str,
    local_script_kwargs: Optional[Dict],
) -> NamedTuple('Outputs', [
    ("trials", list),
]):
    """
    Creates a Google Cloud AI Plaform HyperparameterTuning Job.

    For more information on using hyperparameter tuning please visit:
    https://cloud.google.com/ai-platform-unified/docs/training/using-hyperparameter-tuning

    Args:
        display_name (str):
            Required. The user-defined name of the HyperparameterTuningJob.
            The name can be up to 128 characters long and can be consist
            of any UTF-8 characters.
        metric_spec: Dict[str, str]
            Required. Dicionary representing metrics to optimize. The dictionary key is the metric_id,
            which is reported by your training job, and the dictionary value is the
            optimization goal of the metric('minimize' or 'maximize'). example:
            metric_spec = {'loss': 'minimize', 'accuracy': 'maximize'}
        parameter_spec (Dict[str, hyperparameter_tuning._ParameterSpec]):
            Required. Dictionary representing parameters to optimize. The dictionary key is the metric_id,
            which is passed into your training job as a command line key word argument, and the
            dictionary value is the parameter specification of the metric.
            from google.cloud.aiplatform import hyperparameter_tuning as hpt
            parameter_spec={
                'decay': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear'),
                'learning_rate': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear')
                'batch_size': hpt.DiscreteParamterSpec(values=[4, 8, 16, 32, 64, 128], scale='linear')
            }
            Supported parameter specifications can be found until aiplatform.hyperparameter_tuning.
            These parameter specification are currently supported:
            DoubleParameterSpec, IntegerParameterSpec, CategoricalParameterSpace, DiscreteParameterSpec
        max_trial_count (int):
            Reuired. The desired total number of Trials.
        parallel_trial_count (int):
            Required. The desired number of Trials to run in parallel.
        max_failed_trial_count (int):
            Optional. The number of failed Trials that need to be
            seen before failing the HyperparameterTuningJob.
            If set to 0, Vertex AI decides how many Trials
            must fail before the whole job fails.
        project (str):
            Required. Project to run the HyperparameterTuningJob in.
        location (str):
            Required. Location to run the HyperparameterTuningJob in.
        staging_bucket (str):
            Required. Bucket for produced HyperparameterTuningJob artifacts.
        worker_pool_specs (List[Dict]):
            Required. The spec of the worker pools including machine type and
            Docker image, as a list of dictionaries.
        search_algorithm (str):
            The search algorithm specified for the Study.
            Accepts one of the following:
                `None` - If you do not specify an algorithm, your job uses
                the default Vertex AI algorithm. The default algorithm
                applies Bayesian optimization to arrive at the optimal
                solution with a more effective search over the parameter space.
                'grid' - A simple grid search within the feasible space. This
                option is particularly useful if you want to specify a quantity
                of trials that is greater than the number of points in the
                feasible space. In such cases, if you do not specify a grid
                search, the Vertex AI default algorithm may generate duplicate
                suggestions. To use grid search, all parameter specs must be
                of type `IntegerParameterSpec`, `CategoricalParameterSpace`,
                or `DiscreteParameterSpec`.
                'random' - A simple random search within the feasible space.
        measurement_selection (str):
            This indicates which measurement to use if/when the service
            automatically selects the final measurement from previously reported
            intermediate measurements.
            Accepts: 'best', 'last'
            Choose this based on two considerations:
            A) Do you expect your measurements to monotonically improve? If so,
            choose 'last'. On the other hand, if you're in a situation
            where your system can "over-train" and you expect the performance to
            get better for a while but then start declining, choose
            'best'. B) Are your measurements significantly noisy
            and/or irreproducible? If so, 'best' will tend to be
            over-optimistic, and it may be better to choose 'last'. If
            both or neither of (A) and (B) apply, it doesn't matter which
            selection type is chosen.
        encryption_spec_key_name (str):
            Optional. Customer-managed encryption key options for a
            HyperparameterTuningJob. If this is set, then
            all resources created by the
            HyperparameterTuningJob will be encrypted with
            the provided encryption key.
        base_output_dir (str):
            Optional. GCS output directory of job. If not provided a
            timestamped directory in the staging directory will be used.
        local_script_kwargs (Dict):
            Optional. Kwargs for creating a HyperparameterTuningJob from a
            local script.
            Keyword Args:
                script_path (str):
                    Required. Local path to training script.
                container_uri (str):
                    Required: Uri of the training container image to use for custom job.
                args (Optional[List[Union[str, float, int]]]):
                    Optional. Command line arguments to be passed to the Python task.
                requirements (Sequence[str]):
                    Optional. List of python packages dependencies of script.
                environment_variables (Dict[str, str]):
                    Optional. Environment variables to be passed to the container.
                    Should be a dictionary where keys are environment variable names
                    and values are environment variable values for those names.
                    At most 10 environment variables can be specified.
                    The Name of the environment variable must be unique.
                    environment_variables = {
                        'MY_KEY': 'MY_VALUE'
                    }
                replica_count (int):
                    Optional. The number of worker replicas. If replica count = 1 then one chief
                    replica will be provisioned. If replica_count > 1 the remainder will be
                    provisioned as a worker replica pool.
                machine_type (str):
                    Optional. The type of machine to use for training.
                accelerator_type (str):
                    Optional. Hardware accelerator type. One of ACCELERATOR_TYPE_UNSPECIFIED,
                    NVIDIA_TESLA_K80, NVIDIA_TESLA_P100, NVIDIA_TESLA_V100, NVIDIA_TESLA_P4,
                    NVIDIA_TESLA_T4
                accelerator_count (int):
                    Optional. The number of accelerators to attach to a worker replica.

    Returns:
        List of HyperparameterTuningJob trials
    """

    aiplatform.init(project=project, location=location,
                staging_bucket=staging_bucket)

    deserializer = utils.get_deserializer(dict)
    parameter_spec_kwargs = deserializer(parameter_spec)

    custom_job_display_name = display_name + '_custom_job'

    if local_script_kwargs is None:
        job = aiplatform.CustomJob(
            display_name=custom_job_display_name,
            base_output_dir=base_output_dir,
            worker_pool_specs=worker_pool_specs,
        )
    else:
        job = aiplatform.CustomJob.from_local_script(
            display_name=custom_job_display_name,
            script_path=local_script_kwargs.get('script_path'),
            container_uri=local_script_kwargs.get('container_uri'),
            args=local_script_kwargs.get('args'),
            requirements=local_script_kwargs.get('requirements'),
            environment_variables=local_script_kwargs.get('environment_variables'),
            replica_count=local_script_kwargs.get('replica_count'),
            machine_type=local_script_kwargs.get('machine_type'),
            accelerator_type=local_script_kwargs.get('accelerator_type'),
            accelerator_count=local_script_kwargs.get('accelerator_count'),
            base_output_dir=base_output_dir,
            encryption_spec_key_name=encryption_spec_key_name,
            staging_bucket=staging_bucket
        )

    hp_job = aiplatform.HyperparameterTuningJob(
        display_name=display_name,
        custom_job=job,
        metric_spec=metric_spec,
        parameter_spec={
            **parameter_spec_kwargs
        },
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        max_failed_trial_count=max_failed_trial_count,
        search_algorithm=search_algorithm,
        measurement_selection=measurement_selection,
        encryption_spec_key_name=encryption_spec_key_name
    )

    hp_job.run()

    trials = [json.dumps(trial) for trial in hp_job.trials]

    return trials


if __name__ == '__main__':
    hyperparameter_tuning_job_op = create_component_from_func_v2(
        hyperparameter_tuning_job,
        base_image='python:3.8',
        packages_to_install=['google-cloud-aiplatform', 'kfp', 'google-cloud-pipeline-components'],
        output_component_file='component.yaml',
    )
