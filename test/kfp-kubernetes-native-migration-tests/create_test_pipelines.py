# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json
import time
import requests
import subprocess
import sys
import os
from pathlib import Path
import kfp
import kfp.dsl as dsl
from kfp.client import Client


KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')

# Define simple pipeline functions for testing
@dsl.container_component
def print_hello_op():    
    return dsl.ContainerSpec(
        image="python:3.9",
        command=["python", "-c"],
        args=["print('Hello World')"]
    )

@dsl.container_component
def data_preprocessing_op():
    """Component for data preprocessing.""" 
    return dsl.ContainerSpec(
        image="python:3.9",
        command=["python", "-c"],
        args=["print('Processing data')"]
    )

@dsl.container_component
def model_training_op():
    """Component for model training."""
    return dsl.ContainerSpec(
        image="python:3.9",
        command=["python", "-c"],
        args=["print('Training model')"]
    )

@dsl.container_component
def model_evaluation_op():
    """Component for model evaluation."""
    return dsl.ContainerSpec(
        image="python:3.9",
        command=["python", "-c"],
        args=["print('Evaluating model')"]
    )

@dsl.pipeline(name="simple-pipeline")
def simple_pipeline(message: str = "Hello World"):
    """A simple test pipeline."""
    print_hello_op()

@dsl.pipeline(name="complex-pipeline")
def complex_pipeline(input_data: str = "test_data"):
    """A complex test pipeline with multiple components."""
    preprocessing_task = data_preprocessing_op()
    training_task = model_training_op()
    training_task.after(preprocessing_task)

@dsl.pipeline(name="complex-pipeline-v2")
def complex_pipeline_v2(input_data: str = "test_data"):
    """A complex test pipeline with multiple components including evaluation."""
    preprocessing_task = data_preprocessing_op()
    training_task = model_training_op()
    evaluation_task = model_evaluation_op()
    training_task.after(preprocessing_task)
    evaluation_task.after(training_task)

def create_pipeline(name, description, pipeline_func):
    """Create a pipeline in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    
    pipeline = client.upload_pipeline_from_pipeline_func(
        pipeline_func=pipeline_func,
        pipeline_name=name,
        description=description
    )
    
    return {
        "id": pipeline.pipeline_id,
        "name": pipeline.display_name,
        "description": description
    }

def create_pipeline_version(pipeline_id, name, pipeline_func):
    """Create a pipeline version in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    
    version = client.upload_pipeline_version_from_pipeline_func(
        pipeline_func=pipeline_func,
        pipeline_version_name=name,
        pipeline_id=pipeline_id
    )
    
    return {
        "id": version.pipeline_version_id,
        "name": version.display_name,
        "pipeline_id": pipeline_id
    }

def create_experiment(name, description):
    """Create an experiment in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    
    # Create experiment using KFP client
    experiment = client.create_experiment(
        name=name,
        description=description
    )
    experiment_id = getattr(experiment, 'experiment_id', None)
    experiment_name = getattr(experiment, 'display_name', None)
    
    return {
        "id": experiment_id,
        "name": experiment_name,
        "description": description
    }

def create_run(experiment_id, pipeline_id, pipeline_version_id, name, parameters=None):
    """Create a pipeline run in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    run_params = {}
    if parameters:
        for param in parameters:
            if isinstance(param, dict) and "name" in param and "value" in param:
                run_params[param["name"]] = param["value"]       
   
    run_data = client.run_pipeline(
        experiment_id=experiment_id,
        job_name=name,
        pipeline_id=pipeline_id,
        version_id=pipeline_version_id,
        params=run_params
    )
    run_id = getattr(run_data, 'run_id', None)
    run_name = getattr(run_data, 'display_name', None)
    
    return {
        "id": run_id,
        "name": run_name,
        "pipeline_spec": {
            "pipeline_id": pipeline_id,
            "pipeline_version_id": pipeline_version_id
        },
        "resource_references": [
            {
                "key": {
                    "type": "EXPERIMENT",
                    "id": experiment_id
                },
                "relationship": "OWNER"
            }
        ],
        "parameters": parameters
    }

def create_recurring_run(experiment_id, pipeline_id, pipeline_version_id, name, cron_expression, parameters=None):
    """Create a recurring run in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    run_params = {}
    if parameters:
        for param in parameters:
            if isinstance(param, dict) and "name" in param and "value" in param:
                run_params[param["name"]] = param["value"]
    recurring_run_data = client.create_recurring_run(
        experiment_id=experiment_id,
        job_name=name,
        pipeline_id=pipeline_id,
        version_id=pipeline_version_id,
        cron_expression=cron_expression,
        params=run_params
    )       
    recurring_run_id = getattr(recurring_run_data, 'recurring_run_id', None)
    recurring_run_name = getattr(recurring_run_data, 'display_name', None)
    
    return {
        "id": recurring_run_id,
        "name": recurring_run_name,
        "pipeline_spec": {
            "pipeline_id": pipeline_id,
            "pipeline_version_id": pipeline_version_id
        },
        "resource_references": [
            {
                "key": {
                    "type": "EXPERIMENT",
                    "id": experiment_id
                },
                "relationship": "OWNER"
            }
        ],
        "trigger": {
            "cron_schedule": {
                "cron": cron_expression
            }
        },
        "parameters": parameters
    }

def main():   
    print("Setting up test environment for KFP migration tests...")    
   
    test_data = {
        "pipelines": [],
        "experiments": [],
        "runs": [],
        "recurring_runs": []
    }
    
    # Create simple pipeline
    pipeline1 = create_pipeline("simple-pipeline", "A simple test pipeline", simple_pipeline)
    test_data["pipelines"].append(pipeline1)
    print(f"Created pipeline: {pipeline1['name']} (ID: {pipeline1['id']})")
    
    # Create pipeline version
    version1 = create_pipeline_version(pipeline1["id"], "v1", simple_pipeline)
    test_data["pipelines"].append(version1)
    print(f"Created pipeline version: {version1['name']} (ID: {version1['id']})")
    
    # Create pipeline with multiple versions and different specs
    pipeline2 = create_pipeline("complex-pipeline", "A complex test pipeline", complex_pipeline)
    test_data["pipelines"].append(pipeline2)
    print(f"Created pipeline: {pipeline2['name']} (ID: {pipeline2['id']})")
    
    # Create two versions with different specs
    version2_1 = create_pipeline_version(pipeline2["id"], "v1", complex_pipeline)
    test_data["pipelines"].append(version2_1)
    print(f"Created pipeline version: {version2_1['name']} (ID: {version2_1['id']})")
    
    version2_2 = create_pipeline_version(pipeline2["id"], "v2", complex_pipeline_v2)
    test_data["pipelines"].append(version2_2)
    print(f"Created pipeline version: {version2_2['name']} (ID: {version2_2['id']})")
    
    # Create experiment
    experiment = create_experiment("migration-test-experiment", "Test experiment for migration")
    test_data["experiments"].append(experiment)
    print(f"Created experiment: {experiment['name']} (ID: {experiment['id']})")
    
    # Create a run in the experiment
    run = create_run(
        experiment["id"],
        pipeline1["id"],
        version1["id"],
        "test-run",
        parameters=None
    )
    test_data["runs"].append(run)
    print(f"Created run: {run['name']} (ID: {run['id']})")
    
    # Create a recurring run in the experiment
    recurring_run = create_recurring_run(
        experiment["id"],
        pipeline2["id"],
        version2_1["id"],
        "test-recurring-run",
        "0 0 * * *",  
        parameters=None 
    )
    test_data["recurring_runs"].append(recurring_run)
    print(f"Created recurring run: {recurring_run['name']} (ID: {recurring_run['id']})")
    
    # Save test data for later use
    migration_test_data_file = Path("migration_test_data.json")
    with open(migration_test_data_file, "w") as f:
        json.dump(test_data, f, indent=2)
    
    print(f"\nTest data saved to {migration_test_data_file}")
    print("Test environment setup complete!")    
   
    print(f"\nCreated:")
    print(f"- {len([p for p in test_data['pipelines'] if 'pipeline_id' not in p])} pipelines")
    print(f"- {len([p for p in test_data['pipelines'] if 'pipeline_id' in p])} pipeline versions")
    print(f"- {len(test_data['experiments'])} experiments")
    print(f"- {len(test_data['runs'])} runs")
    print(f"- {len(test_data['recurring_runs'])} recurring runs")

if __name__ == "__main__":
    main()
