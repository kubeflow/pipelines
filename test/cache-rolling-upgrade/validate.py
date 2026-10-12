#!/usr/bin/env python3
"""Exercise actual cache images, admission and watchers on disposable
Kind/MySQL."""
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

NS = 'cache-a'
PREFIX = 'pipelines.kubeflow.org/'
ENABLED = PREFIX + 'cache_enabled'
CACHE_ID = PREFIX + 'cache_id'
KEY = PREFIX + 'execution_cache_key'
HIT = PREFIX + 'reused_from_cache'
OUTPUT = 'workflows.argoproj.io/outputs'
COMPLETE = 'workflows.argoproj.io/completed'
OLD = 'ghcr.io/kubeflow/kfp-cache-server:2.17.2'
NEW = 'kfp-cache-server:rolling-candidate'
TASK = 'busybox:1.37'
REPORT = {
    'source_version': '2.17.2',
    'source_revision': '2511cdbd74cd531c6633f2e094d235082a32917d',
    'phases': []
}


def run(*args, data=None):
    result = subprocess.run(
        args,
        input=data,
        text=True,
        check=False,
        capture_output=True,
        timeout=240)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return result.stdout.strip()


def k(*args, data=None):
    return run('kubectl', '--request-timeout=60s', *args, data=data)


def apply(obj):
    k('apply', '-f', '-', data=json.dumps(obj))


def get(kind, name, ns=NS):
    return json.loads(k('-n', ns, 'get', kind, name, '-o', 'json'))


def wait(predicate, description, seconds=120):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        value = predicate()
        if value:
            return value
        time.sleep(2)
    raise AssertionError('Timed out: ' + description)


def record(phase, **details):
    REPORT['phases'].append({'phase': phase, 'passed': True, **details})
    print(phase, json.dumps(details), flush=True)


def sql(query):
    return k('-n', NS, 'exec', 'mysql', '--', 'mysql', '-uroot',
             '-pfixture-only', '-N', '-B', '-e', query)


def pods():
    return json.loads(
        k('-n', NS, 'get', 'pods', '-l', 'app=cache-server', '-o',
          'json'))['items']


def endpoint(version):
    # Deterministic admission routing, without stopping either background writer.
    k(
        '-n', NS, 'patch', 'service', 'cache-server', '--type=merge', '-p',
        json.dumps(
            {'spec': {
                'selector': {
                    'app': 'cache-server',
                    'version': version
                }
            }}))

    def selected_endpoint():
        expected = {
            p['metadata']['uid']
            for p in pods()
            if p['metadata']['labels']['version'] == version
        }
        actual = {
            address['targetRef']['uid']
            for subset in get('endpoints', 'cache-server').get('subsets', [])
            for address in subset.get('addresses', [])
        }
        return actual and actual == expected

    wait(selected_endpoint, 'selected webhook endpoint')


def template(marker):
    return {
        'container': {
            'image': TASK,
            'command': ['sh', '-c'],
            'args': ['echo ' + marker]
        }
    }


def key(value, namespace=None):
    if namespace is not None:
        value = {'namespace': namespace, 'template': value}
    return hashlib.sha256(
        json.dumps(value, sort_keys=True,
                   separators=(',', ':')).encode()).hexdigest()


def task(name, marker, namespace=NS, enabled=True, forced_key=None):
    annotations = {
        OUTPUT: json.dumps({'parameters': [{
            'name': 'result',
            'value': name
        }]})
    }
    labels = {ENABLED: str(enabled).lower()}
    if forced_key:
        annotations[KEY] = forced_key
        labels[CACHE_ID] = ''
    pod = {
        'apiVersion': 'v1',
        'kind': 'Pod',
        'metadata': {
            'name': name,
            'namespace': namespace,
            'labels': labels,
            'annotations': annotations
        },
        'spec': {
            'restartPolicy':
                'Never',
            'containers': [{
                'name':
                    'main',
                'image':
                    TASK,
                'imagePullPolicy':
                    'IfNotPresent',
                'command': ['sh', '-c', 'echo done'],
                'env': [{
                    'name': 'ARGO_TEMPLATE',
                    'value': json.dumps(template(marker))
                }]
            }]
        }
    }
    k('create', '-f', '-', data=json.dumps(pod))
    result = get('pod', name, namespace)
    if enabled:
        assert result['metadata']['annotations'].get(KEY), 'webhook bypassed'
    wait(
        lambda: get('pod', name, namespace)['status'].get('phase') ==
        'Succeeded', 'task completion ' + name)
    return result


def complete(name):
    k('-n', NS, 'label', 'pod', name, COMPLETE + '=true', '--overwrite')
    return wait(lambda: get('pod', name)['metadata']['labels'].get(CACHE_ID),
                'watcher persisted ' + name)


def assert_hit(pod, expected):
    actual = pod['metadata']['labels'].get(HIT) == 'true'
    assert actual == expected, (pod['metadata']['name'], actual, expected)


def setup():
    for ns in (NS, 'cache-b'):
        apply({
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'metadata': {
                'name': ns
            }
        })
    apply({
        'apiVersion': 'v1',
        'kind': 'ServiceAccount',
        'metadata': {
            'name': 'cache-server',
            'namespace': NS
        }
    })
    apply({
        'apiVersion':
            'rbac.authorization.k8s.io/v1',
        'kind':
            'Role',
        'metadata': {
            'name': 'cache-server',
            'namespace': NS
        },
        'rules': [{
            'apiGroups': [''],
            'resources': ['pods'],
            'verbs': ['get', 'list', 'watch', 'patch']
        }]
    })
    apply({
        'apiVersion': 'rbac.authorization.k8s.io/v1',
        'kind': 'RoleBinding',
        'metadata': {
            'name': 'cache-server',
            'namespace': NS
        },
        'subjects': [{
            'kind': 'ServiceAccount',
            'name': 'cache-server',
            'namespace': NS
        }],
        'roleRef': {
            'apiGroup': 'rbac.authorization.k8s.io',
            'kind': 'Role',
            'name': 'cache-server'
        }
    })
    apply({
        'apiVersion': 'v1',
        'kind': 'Pod',
        'metadata': {
            'name': 'mysql',
            'namespace': NS,
            'labels': {
                'app': 'mysql'
            }
        },
        'spec': {
            'containers': [{
                'name': 'mysql',
                'image': 'mysql:8.0',
                'env': [{
                    'name': 'MYSQL_ROOT_PASSWORD',
                    'value': 'fixture-only'
                }],
                'readinessProbe': {
                    'exec': {
                        'command': [
                            'mysqladmin', 'ping', '-h', '127.0.0.1',
                            '-pfixture-only'
                        ]
                    }
                }
            }]
        }
    })
    apply({
        'apiVersion': 'v1',
        'kind': 'Service',
        'metadata': {
            'name': 'mysql',
            'namespace': NS
        },
        'spec': {
            'selector': {
                'app': 'mysql'
            },
            'ports': [{
                'port': 3306
            }]
        }
    })
    k('-n', NS, 'wait', '--for=condition=Ready', 'pod/mysql', '--timeout=180s')
    certdir = Path('/tmp/cache-rolling-tls')
    certdir.mkdir(exist_ok=True)
    run('openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days',
        '1', '-keyout', str(certdir / 'key.pem'), '-out',
        str(certdir / 'cert.pem'), '-subj', '/CN=cache-server.cache-a.svc',
        '-addext', 'subjectAltName=DNS:cache-server.cache-a.svc')
    apply({
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'name': 'tls',
            'namespace': NS
        },
        'data': {
            name: base64.b64encode((certdir / name).read_bytes()).decode()
            for name in ('key.pem', 'cert.pem')
        }
    })
    apply({
        'apiVersion': 'v1',
        'kind': 'Service',
        'metadata': {
            'name': 'cache-server',
            'namespace': NS
        },
        'spec': {
            'selector': {
                'app': 'cache-server'
            },
            'ports': [{
                'port': 443,
                'targetPort': 8443
            }]
        }
    })
    apply({
        'apiVersion': 'apps/v1',
        'kind': 'Deployment',
        'metadata': {
            'name': 'cache-server',
            'namespace': NS
        },
        'spec': {
            'replicas': 1,
            'selector': {
                'matchLabels': {
                    'app': 'cache-server'
                }
            },
            'strategy': {
                'type': 'RollingUpdate',
                'rollingUpdate': {
                    'maxSurge': 1,
                    'maxUnavailable': 0
                }
            },
            'template': {
                'metadata': {
                    'labels': {
                        'app': 'cache-server',
                        'version': 'old'
                    }
                },
                'spec': {
                    'serviceAccountName':
                        'cache-server',
                    'containers': [{
                        'name':
                            'server',
                        'image':
                            OLD,
                        'imagePullPolicy':
                            'IfNotPresent',
                        'args': [
                            '--db_host=mysql', '--db_password=fixture-only',
                            '--namespace_to_watch=' + NS
                        ],
                        'env': [{
                            'name': 'CACHE_IMAGE',
                            'value': TASK
                        }],
                        'readinessProbe': {
                            'tcpSocket': {
                                'port': 8443
                            },
                            'periodSeconds': 1
                        },
                        'volumeMounts': [{
                            'name': 'tls',
                            'mountPath': '/etc/webhook/certs'
                        }]
                    }],
                    'volumes': [{
                        'name': 'tls',
                        'secret': {
                            'secretName': 'tls'
                        }
                    }]
                }
            }
        }
    })
    k('-n', NS, 'rollout', 'status', 'deployment/cache-server',
      '--timeout=180s')
    apply({
        'apiVersion':
            'admissionregistration.k8s.io/v1',
        'kind':
            'MutatingWebhookConfiguration',
        'metadata': {
            'name': 'cache-rolling-fixture'
        },
        'webhooks': [{
            'name':
                'cache-server.cache-a.svc',
            'clientConfig': {
                'service': {
                    'name': 'cache-server',
                    'namespace': NS,
                    'path': '/mutate'
                },
                'caBundle':
                    base64.b64encode(
                        (certdir / 'cert.pem').read_bytes()).decode()
            },
            'failurePolicy':
                'Ignore',
            'sideEffects':
                'None',
            'timeoutSeconds':
                5,
            'admissionReviewVersions': ['v1beta1'],
            'objectSelector': {
                'matchLabels': {
                    ENABLED: 'true'
                }
            },
            'rules': [{
                'operations': ['CREATE'],
                'apiGroups': [''],
                'apiVersions': ['v1'],
                'resources': ['pods']
            }]
        }]
    })


def validate():
    setup()
    assert_hit(task('source-populate', 'baseline'), False)
    complete('source-populate')
    assert_hit(task('source-hit', 'baseline'), True)
    assert sql("SELECT COUNT(*) FROM information_schema.columns WHERE "
               "table_schema='cachedb' AND table_name='execution_caches' "
               "AND column_name='Namespace'") == '0'
    # Let this old-format task finish, but retain its Argo completion event until
    # after rollout. Production watchers require both completion and success.
    task('old-inflight', 'inflight')
    record(
        'populated_source',
        image_ids=[
            p['status']['containerStatuses'][0]['imageID'] for p in pods()
        ])
    k(
        '-n', NS, 'patch', 'deployment', 'cache-server', '--type=strategic',
        '-p',
        json.dumps({
            'spec': {
                'minReadySeconds': 900,
                'template': {
                    'metadata': {
                        'labels': {
                            'version': 'new'
                        }
                    },
                    'spec': {
                        'containers': [{
                            'name': 'server',
                            'image': NEW
                        }]
                    }
                }
            }
        }))
    wait(
        lambda: any(p['metadata']['labels']['version'] == 'new' and any(
            c['type'] == 'Ready' and c['status'] == 'True'
            for c in p['status'].get('conditions', []))
                    for p in pods()), 'new process Ready after migration')
    overlap = pods()
    assert {p['metadata']['labels']['version'] for p in overlap
           } == {'old', 'new'}
    assert sql(
        "SELECT COUNT(*) FROM cachedb.execution_caches WHERE Namespace=''"
    ) != '0'
    endpoint('old')
    old_probe = task('old-after-migration', 'baseline')
    assert_hit(old_probe, False)
    assert old_probe['metadata']['annotations'][KEY] == key(
        template('baseline'))
    old_write = task('old-after-migration-write', 'fresh-old-write')
    assert_hit(old_write, False)
    assert old_write['metadata']['annotations'][KEY] == key(
        template('fresh-old-write'))
    complete('old-after-migration-write')
    endpoint('new')
    new_probe = task('new-legacy-cold', 'baseline')
    assert_hit(new_probe, False)
    assert new_probe['metadata']['annotations'][KEY] == key(
        template('baseline'), NS)
    # The old watcher can publish an arbitrary annotated hash. Fixed watcher
    # rejects this disabled-cache fixture, forcing the old writer to win.
    scoped_key = key(template('poison'), NS)
    task(
        'old-writer-scoped-key', 'poison', enabled=False, forced_key=scoped_key)
    old_id = complete('old-writer-scoped-key')
    assert sql('SELECT COUNT(*) FROM cachedb.execution_caches WHERE ID=' +
               old_id + " AND Namespace='' AND ExecutionCacheKey='" +
               scoped_key + "'") == '1'
    poison_probe = task('new-rejects-old-write', 'poison')
    assert_hit(poison_probe, False)
    assert poison_probe['metadata']['annotations'][KEY] == scoped_key
    race = task('mixed-write', 'race')
    assert_hit(race, False)
    assert race['metadata']['annotations'][KEY] == key(template('race'), NS)
    mixed_id = complete('mixed-write')
    record(
        'mixed_version_overlap',
        old_namespace_empty_row=old_id,
        mixed_writer_namespace=sql(
            'SELECT Namespace FROM cachedb.execution_caches WHERE ID=' +
            mixed_id),
        pods=[{
            'uid': p['metadata']['uid'],
            'version': p['metadata']['labels']['version'],
            'image_id': p['status']['containerStatuses'][0]['imageID']
        } for p in overlap])
    assert {p['metadata']['labels']['version'] for p in pods()
           } == {'old', 'new'}
    # Return to ordinary service routing and let the Deployment finish normally.
    k('-n', NS, 'patch', 'service', 'cache-server', '--type=json', '-p',
      '[{"op":"remove","path":"/spec/selector/version"}]')
    k('-n', NS, 'patch', 'deployment', 'cache-server', '--type=merge', '-p',
      '{"spec":{"minReadySeconds":0}}')
    k('-n', NS, 'rollout', 'status', 'deployment/cache-server',
      '--timeout=180s')
    wait(
        lambda: len(pods()) == 1 and pods()[0]['metadata']['labels']['version']
        == 'new', 'all old processes terminated')
    k('-n', NS, 'label', 'pod', 'old-inflight', COMPLETE + '=true')
    time.sleep(5)
    assert not get('pod', 'old-inflight')['metadata']['labels'].get(CACHE_ID)
    assert_hit(task('inflight-cold', 'inflight'), False)
    complete('inflight-cold')
    assert_hit(task('inflight-warm', 'inflight'), True)
    assert_hit(task('scoped-populate', 'poison'), False)
    complete('scoped-populate')
    scoped_hit = task('scoped-hit', 'poison')
    assert_hit(scoped_hit, True)
    assert scoped_hit['metadata']['annotations'][OUTPUT] == get(
        'pod', 'scoped-populate')['metadata']['annotations'][OUTPUT]
    assert_hit(
        task('other-namespace-miss', 'poison', namespace='cache-b'), False)
    assert_hit(task('baseline-cold', 'baseline'), False)
    complete('baseline-cold')
    assert_hit(task('baseline-warm', 'baseline'), True)
    record(
        'converged_enforcement',
        old_processes=0,
        same_namespace_hit=True,
        cross_namespace_hit=False,
        old_inflight_cold_then_warm=True,
        legacy_scoped_hash_ignored=True)


if __name__ == '__main__':
    REPORT['target_revision'] = run('git', 'rev-parse', 'HEAD')
    try:
        validate()
        REPORT['passed'] = True
    finally:
        out = Path(
            os.environ.get('CACHE_ROLLING_REPORT',
                           '/tmp/cache-rolling-report.json'))
        out.write_text(json.dumps(REPORT, indent=2) + '\n')
