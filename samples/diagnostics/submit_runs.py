"""Script to submit diagnostic test pipeline runs to Kubeflow Pipelines."""

import kfp


def submit_diagnostic_runs():
    client = kfp.Client(host='http://127.0.0.1:8888')

    # Configure multi-user authentication header
    for api in [
        client._experiment_api,
        client._healthz_api,
        client._pipelines_api,
        client._recurring_run_api,
        client._run_api,
        client._upload_api,
    ]:
        api.api_client.default_headers['kubeflow-userid'] = 'user@example.com'

    namespace = 'user'

    # 1. Submit ImagePullBackOff Pipeline
    print("Submitting ImagePullBackOff test pipeline...")
    try:
        run1 = client.create_run_from_pipeline_package(
            pipeline_file='samples/diagnostics/image_pull_failure_pipeline.yaml',
            arguments={},
            run_name='poc-diag-image-pull-backoff',
            namespace=namespace,
        )
        print(f"✅ ImagePullBackOff run submitted successfully: Run ID = {run1.run_id}")
    except Exception as e:
        print(f"❌ Failed to submit ImagePullBackOff run: {e}")

    # 2. Submit OOMKilled Pipeline
    print("\nSubmitting OOMKilled test pipeline...")
    try:
        run2 = client.create_run_from_pipeline_package(
            pipeline_file='samples/diagnostics/oom_killed_pipeline.yaml',
            arguments={},
            run_name='poc-diag-oom-killed',
            namespace=namespace,
        )
        print(f"✅ OOMKilled run submitted successfully: Run ID = {run2.run_id}")
    except Exception as e:
        print(f"❌ Failed to submit OOMKilled run: {e}")


if __name__ == '__main__':
    submit_diagnostic_runs()
