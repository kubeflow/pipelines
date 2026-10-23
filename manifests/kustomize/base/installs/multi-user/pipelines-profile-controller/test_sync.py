import json
import os
import threading
from unittest import mock

import pytest
import requests
import sync
from sync import get_settings_from_env
from sync import server_factory

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
DEFAULT_FRONTEND_IMAGE = "ghcr.io/kubeflow/kfp-frontend"
DEFAULT_VISUALIZATION_IMAGE = "ghcr.io/kubeflow/kfp-visualization-server"

# Variables used for environment variable sets
VISUALIZATION_SERVER_IMAGE = "vis-image"
VISUALIZATION_SERVER_TAG = "somenumber.1.2.3"
FRONTEND_IMAGE = "frontend-image"
FRONTEND_TAG = "somehash"

KFP_VERSION = "x.y.z"

MINIO_ACCESS_KEY = "abcdef"
MINIO_SECRET_KEY = "uvwxyz"

# "Environments" used in tests
ENV_VARIABLES_BASE = {
    "MINIO_ACCESS_KEY": MINIO_ACCESS_KEY,
    "MINIO_SECRET_KEY": MINIO_SECRET_KEY,
    "CONTROLLER_PORT":
        "0",  # HTTPServer randomly assigns the port to a free port
}

ENV_KFP_VERSION_ONLY = dict(ENV_VARIABLES_BASE, **{
    "KFP_VERSION": KFP_VERSION,
})

ENV_IMAGES_NO_TAGS = dict(
    ENV_VARIABLES_BASE, **{
        "KFP_VERSION": KFP_VERSION,
        "VISUALIZATION_SERVER_IMAGE": VISUALIZATION_SERVER_IMAGE,
        "FRONTEND_IMAGE": FRONTEND_IMAGE,
    })

ENV_IMAGES_WITH_TAGS = dict(
    ENV_VARIABLES_BASE, **{
        "VISUALIZATION_SERVER_IMAGE": VISUALIZATION_SERVER_IMAGE,
        "FRONTEND_IMAGE": FRONTEND_IMAGE,
        "VISUALIZATION_SERVER_TAG": VISUALIZATION_SERVER_TAG,
        "FRONTEND_TAG": FRONTEND_TAG,
    })

ENV_IMAGES_WITH_TAGS_AND_ISTIO = dict(ENV_IMAGES_WITH_TAGS, **{
    "DISABLE_ISTIO_SIDECAR": "false",
})

ENV_ARTIFACT_PROXY_WITH_ALLOWED_ENDPOINTS = dict(
    ENV_KFP_VERSION_ONLY, **{
        "ALLOWED_ARTIFACT_ENDPOINTS": "https://objects.example.com:9443",
        "ARTIFACTS_PROXY_ENABLED": "true",
        "HTTP_BASE_URL": "https://artifacts.example.com:9443/pipelines/",
    })


def generate_image_name(imagename, tag):
    return f"{str(imagename)}:{str(tag)}"


@pytest.fixture(scope="function",)
def sync_server(request):
    """Starts the sync HTTP server for a given set of environment variables on
    a separate thread.

    Yields:
    * the server (useful to interrogate for the server address)
    * environment variables (useful to interrogate for correct responses)
    """
    environ = request.param
    with mock.patch.dict(os.environ, environ):
        # Create a server at an available port and serve it on a thread as a daemon
        # This will result in a collection of servers being active - not a great way
        # if this fixture is run many times during a test, but ok for now
        settings = get_settings_from_env()
        server = server_factory(**settings)
        server_thread = threading.Thread(target=server.serve_forever)
        # Put on daemon so it doesn't keep pytest from ending
        server_thread.daemon = True
        server_thread.start()
        yield server, environ


@pytest.fixture(scope="function",)
def sync_server_from_arguments(request):
    """Starts the sync HTTP server for a given set of parameters passed as
    arguments, with server on a separate thread.

    Yields:
    * the server (useful to interrogate for the server address)
    * environment variables (useful to interrogate for correct responses)
    """
    environ = {k.lower(): v for k, v in request.param.items()}
    settings = environ
    server = server_factory(**settings)
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
            {
                "kubeflow-pipelines-ready": "False"
            },
            generate_image_name(DEFAULT_VISUALIZATION_IMAGE, KFP_VERSION),
            generate_image_name(DEFAULT_FRONTEND_IMAGE, KFP_VERSION),
        ),
        (
            ENV_IMAGES_NO_TAGS,
            DATA_INCORRECT_CHILDREN,
            {
                "kubeflow-pipelines-ready": "False"
            },
            generate_image_name(
                ENV_IMAGES_NO_TAGS["VISUALIZATION_SERVER_IMAGE"], KFP_VERSION),
            generate_image_name(ENV_IMAGES_NO_TAGS["FRONTEND_IMAGE"],
                                KFP_VERSION),
        ),
        (
            ENV_IMAGES_WITH_TAGS,
            DATA_INCORRECT_CHILDREN,
            {
                "kubeflow-pipelines-ready": "False"
            },
            generate_image_name(
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_IMAGE"],
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_TAG"]),
            generate_image_name(ENV_IMAGES_WITH_TAGS["FRONTEND_IMAGE"],
                                ENV_IMAGES_WITH_TAGS["FRONTEND_TAG"]),
        ),
        (
            ENV_IMAGES_WITH_TAGS,
            DATA_CORRECT_CHILDREN,
            {
                "kubeflow-pipelines-ready": "True"
            },
            generate_image_name(
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_IMAGE"],
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_TAG"]),
            generate_image_name(ENV_IMAGES_WITH_TAGS["FRONTEND_IMAGE"],
                                ENV_IMAGES_WITH_TAGS["FRONTEND_TAG"]),
        ),
    ],
    indirect=["sync_server"])
def test_sync_server_with_pipeline_enabled(sync_server, data, expected_status,
                                           expected_visualization_server_image,
                                           expected_frontend_server_image):
    """Nearly end-to-end test of how Controller serves .sync as a POST.

    Tests case where metadata.labels.pipelines.kubeflow.org/enabled
    exists, and thus we should produce children

    Only does spot checks on children to see if key properties are
    correct
    """
    server, environ = sync_server

    # server.server_address = (url, port_as_integer)
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    print("url: ", url)
    print("data")
    print(json.dumps(data))
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    # Test overall status of whether children are ok
    assert results['status'] == expected_status

    # Poke a few children to test things that can vary by environment variable
    assert results['children'][1]["spec"]["template"]["spec"]["containers"][0][
        "image"] == expected_visualization_server_image
    assert results['children'][5]["spec"]["template"]["spec"]["containers"][0][
        "image"] == expected_frontend_server_image


@pytest.mark.parametrize(
    "sync_server_from_arguments, data, expected_status, expected_visualization_server_image, "
    "expected_frontend_server_image", [
        (
            ENV_IMAGES_WITH_TAGS_AND_ISTIO,
            DATA_CORRECT_CHILDREN,
            {
                "kubeflow-pipelines-ready": "True"
            },
            generate_image_name(
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_IMAGE"],
                ENV_IMAGES_WITH_TAGS["VISUALIZATION_SERVER_TAG"]),
            generate_image_name(ENV_IMAGES_WITH_TAGS["FRONTEND_IMAGE"],
                                ENV_IMAGES_WITH_TAGS["FRONTEND_TAG"]),
        ),
    ],
    indirect=["sync_server_from_arguments"])
def test_sync_server_with_direct_passing_of_settings(
        sync_server_from_arguments, data, expected_status,
        expected_visualization_server_image, expected_frontend_server_image):
    """Nearly end-to-end test of how Controller serves .sync as a POST, taking
    variables as arguments.

    Only does spot checks on children to see if key properties are
    correct
    """
    server, environ = sync_server_from_arguments

    # server.server_address = (url, port_as_integer)
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    print("url: ", url)
    print("data")
    print(json.dumps(data))
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    # Test overall status of whether children are ok
    assert results['status'] == expected_status

    # Poke a few children to test things that can vary by environment variable
    assert results['children'][1]["spec"]["template"]["spec"]["containers"][0][
        "image"] == expected_visualization_server_image
    assert results['children'][5]["spec"]["template"]["spec"]["containers"][0][
        "image"] == expected_frontend_server_image


@pytest.mark.parametrize(
    "sync_server, data, expected_status, expected_children", [
        (ENV_IMAGES_WITH_TAGS, DATA_MISSING_PIPELINE_ENABLED, {}, []),
    ],
    indirect=["sync_server"])
def test_sync_server_without_pipeline_enabled(sync_server, data,
                                              expected_status,
                                              expected_children):
    """Nearly end-to-end test of how Controller serves .sync as a POST.

    Tests case where metadata.labels.pipelines.kubeflow.org/enabled does
    not exist and thus server returns an empty reply
    """
    server, environ = sync_server

    # server.server_address = (url, port_as_integer)
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    # Test overall status of whether children are ok
    assert results['status'] == expected_status
    assert results['children'] == expected_children


@pytest.mark.parametrize(
    "sync_server",
    [ENV_ARTIFACT_PROXY_WITH_ALLOWED_ENDPOINTS],
    indirect=True,
)
def test_artifact_proxy_receives_allowed_endpoints(sync_server):
    server, _ = sync_server
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    existing_secret = {
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'name': 'mlpipeline-minio-artifact',
            'namespace': 'myName',
        },
    }
    response = requests.post(
        url,
        json={
            'object': DATA_CORRECT_CHILDREN['parent'],
            'attachments': {
                'Secret.v1': {
                    'myName/mlpipeline-minio-artifact': existing_secret
                },
                'ConfigMap.v1': {},
                'Deployment.apps/v1': {},
                'Service.v1': {},
            },
        })
    results = json.loads(response.text)
    artifact_deployment = next(
        child for child in results['attachments']
        if child.get('kind') == 'Deployment' and
        child.get('metadata', {}).get('name') == 'ml-pipeline-ui-artifact')
    container_env = artifact_deployment['spec']['template']['spec'][
        'containers'][0]['env']

    assert {
        'name': 'HTTP_BASE_URL',
        'value': 'https://artifacts.example.com:9443/pipelines/',
    } in container_env

    assert {
        'name': 'ALLOWED_ARTIFACT_ENDPOINTS',
        'value': 'https://objects.example.com:9443',
    } in container_env


def test_create_iam_client_uses_endpoint(monkeypatch):
    called = {}

    class DummySession:

        def create_client(self,
                          service_name,
                          region_name=None,
                          endpoint_url=None):
            called["service_name"] = service_name
            called["endpoint_url"] = endpoint_url
            return object()

    monkeypatch.setenv("AWS_ENDPOINT_URL", "http://seaweedfs.kubeflow:8111")
    monkeypatch.setattr(sync, "session", DummySession())

    sync.create_iam_client()

    assert called["service_name"] == "iam"
    assert called["endpoint_url"] == "http://seaweedfs.kubeflow:8111"


# Health endpoint test helpers (master's shared fixtures are not on this branch).
PARENT = {
    "metadata": {
        "labels": {
            "pipelines.kubeflow.org/enabled": "true"
        },
        "name": "myName",
    }
}


def observed_attachments():
    return {}


def post_sync(server, parent, attachments):
    host, port = server.server_address
    response = requests.post(
        f"http://{host}:{port}",
        json={
            "object": parent,
            "attachments": attachments
        },
        timeout=5,
    )
    response.raise_for_status()
    return response.json()


@pytest.fixture
def start_sync_server():
    servers = []

    def start(**settings):
        server = server_factory(url="127.0.0.1", **settings)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        servers.append((server, thread))
        return server

    yield start
    for server, thread in servers:
        server.shutdown()
        thread.join()
        server.server_close()


@pytest.fixture
def sync_server(request, start_sync_server):
    with mock.patch.dict(os.environ, request.param, clear=True):
        return start_sync_server(**get_settings_from_env())


def post_sync(server, parent, attachments):
    host, port = server.server_address
    response = requests.post(
        f'http://{host}:{port}',
        json={'object': parent, 'attachments': attachments},
        timeout=5,
    )
    response.raise_for_status()
    return response.json()


def assert_profile_resources(result, expected_image, ca_cert=None):
    resources = result['attachments']
    identities = {
        (item['kind'], item['metadata']['name']) for item in resources
    }
    expected = {
        ('Secret', 'mlpipeline-minio-artifact'),
        (
            'Secret',
            'ml-pipeline-driver-agent-executor-plugin.service-account-token',
        ),
        ('Secret', 'default-editor.service-account-token'),
        ('ConfigMap', 'kfp-launcher'),
        ('ConfigMap', 'artifact-repositories'),
        ('ServiceAccount', 'ml-pipeline-driver-agent-executor-plugin'),
        ('RoleBinding', 'ml-pipeline-driver-agent-executor-plugin-binding'),
    }
    if expected_image:
        expected.update(
            {
                ('Deployment', 'ml-pipeline-ui-artifact'),
                ('Service', 'ml-pipeline-ui-artifact'),
            }
        )
        deployment = next(
            item for item in resources if item['kind'] == 'Deployment'
        )
        assert (
            deployment['spec']['template']['spec']['containers'][0]['image']
            == expected_image
        )
    if ca_cert is not None:
        expected.add(('Secret', sync.DRIVER_AGENT_CA_SECRET_NAME))
        secret = next(
            item
            for item in resources
            if item['metadata']['name'] == sync.DRIVER_AGENT_CA_SECRET_NAME
        )
        assert secret['metadata']['namespace'] == NAMESPACE
        assert secret['data'] == {
            'ca.crt': base64.b64encode(ca_cert).decode('utf-8')
        }
    # No Python visualization Deployment, Service, or Istio resources are desired.
    assert identities == expected
    assert len(resources) == len(expected)


@pytest.mark.parametrize(
    'sync_server, expected_image',
    [
        (ENV_BASE, None),
        (
            {**ENV_BASE, 'ARTIFACTS_PROXY_ENABLED': 'true'},
            f'ghcr.io/kubeflow/kfp-frontend:{KFP_VERSION}',
        ),
        (
            {
                **ENV_BASE,
                'ARTIFACTS_PROXY_ENABLED': 'true',
                'FRONTEND_IMAGE': 'custom',
            },
            f'custom:{KFP_VERSION}',
        ),
        (ENV_IMAGES, 'frontend-image:somehash'),
        (
            {**ENV_IMAGES, 'DISABLE_ISTIO_SIDECAR': 'false'},
            'frontend-image:somehash',
        ),
    ],
    indirect=['sync_server'],
)
@pytest.mark.parametrize('ready', [False, True])
def test_sync_desires_profile_resources(sync_server, expected_image, ready):
    result = post_sync(
        sync_server, PARENT, observed_attachments(bool(expected_image), ready)
    )
    assert result['status'] == {'kubeflow-pipelines-ready': str(ready)}
    assert_profile_resources(result, expected_image)


def test_sync_server_with_direct_settings(start_sync_server):
    server = start_sync_server(
        frontend_image='direct-image',
        frontend_tag='direct-tag',
        disable_istio_sidecar=False,
        artifacts_proxy_enabled='true',
        artifact_retention_days=-1,
        controller_port=0,
    )
    result = post_sync(server, PARENT, observed_attachments(True, True))
    assert result['status'] == {'kubeflow-pipelines-ready': 'True'}
    assert_profile_resources(result, 'direct-image:direct-tag')


def start_ca_server(start_sync_server, ca_cert_path):
    return start_sync_server(
        frontend_image='image',
        frontend_tag='tag',
        disable_istio_sidecar=False,
        artifacts_proxy_enabled='false',
        artifact_retention_days=-1,
        controller_port=0,
        driver_agent_ca_cert_path=str(ca_cert_path),
    )


def test_sync_copies_driver_agent_ca_into_profile(start_sync_server, tmp_path):
    ca_cert_path = tmp_path / 'ca.crt'
    ca_cert_path.write_bytes(b'-----BEGIN CERTIFICATE-----\nfirst\n')
    server = start_ca_server(start_sync_server, ca_cert_path)

    result = post_sync(server, PARENT, observed_attachments(ready=True))
    # The copy is not observed yet, so the profile is not ready.
    assert result['status'] == {'kubeflow-pipelines-ready': 'False'}
    assert_profile_resources(result, None, ca_cert_path.read_bytes())

    result = post_sync(
        server, PARENT, observed_attachments(ready=True, ca=True)
    )
    assert result['status'] == {'kubeflow-pipelines-ready': 'True'}
    assert_profile_resources(result, None, ca_cert_path.read_bytes())

    # Rotation: the next sync carries the current CA without a restart.
    ca_cert_path.write_bytes(b'-----BEGIN CERTIFICATE-----\nrotated\n')
    result = post_sync(
        server, PARENT, observed_attachments(ready=True, ca=True)
    )
    assert_profile_resources(result, None, ca_cert_path.read_bytes())


def test_ca_rotation_reconciles_without_changing_other_secrets(
    start_sync_server, tmp_path
):
    controller = yaml.safe_load(
        Path(sync.__file__).with_name('decorator-controller.yaml').read_text()
    )
    secret_rules = [
        rule
        for rule in controller['spec']['attachments']
        if rule['resource'] == 'secrets'
    ]
    assert len(secret_rules) == 1
    assert secret_rules[0]['updateStrategy']['method'] == 'InPlace'

    ca_cert_path = tmp_path / 'ca.crt'
    ca_cert_path.write_bytes(b'original CA')
    server = start_ca_server(start_sync_server, ca_cert_path)
    result = post_sync(server, PARENT, observed_attachments(ready=True))
    observed = observed_attachments(ready=True)
    for resource in result['attachments']:
        if resource['kind'] == 'Secret':
            name = resource['metadata']['name']
            observed['Secret.v1'][f'{NAMESPACE}/{name}'] = resource
    # Token controllers and operators may populate or customize these Secrets.
    for secret in observed['Secret.v1'].values():
        if secret['metadata']['name'] != sync.DRIVER_AGENT_CA_SECRET_NAME:
            secret['data'] = {
                'token': 'existing-token',
                'ca.crt': 'existing-ca',
            }
            secret['metadata']['annotations'] = {'custom': 'preserve-me'}
    before = copy.deepcopy(observed)

    ca_cert_path.write_bytes(b'rotated CA')
    result = post_sync(server, PARENT, observed)
    assert_profile_resources(result, None, b'rotated CA')
    for secret in result['attachments']:
        if secret['kind'] != 'Secret':
            continue
        previous = before['Secret.v1'][
            f'{NAMESPACE}/{secret["metadata"]["name"]}'
        ]
        if secret['metadata']['name'] == sync.DRIVER_AGENT_CA_SECRET_NAME:
            assert secret['data'] != previous['data']
        else:
            assert secret == previous
    assert observed == before


# Read-only metadata that Metacontroller v4.16.2 reverts after merging
# (objectMetaSystemFields in pkg/controller/common/manage_children.go).
METACONTROLLER_SYSTEM_FIELDS = (
    'selfLink',
    'uid',
    'resourceVersion',
    'generation',
    'creationTimestamp',
    'deletionTimestamp',
    'deletionGracePeriodSeconds',
    'managedFields',
)


def metacontroller_merge(destination, last_applied, desired):
    """Port of dynamicapply.merge from Metacontroller v4.16.2: apply the diff
    from last_applied to desired onto destination.

    Secrets hold no lists, so lists take the desired value.
    """
    if isinstance(destination, dict):
        last_applied = last_applied or {}
        desired = desired or {}
        for key in last_applied:
            if key not in desired:
                destination.pop(key, None)
        for key, value in desired.items():
            destination[key] = metacontroller_merge(
                destination.get(key), last_applied.get(key), value
            )
        return destination
    return desired


def metacontroller_apply_update(observed, desired):
    """Port of common.ApplyUpdate from Metacontroller v4.16.2, which decides
    whether an InPlace attachment differs from the observed object."""
    desired = copy.deepcopy(desired)
    desired.get('metadata', {}).get('annotations', {}).pop(
        sync.LAST_APPLIED_ANNOTATION, None
    )
    last_applied = json.loads(
        observed['metadata']
        .get('annotations', {})
        .get(sync.LAST_APPLIED_ANNOTATION)
        or 'null'
    )
    merged = metacontroller_merge(
        copy.deepcopy(observed), last_applied, desired
    )
    for field in METACONTROLLER_SYSTEM_FIELDS:
        if field in observed['metadata']:
            merged['metadata'][field] = observed['metadata'][field]
        else:
            merged['metadata'].pop(field, None)
    merged['metadata'].setdefault('annotations', {})[
        sync.LAST_APPLIED_ANNOTATION
    ] = json.dumps(desired, sort_keys=True)
    return merged


def metacontroller_create(desired, resource_versions):
    """A Secret as the API server stores it after Metacontroller creates it:
    the last-applied annotation, the controller ownerReference, and the
    system metadata that observed attachments carry."""
    stored = copy.deepcopy(desired)
    stored['metadata'].setdefault('annotations', {})[
        sync.LAST_APPLIED_ANNOTATION
    ] = json.dumps(desired, sort_keys=True)
    stored['metadata']['ownerReferences'] = [
        {
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'name': NAMESPACE,
            'uid': 'parent-uid',
            'controller': True,
            'blockOwnerDeletion': True,
        }
    ]
    stored['metadata'].update(
        {
            'uid': f'uid-{desired["metadata"]["name"]}',
            'creationTimestamp': '2026-10-14T00:00:00Z',
            'managedFields': [{'manager': 'metacontroller'}],
            'resourceVersion': str(next(resource_versions)),
        }
    )
    stored.setdefault('type', 'Opaque')
    return stored


def test_preserved_secrets_converge_under_in_place_updates(
    start_sync_server, tmp_path
):
    ca_cert_path = tmp_path / 'ca.crt'
    ca_cert_path.write_bytes(b'original CA')
    server = start_ca_server(start_sync_server, ca_cert_path)
    resource_versions = itertools.count(1)
    observed = observed_attachments(ready=True)
    observed['Secret.v1'] = {
        f'{NAMESPACE}/mlpipeline-minio-artifact': metacontroller_create(
            {
                **copy.deepcopy(EXISTING_SECRET),
                'data': {'accesskey': 'YWNjZXNz', 'secretkey': 'c2VjcmV0'},
            },
            resource_versions,
        ),
    }

    def reconcile():
        """One Metacontroller sync: create missing Secrets, apply the others
        in place, and count how many the API server had to write."""
        result = post_sync(server, PARENT, observed)
        writes = 0
        for desired in result['attachments']:
            if desired['kind'] != 'Secret':
                continue
            key = f'{NAMESPACE}/{desired["metadata"]["name"]}'
            assert not set(desired['metadata']) & (
                set(METACONTROLLER_SYSTEM_FIELDS) | {'ownerReferences'}
            ), key
            assert sync.LAST_APPLIED_ANNOTATION not in desired['metadata'].get(
                'annotations', {}
            ), key
            current = observed['Secret.v1'].get(key)
            if current is None:
                observed['Secret.v1'][key] = metacontroller_create(
                    desired, resource_versions
                )
                writes += 1
                continue
            updated = metacontroller_apply_update(current, desired)
            if updated != current:
                updated['metadata']['resourceVersion'] = str(
                    next(resource_versions)
                )
                observed['Secret.v1'][key] = updated
                writes += 1
        return writes

    # The first sync creates the CA and token Secrets and may adopt
    # server-defaulted fields of the pre-existing credentials once.
    assert reconcile() >= 3
    # The token controller then populates the service-account token Secrets.
    token_data = {
        'token': 'ZXhpc3Rpbmctand0',
        'ca.crt': 'Y2x1c3Rlci1jYQ==',
        'namespace': 'bXlOYW1l',
    }
    for secret in observed['Secret.v1'].values():
        if secret.get('type') == 'kubernetes.io/service-account-token':
            secret['data'] = dict(token_data)
            secret['metadata']['annotations'][
                'kubernetes.io/service-account.uid'
            ] = 'sa-uid'
            secret['metadata']['resourceVersion'] = str(next(resource_versions))
    assert reconcile() == 2
    for _ in range(8):
        assert reconcile() == 0, observed['Secret.v1']

    # Rotation rewrites only the CA copy, and the loop settles again.
    ca_cert_path.write_bytes(b'rotated CA')
    assert reconcile() == 1
    for _ in range(8):
        assert reconcile() == 0, observed['Secret.v1']

    secrets = observed['Secret.v1']
    assert secrets[f'{NAMESPACE}/{sync.DRIVER_AGENT_CA_SECRET_NAME}'][
        'data'
    ] == {'ca.crt': base64.b64encode(b'rotated CA').decode('utf-8')}
    assert secrets[f'{NAMESPACE}/mlpipeline-minio-artifact']['data'] == {
        'accesskey': 'YWNjZXNz',
        'secretkey': 'c2VjcmV0',
    }
    for name in (
        'default-editor.service-account-token',
        'ml-pipeline-driver-agent-executor-plugin.service-account-token',
    ):
        secret = secrets[f'{NAMESPACE}/{name}']
        assert secret['data'] == token_data
        assert (
            secret['metadata']['annotations'][
                'kubernetes.io/service-account.uid'
            ]
            == 'sa-uid'
        )
        assert secret['metadata']['ownerReferences'][0]['uid'] == 'parent-uid'


@pytest.mark.parametrize('state', ['missing', 'empty'])
def test_sync_without_driver_agent_ca_is_unchanged(
    start_sync_server, tmp_path, state
):
    ca_cert_path = tmp_path / 'ca.crt'
    if state == 'empty':
        ca_cert_path.write_bytes(b'')
    server = start_ca_server(start_sync_server, ca_cert_path)

    result = post_sync(server, PARENT, observed_attachments(ready=True))
    assert result['status'] == {'kubeflow-pipelines-ready': 'True'}
    assert_profile_resources(result, None)


def test_driver_agent_ca_cert_path_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env()['driver_agent_ca_cert_path'] == ''
    with mock.patch.dict(
        os.environ,
        {
            **ENV_BASE,
            'DRIVER_AGENT_CA_CERT_PATH': '/etc/kfp/driver-agent-ca/ca.crt',
        },
        clear=True,
    ):
        assert (
            get_settings_from_env()['driver_agent_ca_cert_path']
            == '/etc/kfp/driver-agent-ca/ca.crt'
        )
        assert (
            get_settings_from_env(driver_agent_ca_cert_path='')[
                'driver_agent_ca_cert_path'
            ]
            == ''
        )


def test_artifact_identity_is_sourced_from_installation_config() -> None:
    """Forward identity without introducing a deployment-level preview
    policy."""
    directory = Path(__file__).parent
    deployment = yaml.safe_load((directory / 'deployment.yaml').read_text())
    launcher = yaml.safe_load(
        (
            directory.parents[2] / 'pipeline' / 'kfp-launcher-configmap.yaml'
        ).read_text()
    )
    assert 'artifactPreviewsEnabled' not in launcher['data']
    environment = deployment['spec']['template']['spec']['containers'][0]['env']
    assert not any(
        item['name'] == 'ARTIFACT_PREVIEWS_ENABLED' for item in environment
    )
    identity = next(
        item for item in environment if item['name'] == 'USERID_HEADER'
    )
    assert identity['valueFrom']['configMapKeyRef'] == {
        'name': 'pipeline-install-config',
        'key': 'USERID_HEADER',
        'optional': True,
    }


def test_user_id_header_default_and_override() -> None:
    """Use the installation identity contract for generated artifact
    servers."""
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env()['user_id_header'] is None
    with mock.patch.dict(
        os.environ, {**ENV_BASE, 'USERID_HEADER': 'x-company-user'}, clear=True
    ):
        assert get_settings_from_env()['user_id_header'] == 'x-company-user'
        assert (
            get_settings_from_env(user_id_header='x-override')['user_id_header']
            == 'x-override'
        )


@pytest.mark.parametrize('sync_server', [ENV_BASE], indirect=True)
def test_sync_server_without_pipeline_enabled(sync_server):
    assert post_sync(sync_server, {}, {}) == {'status': {}, 'attachments': []}


def test_http_base_url_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env()['http_base_url'] == ''

    with mock.patch.dict(
        os.environ,
        {
            'KFP_VERSION': KFP_VERSION,
            'HTTP_BASE_URL': 'https://artifacts.example.com/pipelines/',
        },
        clear=True,
    ):
        assert (
            get_settings_from_env()['http_base_url']
            == 'https://artifacts.example.com/pipelines/'
        )
        assert get_settings_from_env(http_base_url='')['http_base_url'] == ''
        assert (
            get_settings_from_env(
                http_base_url='https://other.example.com/prefix'
            )['http_base_url']
            == 'https://other.example.com/prefix'
        )


def test_allowed_gcs_universe_domains_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert (
            get_settings_from_env()['allowed_gcs_universe_domains']
            == 'googleapis.com'
        )
    with mock.patch.dict(
        os.environ,
        {
            **ENV_BASE,
            'ALLOWED_GCS_UNIVERSE_DOMAINS': 'googleapis.com,gdc.example',
        },
        clear=True,
    ):
        assert (
            get_settings_from_env()['allowed_gcs_universe_domains']
            == 'googleapis.com,gdc.example'
        )


@pytest.mark.parametrize(
    'sync_server',
    [
        {
            **ENV_IMAGES,
            'HTTP_BASE_URL': 'https://artifacts.example.com:9443/pipelines/',
            'ALLOWED_ARTIFACT_ENDPOINTS': 'https://objects.example.com:9443',
            'ALLOWED_GCS_UNIVERSE_DOMAINS': 'googleapis.com,gdc.example',
            'USERID_HEADER': 'x-company-user',
        }
    ],
    indirect=True,
)
def test_artifact_proxy_receives_allowed_endpoints(sync_server):
    result = post_sync(sync_server, PARENT, observed_attachments())
    deployment = next(
        item for item in result['attachments'] if item['kind'] == 'Deployment'
    )
    environment = deployment['spec']['template']['spec']['containers'][0]['env']
    assert {
        'name': 'HTTP_BASE_URL',
        'value': 'https://artifacts.example.com:9443/pipelines/',
    } in environment
    assert {
        'name': 'ALLOWED_ARTIFACT_ENDPOINTS',
        'value': 'https://objects.example.com:9443',
    } in environment
    assert {
        'name': 'ALLOWED_GCS_UNIVERSE_DOMAINS',
        'value': 'googleapis.com,gdc.example',
    } in environment
    assert {
        'name': 'KUBEFLOW_USERID_HEADER',
        'value': 'x-company-user',
    } in environment
    assert {
        'name': 'ARTIFACTS_RATE_LIMIT_DISABLED',
        'value': 'true',
    } in environment
    assert not any('METADATA' in variable['name'] for variable in environment)
    repositories = next(
        item
        for item in result['attachments']
        if item['metadata']['name'] == 'artifact-repositories'
    )
    archive = json.loads(repositories['data']['default-namespaced'])
    assert archive['archiveLogs'] is True
    assert archive['s3']['keyFormat'].startswith(
        f'private-artifacts/{NAMESPACE}/'
    )


def test_create_iam_client_uses_endpoint(monkeypatch):
    called = {}

    class DummySession:
        def create_client(
            self, service_name, region_name=None, endpoint_url=None
        ):
            called['service_name'] = service_name
            called['endpoint_url'] = endpoint_url
            return object()

    monkeypatch.setenv('AWS_ENDPOINT_URL', 'http://seaweedfs.kubeflow:8111')
    monkeypatch.setattr(sync, 'session', DummySession())
    sync.create_iam_client()
    assert called['service_name'] == 'iam'
    assert called['endpoint_url'] == 'http://seaweedfs.kubeflow:8111'


def start_minimal_server(start_sync_server):
    return start_sync_server(
        frontend_image="image",
        frontend_tag="tag",
        disable_istio_sidecar=False,
        artifacts_proxy_enabled="false",
        artifact_retention_days=-1,
        controller_port=0,
    )


def get_path(server, path, timeout=5):
    host, port = server.server_address
    return requests.get(f"http://{host}:{port}{path}", timeout=timeout)


def test_healthz_returns_ok(start_sync_server):
    response = get_path(start_minimal_server(start_sync_server), "/healthz")
    assert response.status_code == 200
    assert response.text == "ok"


def test_get_unknown_path_returns_not_found(start_sync_server):
    response = get_path(start_minimal_server(start_sync_server), "/unknown")
    assert response.status_code == 404


def test_healthz_is_not_logged(start_sync_server, capsys):
    server = start_minimal_server(start_sync_server)
    assert get_path(server, "/healthz").status_code == 200
    assert get_path(server, "/unknown").status_code == 404
    logs = capsys.readouterr().err
    assert "/healthz" not in logs
    assert "/unknown" in logs


def test_post_to_healthz_path_is_logged(start_sync_server, capsys):
    server = start_minimal_server(start_sync_server)
    host, port = server.server_address
    with mock.patch.object(
            server.RequestHandlerClass, "sync",
            lambda handler, parent, attachments: {
                "status": {},
                "attachments": []
            }):
        requests.post(
            f"http://{host}:{port}/healthz",
            json={
                "object": PARENT,
                "attachments": {}
            },
            timeout=5).raise_for_status()
    assert '"POST /healthz' in capsys.readouterr().err


def test_healthz_responds_during_slow_sync(start_sync_server):
    server = start_minimal_server(start_sync_server)
    sync_started = threading.Event()
    release_sync = threading.Event()

    def slow_sync(handler, parent, attachments):
        sync_started.set()
        release_sync.wait(timeout=10)
        return {"status": {}, "attachments": []}

    with mock.patch.object(server.RequestHandlerClass, "sync", slow_sync):
        post = threading.Thread(
            target=post_sync, args=(server, PARENT, observed_attachments()))
        post.start()
        try:
            assert sync_started.wait(timeout=5)
            # A single-threaded server could not answer until the sync ends.
            assert get_path(server, "/healthz", timeout=2).status_code == 200
        finally:
            release_sync.set()
            post.join(timeout=10)
