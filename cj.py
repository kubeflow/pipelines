from google.cloud import aiplatform

project = 'managed-pipeline-test'
location = 'us-central1'

client = aiplatform.gapic.JobServiceClient(
    client_options={"api_endpoint": f"{location}-aiplatform.googleapis.com"})

# Configure the custom job
custom_job = {
    "workerPoolSpecs": [{
        "machineSpec": {
            "machineType": "e2-standard-4"
        },
        "replicaCount": "1",
        "diskSpec": {
            "bootDiskType": "pd-ssd",
            "bootDiskSizeGb": 100
        },
        "containerSpec": {
            "imageUri":
                "python:3.7",
            "command": [
                "sh", "-c",
                "\nif ! [ -x \"$(command -v pip)\" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location 'kfp==2.6.0' '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<\"3.9\"' && \"$0\" \"$@\"\n",
                "sh", "-ec",
                "program_path=$(mktemp -d)\n\nprintf \"%s\" \"$0\" > \"$program_path/ephemeral_component.py\"\n_KFP_RUNTIME=true python3 -m kfp.dsl.executor_main                         --component_module_path                         \"$program_path/ephemeral_component.py\"                         \"$@\"\n",
                "\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef test_add_1(a: int, b: int) -> int:\n    if a > 2:\n        sys.exit(\"a is greater than 2, invalid\")\n    return a + b\n\n"
            ],
            "args": [
                "--executor_input",
                "{\"inputs\":{\"parameterValues\":{\"a\":3,\"b\":2},\"parameters\":{\"a\":{\"intValue\":\"3\"},\"b\":{\"intValue\":\"2\"}}},\"outputs\":{\"outputFile\":\"/gcs/rickyxie-test/186556260430/hello-world-20240314011708/test-add-1_-5950113757618241536/executor_output.json\",\"parameters\":{\"Output\":{\"outputFile\":\"/gcs/rickyxie-test/186556260430/hello-world-20240314011708/test-add-1_-5950113757618241536/Output\"}}}}",
                "--function_to_execute", "test_add_1"
            ],
            "env": [{
                "name":
                    "VERTEX_AI_PIPELINES_RUN_LABELS",
                "value":
                    "{\"vertex-ai-pipelines-run-billing-id\":\"8717761889699889152\"}"
            }]
        }
    }],
    "scheduling": {
        "disableRetries": True
    },
    "serviceAccount": "186556260430-compute@developer.gserviceaccount.com"
}

parent = f"projects/{project}/locations/{location}"
response = client.create_custom_job(parent=parent, custom_job=custom_job)

print("Custom job name:", response.name)
