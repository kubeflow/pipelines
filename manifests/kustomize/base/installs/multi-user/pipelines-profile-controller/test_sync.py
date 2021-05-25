import os
from unittest import mock
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from sync import Controller
import json
import time 

import pytest
import requests

# Misc Constants
SERVER_ADDRESS = ("", 0)  # resolves to localhost:some_free_port

# Data sets passed to server
DATA_INCORRECT_CHILDREN = {
    "parent": {
        "metadata": {
            "labels": {
                "pipelines.kubeflow.org/enabled": "true"
            },
            "name": "myName"
        }
    },
    "children": {
        "Secret.v1": [],
        "ConfigMap.v1": [],
        "Deployment.apps/v1": [],
        "Service.v1": [],
        "DestinationRule.networking.istio.io/v1alpha3": [],
        "AuthorizationPolicy.security.istio.io/v1beta1": [],
    }
}

DATA_CORRECT_CHILDREN = {
    "parent": {
        "metadata": {
            "labels": {
                "pipelines.kubeflow.org/enabled": "true"
            },
            "name": "myName"
        }
    },
    "children": {
        "Secret.v1": [1],
        "ConfigMap.v1": [1],
        "Deployment.apps/v1": [1, 1],
        "Service.v1": [1, 1],
        "DestinationRule.networking.istio.io/v1alpha3": [1],
        "AuthorizationPolicy.security.istio.io/v1beta1": [1],
    }
}

DATA_MISSING_PIPELINE_ENABLED = {"parent": {}, "children": {}}

# Default values when environments are not explicit
DEFAULT_FRONTEND_IMAGE="gcr.io/ml-pipeline/frontend"
DEFAULT_VISUALIZATION_IMAGE="gcr.io/ml-pipeline/visualization-server"

# Variables used for environment variable sets
VISUALIZATION_SERVER_IMAGE="vis-image"
VISUALIZATION_SERVER_TAG="somenumber.1.2.3"
FRONTEND_IMAGE="frontend-image"
FRONTEND_TAG="somehash"

KFP_VERSION="x.y.z"

MINIO_ACCESS_KEY="abcdef"
MINIO_SECRET_KEY="uvwxyz"

# "Environments" used in tests
ENV_VARIABLES_BASE = {
    "MINIO_ACCESS_KEY": MINIO_ACCESS_KEY,
    "MINIO_SECRET_KEY": MINIO_SECRET_KEY,
}

ENV_KFP_VERSION_ONLY = dict(ENV_VARIABLES_BASE, 
    **{
        "KFP_VERSION": KFP_VERSION,
    }
)

ENV_IMAGES_NO_TAGS = dict(ENV_VARIABLES_BASE, 
    **{
        "KFP_VERSION": KFP_VERSION,
        "VISUALIZATION_SERVER_IMAGE": VISUALIZATION_SERVER_IMAGE,
        "FRONTEND_IMAGE": FRONTEND_IMAGE,
    }
)

ENV_IMAGES_WITH_TAGS = dict(ENV_VARIABLES_BASE, 
    **{
    "VISUALIZATION_SERVER_IMAGE": VISUALIZATION_SERVER_IMAGE,
    "FRONTEND_IMAGE": FRONTEND_IMAGE,
    "VISUALIZATION_SERVER_TAG": VISUALIZATION_SERVER_TAG,
    "FRONTEND_TAG": FRONTEND_TAG,
    }
)


def generate_image_name(imagename, tag):
    return f"{str(imagename)}:{str(tag)}"


@pytest.fixture(
    scope="function",
)
def sync_server(request):
    """
    Starts the sync HTTP server for a given set of environment variables on a separate thread

    Yields:
    * the server (useful to interrogate for the server address)
    * environment variables (useful to interrogate for correct responses)
    """
    environ = request.param
    with mock.patch.dict(os.environ, environ):
        # Create a server at an available port and serve it on a thread as a daemon
        # This will result in a collection of servers being active - not a great way
        # if this fixture is run many times during a test, but ok for now
        handler_partial = Controller.partial_init_from_environment()
        server = HTTPServer(SERVER_ADDRESS, handler_partial)
        server_thread = threading.Thread(target=server.serve_forever)
        # Put on daemon so it doesn't keep pytest from ending
        server_thread.daemon = True
        server_thread.start()
        yield server, environ


@pytest.mark.parametrize(
    "sync_server, data, expected_status, expected_visualization_server_image, expected_frontend_server_image",
    [
        (
            ENV_KFP_VERSION_ONLY, 
            DATA_INCORRECT_CHILDREN, 
            {"kubeflow-pipelines-ready": "False"},
            generate_image_name(DEFAULT_VISUALIZATION_IMAGE, KFP_VERSION),
            generate_image_name(DEFAULT_FRONTEND_IMAGE, KFP_VERSION),
        ),
        (
            ENV_IMAGES_NO_TAGS,
            DATA_INCORRECT_CHILDREN,
            {"kubeflow-pipelines-ready": "False"},
            generate_image_name(ENV_IMAGES_NO_TAGS["VISUALIZATION_SERVER_IMAGE"], KFP_VERSION),
            generate_image_name(ENV_IMAGES_NO_TAGS["FRONTEND_IMAGE"], KFP_VERSION),
        ),
        (
            ENV_IMAGES_WITH_TAGS,
            DATA_INCORRECT_CHILDREN,
            {"kubeflow-pipelines-ready": "False"},
            generate_image_name(ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_IMAGE"], ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_TAG"]),
            generate_image_name(ENV_IMAGES_WITH_TAGS["FRONTEND_IMAGE"], ENV_IMAGES_WITH_TAGS["FRONTEND_TAG"]),
        ),
        (
            ENV_IMAGES_WITH_TAGS,
            DATA_CORRECT_CHILDREN,
            {"kubeflow-pipelines-ready": "True"},
            generate_image_name(ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_IMAGE"], ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_TAG"]),
            generate_image_name(ENV_IMAGES_WITH_TAGS["FRONTEND_IMAGE"], ENV_IMAGES_WITH_TAGS["FRONTEND_TAG"]),
        ),
    ],
    indirect=["sync_server"]
)
def test_sync_server_with_pipeline_enabled(sync_server, data, expected_status, 
    expected_visualization_server_image, expected_frontend_server_image):
    """
    Nearly end-to-end test of how Controller serves .sync as a POST

    Tests case where metadata.labels.pipelines.kubeflow.org/enabled exists, and thus
    we should produce children

    Only does spot checks on children to see if key properties are correct
    """
    server, environ = sync_server

    # server.server_address = (url, port_as_integer)
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    print("url: ", url)
    print("data")
    print(json.dumps(data, indent=2))
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    # Test overall status of whether children are ok
    assert results['status'] == expected_status

    # Poke a few children to test things that can vary by environment variable
    assert results['children'][1]["spec"]["template"]["spec"]["containers"][0]["image"] == expected_visualization_server_image
    assert results['children'][5]["spec"]["template"]["spec"]["containers"][0]["image"] == expected_frontend_server_image


@pytest.mark.parametrize(
    "sync_server, data, expected_status, expected_children",
    [
        (ENV_IMAGES_WITH_TAGS, DATA_MISSING_PIPELINE_ENABLED, {}, []),
    ],
    indirect=["sync_server"]
)
def test_sync_server_without_pipeline_enabled(sync_server, data, expected_status, 
    expected_children):
    """
    Nearly end-to-end test of how Controller serves .sync as a POST

    Tests case where metadata.labels.pipelines.kubeflow.org/enabled does not 
    exist and thus server returns an empty reply
    """
    server, environ = sync_server

    # server.server_address = (url, port_as_integer)
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    # Test overall status of whether children are ok
    assert results['status'] == expected_status
    assert results['children'] == expected_children
