import time
import os
from kfp import Client , dsl , compiler

# Kubeflow Pipelines API URL
KFP_ENDPOINT = os.environ['KFP_ENDPOINT']
KFP_CLIENT = Client(host=KFP_ENDPOINT)

# Pipeline and Experiment Details
EXPERIMENT_NAME = "scheduled-test-experiment"
PIPELINE_NAME = "hello-world"
NAMESPACE = "kubeflow"  

@dsl.component
def say_hello(name: str) -> str:
    hello_text = f'Hello, {name}!'
    print(hello_text)
    return hello_text

@dsl.pipeline
def hello_pipeline(recipient: str) -> str:
    hello_task = say_hello(name=recipient)
    return hello_task.output

def compile_pipeline():
    compiler.Compiler().compile(hello_pipeline, 'pipeline.yaml' , pipeline_name=PIPELINE_NAME)

def create_experiment():
    """Creates an experiment if it doesn't exist."""
    experiment = KFP_CLIENT.create_experiment(name=EXPERIMENT_NAME, namespace=NAMESPACE)
    return experiment.experiment_id

def schedule_pipeline():
    """Schedules the pipeline to run every minute."""
    experiment_id = create_experiment()

    trigger = {
        "start_time": None,
        "end_time": None,
        "cron_schedule": "*/1 * * * *",  # Every minute
        "enabled": True
    }

    run_name = f"{PIPELINE_NAME}-schedule"
    pipeline_params = {
        "recipient": "World" 
    }
    response = KFP_CLIENT.create_recurring_run(
        pipeline_package_path="pipeline.yaml",
        experiment_id=experiment_id,
        job_name=run_name,
        start_time=trigger.get("start_time"),
        end_time=trigger.get("end_time"),
        cron_expression=trigger.get("cron_schedule"),
        enabled=trigger.get("enabled"),
        params=pipeline_params
    )
    print(f"Scheduled pipeline: {response.recurring_run_id}")
    return response.recurring_run_id

def get_successful_runs():
    """Fetches successful runs for the experiment."""
    experiment_id = create_experiment()
    runs = KFP_CLIENT.list_runs(experiment_id=experiment_id).runs
    successful_runs = []
    for run in runs:
        print(f"run id {run.run_id} has state {run.state}")
        if run.state == "SUCCEEDED":
            successful_runs.append(run)
    return successful_runs

def wait_and_verify():
    """Waits for 4 minutes and checks if at least 2 runs succeeded."""
    print("Waiting for scheduled runs to execute...")
    time.sleep(240)  # Wait for 4 minutes

    successful_runs = get_successful_runs()
    assert len(successful_runs) >= 2, f"Expected at least 2 successful runs, but got {len(successful_runs)}"
    print("Test passed: At least 2 runs succeeded.")

if __name__ == "__main__":
    compile_pipeline()
    schedule_pipeline()
    wait_and_verify()
