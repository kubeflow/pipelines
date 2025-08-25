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
import pickle
import time
import requests
import subprocess
import sys
import os
from pathlib import Path
from datetime import datetime
import kfp
from kfp.client import Client


KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')

def to_json_for_comparison(obj):
    """Convert object to JSON string for comparison."""
    import datetime
    
    def convert_datetime_to_string(data):
        """Convert datetime objects to strings."""        
        if isinstance(data, dict):
            return {k: convert_datetime_to_string(v) for k, v in data.items()}
        elif isinstance(data, datetime.datetime):
            return data.isoformat()
        elif isinstance(data, list):
            return [convert_datetime_to_string(item) for item in data]
        else:
            return data    
    
    data = obj.to_dict()  
    data = convert_datetime_to_string(data)
    
    return json.dumps(data, sort_keys=True, indent=2)

# Pipeline files to use for testing
PIPELINE_FILES = [
    {
        "name": "hello-world", 
        "path": "test_data/pipeline_files/valid/critical/hello-world.yaml",
        "description": "Simple hello world pipeline for migration testing"
    },
    {
        "name": "add-numbers",
        "path": "test_data/pipeline_files/valid/add_numbers.yaml", 
        "description": "Simple arithmetic pipeline with parameters for migration testing"
    }
]

def create_pipeline(name, description, pipeline_file_path):
    """Create a pipeline in KFP Database mode from YAML file."""
    client = Client(host=KFP_ENDPOINT)

    # Upload the pipeline
    pipeline = client.upload_pipeline(
        pipeline_package_path=pipeline_file_path,
        pipeline_name=name,
        description=description
    )
    complete_pipeline = client.get_pipeline(pipeline.pipeline_id)

    return complete_pipeline

def create_pipeline_version(pipeline_id, name, pipeline_file_path):
    """Create a pipeline version in KFP Database mode from YAML file."""
    client = Client(host=KFP_ENDPOINT)

    # Upload the pipeline version
    version = client.upload_pipeline_version(
        pipeline_package_path=pipeline_file_path,
        pipeline_version_name=name,
        pipeline_id=pipeline_id
    )
    complete_version = client.get_pipeline_version(pipeline_id, version.pipeline_version_id)

    return complete_version

def create_experiment(name, description):
    """Create an experiment in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)

    # Create experiment using KFP client
    experiment = client.create_experiment(
        name=name,
        description=description
    )
    experiment_id = experiment.experiment_id
    complete_experiment = client.get_experiment(experiment_id)
   
    return complete_experiment

def create_run(experiment_id, pipeline_id, pipeline_version_id, name, parameters=None):
    """Create a pipeline run in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    run_params = {}
    if parameters:
        for param in parameters:
            if isinstance(param, dict) and "name" in param and "value" in param:
                run_params[param["name"]] = param["value"]       

    # Create the run
    run_data = client.run_pipeline(
        experiment_id=experiment_id,
        job_name=name,
        pipeline_id=pipeline_id,
        version_id=pipeline_version_id,
        params=run_params
    )
    run_id = run_data.run_id
    complete_run = client.get_run(run_id)

    return complete_run

def create_recurring_run(experiment_id, pipeline_id, pipeline_version_id, name, cron_expression, parameters=None):
    """Create a recurring run in KFP Database mode."""
    client = Client(host=KFP_ENDPOINT)
    run_params = {}
    if parameters:
        for param in parameters:
            if isinstance(param, dict) and "name" in param and "value" in param:
                run_params[param["name"]] = param["value"]

    # Create the recurring run
    recurring_run_data = client.create_recurring_run(
        experiment_id=experiment_id,
        job_name=name,
        pipeline_id=pipeline_id,
        version_id=pipeline_version_id,
        cron_expression=cron_expression,
        params=run_params
    )       
    recurring_run_id = recurring_run_data.recurring_run_id
   
    complete_recurring_run = client.get_recurring_run(recurring_run_id)
 
    return complete_recurring_run

def main():   
    print("Setting up test environment for KFP migration tests using existing pipeline files...")  

    test_data = {
        "pipelines": [],
        "experiments": [],
        "runs": [],
        "recurring_runs": []
    }

    # Verify pipeline files exist
    for pipeline_file in PIPELINE_FILES:
        file_path = Path(pipeline_file["path"])
        if not file_path.exists():
            print(f"Error: Pipeline file not found: {file_path}")
            sys.exit(1)
        print(f"Found pipeline file: {file_path}")

    # Create pipelines from existing YAML files
    pipeline1_config = PIPELINE_FILES[0]
    pipeline1 = create_pipeline(
        pipeline1_config["name"], 
        pipeline1_config["description"], 
        pipeline1_config["path"]
    )   
    test_data["pipelines"].append(pipeline1)
    
    # Create a version for the first pipeline
    version1 = create_pipeline_version(
        pipeline1.pipeline_id, 
        "v1", 
        pipeline1_config["path"]
    )    
    test_data["pipelines"].append(version1)
   
    # Create second pipeline from file
    pipeline2_config = PIPELINE_FILES[1]
    pipeline2 = create_pipeline(
        pipeline2_config["name"], 
        pipeline2_config["description"],
        pipeline2_config["path"]
    )    
    test_data["pipelines"].append(pipeline2)
    print(f"Created pipeline: {pipeline2.display_name} (ID: {pipeline2.pipeline_id})")

    # Create multiple versions for the second pipeline
    version2_v1 = create_pipeline_version(
        pipeline2.pipeline_id, 
        "v1", 
        pipeline2_config["path"]
    )    
    test_data["pipelines"].append(version2_v1)
    print(f"Created pipeline version: {version2_v1.display_name} (ID: {version2_v1.pipeline_version_id})")
    
    # Create second version with same spec to test multiple versions scenario
    version2_v2 = create_pipeline_version(
        pipeline2.pipeline_id, 
        "v2", 
        pipeline2_config["path"]
    )    
    test_data["pipelines"].append(version2_v2)
    print(f"Created pipeline version: {version2_v2.display_name} (ID: {version2_v2.pipeline_version_id})")
    
    # Create third version with different spec (using a different pipeline file)
    version2_v3 = create_pipeline_version(
        pipeline2.pipeline_id, 
        "v3", 
        "test_data/pipeline_files/valid/two_step_pipeline.yaml"
    )    
    test_data["pipelines"].append(version2_v3)
    print(f"Created pipeline version: {version2_v3.display_name} (ID: {version2_v3.pipeline_version_id})")

    # Create experiment
    experiment = create_experiment("migration-test-experiment", "Test experiment for migration")
   
    test_data["experiments"].append(experiment)
    print(f"Created experiment: {experiment.display_name} (ID: {experiment.experiment_id})")

    # Create a run with the hello-world pipeline
    run = create_run(
        experiment.experiment_id,
        pipeline1.pipeline_id,
        version1.pipeline_version_id,
        "test-hello-world-run",
        parameters=None
    )
    
    test_data["runs"].append(run)
    print(f"Created run: {run.display_name} (ID: {run.run_id})")

    # Create a recurring run with the add-numbers pipeline (using v1)
    recurring_run = create_recurring_run(
        experiment.experiment_id,
        pipeline2.pipeline_id,
        version2_v1.pipeline_version_id,
        "test-add-numbers-recurring-run",
        "0 0 * * *",  
        parameters=[{"name": "a", "value": 5}, {"name": "b", "value": 3}]
    )
   
    test_data["recurring_runs"].append(recurring_run)
    print(f"Created recurring run: {recurring_run.display_name} (ID: {recurring_run.recurring_run_id})")

    # Save test data for later use
    migration_test_data_file = Path("migration_test_data.pkl")
    with open(migration_test_data_file, "wb") as f:
        pickle.dump(test_data, f) 

    print(f"\nTest data saved to {migration_test_data_file}")
    print("Test environment setup complete!")  
    
if __name__ == "__main__":
    main()
