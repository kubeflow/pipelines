import kfp
import os
from typing import Dict, List, Callable, Optional
from dataclasses import dataclass

MINUTE = 60


def _default_verify_func(run_id, run):
    assert run.status == 'Succeeded'


@dataclass
class TestCase:
    '''Test case for running a KFP sample'''
    pipeline_func: Callable
    mode: kfp.dsl.PipelineExecutionMode = kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
    arguments: Optional[Dict[str, str]] = None
    verify_func: Callable = _default_verify_func


def run_pipeline_func(test_cases: List[TestCase]):
    """Run a pipeline function and wait for its result.

    :param pipeline_func: pipeline function to run
    :type pipeline_func: function
    """

    def test_wrapper(run_pipeline):
        for case in test_cases:
            result = run_pipeline(
                pipeline_func=case.pipeline_func,
                mode=case.mode,
                arguments=case.arguments or {}
            )
            case.verify_func(run=result, run_id=result.id)

    _run_test(test_wrapper)


def _run_test(callback):

    def main(
        output_directory: Optional[str] = None,  # example
        host: Optional[str] = None,
        external_host: Optional[str] = None,
        launcher_image: Optional['URI'] = None,
        experiment: str = 'v2_sample_test_samples',
    ):
        """Test file CLI entrypoint used by Fire.

        :param host: Hostname pipelines can access, defaults to 'http://ml-pipeline:8888'.
        :type host: str, optional
        :param external_host: External hostname users can access from their browsers.
        :type external_host: str, optional
        :param output_directory: pipeline output directory that holds intermediate
        artifacts, example gs://your-bucket/path/to/workdir.
        :type output_directory: str, optional
        :param launcher_image: override launcher image, only used in V2_COMPATIBLE mode
        :type launcher_image: URI, optional
        :param experiment: experiment the run is added to, defaults to 'v2_sample_test_samples'
        :type experiment: str, optional
        """

        # Default to env values, so people can set up their env and run these
        # tests without specifying any commands.
        if host is None:
            host = os.getenv('KFP_HOST', 'http://ml-pipeline:8888')
        if external_host is None:
            external_host = host
        if output_directory is None:
            output_directory = os.getenv('KFP_OUTPUT_DIRECTORY')

        client = kfp.Client(host=host)

        def run_pipeline(
            pipeline_func: Callable,
            mode: kfp.dsl.PipelineExecutionMode = kfp.dsl.PipelineExecutionMode.
            V2_COMPATIBLE,
            arguments: dict = {},
        ):
            run_result = client.create_run_from_pipeline_func(
                pipeline_func,
                mode=mode,
                arguments={
                    kfp.dsl.ROOT_PARAMETER_NAME: output_directory,
                    **arguments
                },
                launcher_image=launcher_image,
                experiment_name=experiment,
            )
            print("Run details page URL:")
            print(f"{external_host}/#/runs/details/{run_result.run_id}")
            run_response = run_result.wait_for_run_completion(10 * MINUTE)
            run = run_response.run
            from pprint import pprint
            # Hide detailed information
            run.pipeline_spec.workflow_manifest = None
            pprint(run)
            return run

        callback(run_pipeline=run_pipeline)

    import fire
    fire.Fire(main)
