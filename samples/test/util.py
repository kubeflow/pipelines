import kfp

MINUTE = 60


def default_verify_func(run_id, run):
    assert run.status == 'Succeeded'


def run_pipeline_func(
    pipeline_func,
    verify_func=default_verify_func,
    mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
):
    """Run a pipeline function and wait for its result.

    :param pipeline_func: pipeline function to run
    :type pipeline_func: function
    """

    def main(
        output_directory: str,  # example gs://your-bucket/path/to/workdir
        host: str = 'http://ml-pipeline:8888',
        launcher_image: 'URI' = None,
        experiment: str = 'v2_sample_test_samples',
    ):
        """Test file CLI entrypoint used by Fire.

        :param output_directory: pipeline output directory that holds intermediate artifacts.
        :type output_directory: str
        :param launcher_image: override launcher image, only used in V2_COMPATIBLE mode
        :type launcher_image: URI, optional
        :param experiment: experiment the run is added to, defaults to 'v2_sample_test_samples'
        :type experiment: str, optional
        """
        client = kfp.Client(host=host)
        run_result = client.create_run_from_pipeline_func(
            pipeline_func,
            mode=mode,
            arguments={kfp.dsl.ROOT_PARAMETER_NAME: output_directory},
            launcher_image=launcher_image,
            experiment_name=experiment,
        )
        print("Run details page URL:")
        print(f"{host}/#/runs/details/{run_result.run_id}")
        run_response = run_result.wait_for_run_completion(10 * MINUTE)
        run = run_response.run
        from pprint import pprint
        pprint(run_response.run)
        print("Run details page URL:")
        print(f"{host}/#/runs/details/{run_result.run_id}")
        verify_func(run_id=run_result.run_id, run=run)

    import fire
    fire.Fire(main)
