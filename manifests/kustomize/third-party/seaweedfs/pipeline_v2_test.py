#!/usr/bin/env python3

import kfp
import sys
import time
from kfp import dsl
from kfp_server_api.exceptions import ApiException


@dsl.component
def hello_world_op() -> str:
    print("Hello World from Kubeflow Pipelines V2!")
    return "Hello World"


@dsl.pipeline(
    name="hello-world-v2",
    description="A very simple hello world pipeline"
)
def hello_world_pipeline():
    hello_world_op()


def run_pipeline(token, namespace):
    client = kfp.Client(host="http://localhost:8080/pipeline", existing_token=token)
    
    try:
        pipelines = client.list_pipelines()
        print(f"Successfully connected to KFP server, found {len(pipelines.pipelines)} pipelines")
        
        experiment = client.create_experiment("v2-pipeline-test", namespace=namespace)
        print(f"Created experiment: v2-pipeline-test in namespace {namespace}")
        
        run = client.create_run_from_pipeline_func(
            pipeline_func=hello_world_pipeline,
            experiment_name="v2-pipeline-test",
            run_name="v2-test-run",
            arguments={},
            namespace=namespace
        )
        
        run_id = run.run_id
        
        for _ in range(30): 
            status = client.get_run(run_id=run_id).state
            
            if status == "SUCCEEDED":
                return
            elif status not in ["PENDING", "RUNNING"]:
                print(f"Pipeline failed with status: {status}")
                
                pods = client._get_k8s_client().list_namespaced_pod(
                    namespace=namespace,
                    label_selector=f"pipeline/runid={run_id}"
                )
                
                print(f"Found {len(pods.items)} pods for this run")
                for pod in pods.items:
                    print(f"Pod {pod.metadata.name}: {pod.status.phase}")
                
                sys.exit(1)
                
            time.sleep(10)
        
        sys.exit(1)
        
    except Exception as exception:
        print(f"Error in pipeline execution: {exception}")
        sys.exit(1)


def test_unauthorized_access(token, namespace):
    client = kfp.Client(host="http://localhost:8080/pipeline", existing_token=token)

    try:
        pipeline = client.list_runs(namespace=namespace)
        sys.exit(1)
    except ApiException as exception:
        if exception.status != 403:
            sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(1)
    
    action = sys.argv[1]
    token = sys.argv[2]
    namespace = sys.argv[3]

    if action == "run_pipeline":
        run_pipeline(token, namespace)
    elif action == "test_unauthorized_access":
        test_unauthorized_access(token, namespace)
    else:
        sys.exit(1)