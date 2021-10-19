# pytype: disable=annotation-type-mismatch
from kfp import components
from typing import NamedTuple
from google.cloud.aiplatform_v1.types import study

def hyperparameter_tuning_job_run_op(
    display_name: str,
    project: str,
    base_output_directory: str,
    worker_pool_specs: list,
    metrics: dict,
    parameters: list,
    max_trial_count: int,
    parallel_trial_count: int,
    max_failed_trial_count: int = 0,
    location: str = "us-central1",
    algorithm: str = "ALGORITHM_UNSPECIFIED",
    measurement_selection_type: str = "BEST_MEASUREMENT",
    encryption_spec_key_name: str = None,  # pytype: disable=annotation-type-mismatch
    service_account: str = None,  # pytype: disable=annotation-type-mismatch
    network: str = None,  # pytype: disable=annotation-type-mismatch
) -> NamedTuple('Outputs', [
    ("trials", list),
]):
    """
    Creates a Google Cloud AI Plaform HyperparameterTuning Job.

    Example usage:
    ```
    from google.cloud.aiplatform import hyperparameter_tuning as hpt
    from google_cloud_pipeline_components.experimental import hyperparameter_tuning_job
    from kfp.v2 import dsl

    worker_pool_specs = [
            {
                "machine_spec": {
                    "machine_type": "n1-standard-4",
                    "accelerator_type": "NVIDIA_TESLA_K80",
                    "accelerator_count": 1,
                },
                "replica_count": 1,
                "container_spec": {
                    "image_uri": container_image_uri,
                    "command": [],
                    "args": [],
                },
            }
        ]
    parameters = hyperparameter_tuning_job.serialize_parameters({
        'lr': hpt.DoubleParameterSpec(min=0.001, max=0.1, scale='log'),
        'units': hpt.IntegerParameterSpec(min=4, max=128, scale='linear'),
        'activation': hpt.CategoricalParameterSpec(values=['relu', 'selu']),
        'batch_size': hpt.DiscreteParameterSpec(values=[128, 256], scale='linear')
    })


    @dsl.pipeline(pipeline_root='', name='sample-pipeline')
    def pipeline():
        hp_tuning_task = hyperparameter_tuning_job.HyperparameterTuningJobRunOp(
            display_name='hp-job',
            project='my-project',
            location='us-central1',
            worker_pool_specs=worker_pool_specs,
            metrics={
                'loss': 'minimize',
            },
            parameter_spec=parameters,
            max_trial_count=128,
            parallel_trial_count=8,
        )

    ```

    For more information on using hyperparameter tuning please visit:
    https://cloud.google.com/ai-platform-unified/docs/training/using-hyperparameter-tuning

    Args:
        display_name (str):
            Required. The user-defined name of the HyperparameterTuningJob.
            The name can be up to 128 characters long and can be consist
            of any UTF-8 characters.
        project (str):
            Required. Project to run the HyperparameterTuningJob in.
        base_output_directory (str):
            Required. Bucket for produced HyperparameterTuningJob artifacts.
        worker_pool_specs (List[Dict]):
            Required. The spec of the worker pools including machine type and
            Docker image, as a list of dictionaries.
        metrics: (Dict[str, str]):
            Required. Dictionary representing metrics to optimize. The dictionary key is the metric_id,
            which is reported by your training job, and the dictionary value is the
            optimization goal of the metric('minimize' or 'maximize'). example:
            metrics = {'loss': 'minimize', 'accuracy': 'maximize'}
        parameters (list[str]):
            Required. List serialized from the parameter dictionary. The dictionary
            represents parameters to optimize. The dictionary key is the metric_id,
            which is passed into your training job as a command line key word argument, and the
            dictionary value is the parameter specification of the metric.
            from google.cloud.aiplatform import hyperparameter_tuning as hpt
            from google_cloud_pipeline_components.experimental import hyperparameter_tuning_job
            parameters = hyperparameter_tuning_job.serialize_parameters({
                'lr': hpt.DoubleParameterSpec(min=0.001, max=0.1, scale='log'),
                'units': hpt.IntegerParameterSpec(min=4, max=128, scale='linear'),
                'activation': hpt.CategoricalParameterSpec(values=['relu', 'selu']),
                'batch_size': hpt.DiscreteParameterSpec(values=[128, 256], scale='linear')
            })
            Supported parameter specifications can be found in aiplatform.hyperparameter_tuning.
            These parameter specification are currently supported:
            DoubleParameterSpec, IntegerParameterSpec, CategoricalParameterSpace, DiscreteParameterSpec
        max_trial_count (int):
            Required. The desired total number of Trials.
        parallel_trial_count (int):
            Required. The desired number of Trials to run in parallel.
        max_failed_trial_count (int):
            Optional. The number of failed Trials that need to be
            seen before failing the HyperparameterTuningJob.
            If set to 0, Vertex AI decides how many Trials
            must fail before the whole job fails.
        location (str):
            Optional. Location to run the HyperparameterTuningJob in, defaults
            to "us-central1"
        algorithm (str):
            The search algorithm specified for the Study.
            Accepts one of the following:
                `ALGORITHM_UNSPECIFIED` - If you do not specify an algorithm,
                your job uses the default Vertex AI algorithm. The default
                algorithm applies Bayesian optimization to arrive at the optimal
                solution with a more effective search over the parameter space.
                'GRID_SEARCH' - A simple grid search within the feasible space.
                This option is particularly useful if you want to specify a
                quantity of trials that is greater than the number of points in
                the feasible space. In such cases, if you do not specify a grid
                search, the Vertex AI default algorithm may generate duplicate
                suggestions. To use grid search, all parameter specs must be
                of type `IntegerParameterSpec`, `CategoricalParameterSpace`,
                or `DiscreteParameterSpec`.
                'RANDOM_SEARCH' - A simple random search within the feasible
                space.
        measurement_selection_type (str):
            This indicates which measurement to use if/when the service
            automatically selects the final measurement from previously reported
            intermediate measurements.
            Accepts: 'BEST_MEASUREMENT', 'LAST_MEASUREMENT'
            Choose this based on two considerations:
            A) Do you expect your measurements to monotonically improve? If so,
            choose 'LAST_MEASUREMENT'. On the other hand, if you're in a situation
            where your system can "over-train" and you expect the performance to
            get better for a while but then start declining, choose
            'BEST_MEASUREMENT'. B) Are your measurements significantly noisy
            and/or irreproducible? If so, 'BEST_MEASUREMENT' will tend to be
            over-optimistic, and it may be better to choose 'LAST_MEASUREMENT'. If
            both or neither of (A) and (B) apply, it doesn't matter which
            selection type is chosen.
        encryption_spec_key_name (str):
            Optional. Customer-managed encryption key options for a
            HyperparameterTuningJob. If this is set, then
            all resources created by the
            HyperparameterTuningJob will be encrypted with
            the provided encryption key.
        service_account (str):
            Optional. Specifies the service account for workload run-as account.
            Users submitting jobs must have act-as permission on this run-as account.
        network (str):
            Optional. The full name of the Compute Engine network to which the job
            should be peered. For example, projects/12345/global/networks/myVPC.
            Private services access must already be configured for the network.
            If left unspecified, the job is not peered with any network.
    Returns:
        List of HyperparameterTuningJob trials
    """
    from google.cloud import aiplatform
    from google.cloud.aiplatform import hyperparameter_tuning as hpt
    from google.cloud.aiplatform_v1.types import study
    from google.cloud.aiplatform.hyperparameter_tuning import _SCALE_TYPE_MAP

    # Reverse the _SCALE_TYPE_MAP dict for deserialization
    SCALE_MAP = dict((reversed(item) for item in _SCALE_TYPE_MAP.items()))

    PARAMETER_SPEC_MAP = {
        hpt.DoubleParameterSpec._parameter_spec_value_key: hpt.DoubleParameterSpec,
        hpt.IntegerParameterSpec._parameter_spec_value_key: hpt.IntegerParameterSpec,
        hpt.CategoricalParameterSpec._parameter_spec_value_key: hpt.CategoricalParameterSpec,
        hpt.DiscreteParameterSpec._parameter_spec_value_key: hpt.DiscreteParameterSpec,
    }

    ALGORITHM_MAP = {
        'ALGORITHM_UNSPECIFIED': None,
        'GRID_SEARCH': 'grid',
        'RANDOM_SEARCH': 'random',
    }

    MEASUREMENT_SELECTION_TYPE_MAP = {
        'BEST_MEASUREMENT': 'best',
        'LAST_MEASUREMENT': 'last',
    }

    aiplatform.init(project=project, location=location,
                staging_bucket=base_output_directory)

    # Deserialize the parameters
    parameters_kwargs = {}
    for parameter in parameters:
        param = study.StudySpec.ParameterSpec.from_json(parameter)
        parameter_id = param.parameter_id
        param_attrs = {}
        for parameter_spec_value_key, parameter_spec in PARAMETER_SPEC_MAP.items():
            if getattr(param, parameter_spec_value_key):
                attrs = getattr(param, parameter_spec_value_key)
                for parameter, value in parameter_spec._parameter_value_map:
                    if hasattr(attrs, value):
                        param_attrs[parameter] = getattr(attrs, value)
                # Detect 'scale' in list of arguments to parameter_spec.__init__
                param_spec_code = parameter_spec.__init__.__code__
                if 'scale' in param_spec_code.co_varnames[:param_spec_code.co_argcount]:
                    param_attrs['scale'] = SCALE_MAP[param.scale_type]
                parameters_kwargs[parameter_id] = parameter_spec(
                    **param_attrs)  # pytype: disable=wrong-keyword-args
                break

    custom_job_display_name = display_name + '_custom_job'

    job = aiplatform.CustomJob(
        display_name=custom_job_display_name,
        staging_bucket=base_output_directory,
        worker_pool_specs=worker_pool_specs,
    )

    hp_job = aiplatform.HyperparameterTuningJob(
        display_name=display_name,
        custom_job=job,
        metric_spec=metrics,
        parameter_spec={
            **parameters_kwargs
        },
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        max_failed_trial_count=max_failed_trial_count,
        search_algorithm=ALGORITHM_MAP[algorithm],
        measurement_selection=MEASUREMENT_SELECTION_TYPE_MAP[
            measurement_selection_type
        ],
        encryption_spec_key_name=encryption_spec_key_name
    )

    hp_job.run(
        service_account=service_account,
        network=network)

    trials = [study.Trial.to_json(trial) for trial in hp_job.trials]

    return trials  # pytype: disable=bad-return-type


def serialize_parameters(parameters: dict) -> list:
    """
    Serializes the hyperparameter tuning parameter spec.

    Args:
        parameters (Dict[str, hyperparameter_tuning._ParameterSpec]):
            Dictionary representing parameters to optimize. The dictionary key is the metric_id,
            which is passed into your training job as a command line key word argument, and the
            dictionary value is the parameter specification of the metric.
            from google.cloud.aiplatform import hyperparameter_tuning as hpt
            parameters={
                'decay': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear'),
                'learning_rate': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear')
                'batch_size': hpt.DiscreteParamterSpec(values=[4, 8, 16, 32, 64, 128], scale='linear')
            }
            Supported parameter specifications can be found until aiplatform.hyperparameter_tuning.
            These parameter specification are currently supported:
            DoubleParameterSpec, IntegerParameterSpec, CategoricalParameterSpace, DiscreteParameterSpec

    Returns:
        List containing an intermediate JSON representation of the parameter spec

    """
    return [
      study.StudySpec.ParameterSpec.to_json(
          parameter._to_parameter_spec(parameter_id=parameter_id))
          for parameter_id, parameter in parameters.items()
    ]

if __name__ == '__main__':
    HyperparameterTuningJobRunOp = components.create_component_from_func(
        hyperparameter_tuning_job_run_op,
        base_image='python:3.8',
        packages_to_install=['google-cloud-aiplatform', 'kfp'],
        output_component_file='component.yaml',
    )
