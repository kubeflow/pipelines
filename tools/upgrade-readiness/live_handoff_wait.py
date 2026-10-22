# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Single-request handoff acceptance in the disposable adoption cluster."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import http.client
import json
from pathlib import Path
import socket
import subprocess
import time
import uuid

from fixture_http import FixtureClient
from live_adoption_check import get
from live_adoption_check import kube
from live_adoption_check import require
from live_adoption_check import sql
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
from provision_live_schedules import write_object

RELEASE_FILE = '/tmp/readiness-release-writer'


def request(port, token, method, path, body=None, timeout=40):
    """Exactly one HTTP request, without a redirect handler or retry
    adapter."""
    connection = http.client.HTTPConnection('127.0.0.1', port, timeout=timeout)
    started = time.monotonic()
    try:
        connection.request(method, path,
                           json.dumps(body) if body else None, {
                               'Authorization': 'Bearer ' + token,
                               'Content-Type': 'application/json'
                           })
        response = connection.getresponse()
        raw = response.read(4 * 1024 * 1024 + 1)
        require(len(raw) <= 4 * 1024 * 1024, 'handoff_response_too_large')
        return dict(
            status=response.status,
            body=json.loads(raw or b'{}'),
            elapsed=time.monotonic() - started,
            attempts=1)
    finally:
        connection.close()


def wait_for(check, reason, timeout=180):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        result = check()
        if result:
            return result
        time.sleep(1)
    raise ValueError(reason)


def current_pods(app):
    return json.loads(
        kube('-n', 'kubeflow', 'get', 'pods', '-l', 'app=' + app, '-o',
             'json'))['items']


def ready(pod):
    return not pod['metadata'].get('deletionTimestamp') and any(
        c['type'] == 'Ready' and c['status'] == 'True'
        for c in pod.get('status', {}).get('conditions', []))


def inventory():
    return dict(
        jobs=sql('jobs', ('UUID', 'Name', 'Enabled'),
                 f"WHERE Namespace='{NAMESPACE}'"),
        schedules=sorted(
            (p['metadata']['uid'], p['spec'].get('enabled'))
            for p in get(NAMESPACE, 'scheduledworkflows')['items']))


def release(pod):
    kube('-n', 'kubeflow', 'exec', pod, '-c', 'ml-pipeline-scheduledworkflow',
         '--', 'touch', RELEASE_FILE)


def validate_timeout(result):
    require(result['status'] == 503, 'handoff_timeout_not_503')
    require(result['body'].get('code') == 14, 'handoff_timeout_not_unavailable')
    require(
        any(
            detail.get('@type') == 'type.googleapis.com/google.rpc.ErrorInfo'
            and detail.get('reason') == 'SCHEDULE_HANDOFF_IN_PROGRESS' and
            detail.get('domain') == 'kubeflow.org'
            for detail in result['body'].get('details', [])
            if isinstance(detail, dict)), 'handoff_timeout_missing_error_info')
    require(28 <= result['elapsed'] < 40, 'handoff_timeout_not_bounded')
    require(result['attempts'] == 1, 'handoff_timeout_retried')


def exercise(port, token, payload, unblock):
    path = '/apis/v2beta1/recurringruns'
    baseline = inventory()
    results = {}
    # Closing the connection cancels the gateway context before any write.
    canceled = dict(payload, display_name=payload['display_name'] + '-cancel')
    try:
        request(port, token, 'POST', path, canceled, timeout=2)
        raise ValueError('handoff_canceled_request_finished_early')
    except (TimeoutError, socket.timeout):
        results['cancellation'] = dict(attempts=1, outcome='client_timeout')
    require(inventory() == baseline, 'handoff_cancellation_mutated_state')
    timed = dict(payload, display_name=payload['display_name'] + '-timeout')
    results['timeout'] = request(port, token, 'POST', path, timed)
    validate_timeout(results['timeout'])
    require(inventory() == baseline, 'handoff_timeout_mutated_state')
    with ThreadPoolExecutor(max_workers=1) as pool:
        pending = pool.submit(request, port, token, 'POST', path, payload)
        time.sleep(3)
        require(not pending.done(), 'handoff_request_did_not_wait')
        results['health'] = request(port, token, 'GET', '/apis/v2beta1/healthz')
        require(results['health']['status'] == 200,
                'handoff_health_unavailable')
        results['read'] = request(
            port, token, 'GET',
            '/apis/v2beta1/recurringruns?namespace=' + NAMESPACE)
        require(results['read']['status'] == 200, 'handoff_read_unavailable')
        require(results['health']['elapsed'] < 3, 'handoff_health_blocked')
        require(results['read']['elapsed'] < 3, 'handoff_read_blocked')
        require(inventory() == baseline,
                'handoff_pending_request_mutated_state')
        require(not pending.done(), 'handoff_request_finished_before_release')
        unblock()
        results['success'] = pending.result(timeout=37)
    success = results['success']
    require(success['status'] == 200, 'handoff_single_request_failed')
    require(3 <= success['elapsed'] < 30, 'handoff_success_not_within_bound')
    require(success['attempts'] == 1, 'handoff_success_retried')
    current = inventory()
    require(
        len(current['jobs']) == len(baseline['jobs']) + 1,
        'handoff_request_not_exactly_one_job')
    require(
        len(current['schedules']) == len(baseline['schedules']) + 1,
        'handoff_request_not_exactly_one_schedule')
    # Recheck after release: canceled/expired requests must never commit later.
    time.sleep(3)
    require(inventory() == current, 'handoff_expired_request_committed_late')
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--state', required=True)
    args = parser.parse_args()
    state = Path(args.state)
    report_path = state / 'reports/handoff-wait.json'
    report = dict(
        outcome='inconclusive',
        scope='single_request_server_handoff_wait',
        candidate_sha=subprocess.check_output(['git', 'rev-parse', 'HEAD'],
                                              text=True,
                                              timeout=10).strip(),
        overlap='controlled_prestop_drain_during_real_restart')
    write_object(report_path, report)
    fixture = read_object(state / 'fixture/state.json')
    # Reuse the transport's strict token validation; never include credentials in evidence.
    token = FixtureClient('http://127.0.0.1:8888', state / 'token').token
    # Keep the real writer alive during a bounded, explicitly released preStop.
    # No process is force-deleted and the ordinary shutdown follows release.
    controller = get('kubeflow', 'deployment/ml-pipeline-scheduledworkflow')
    original_template = controller['spec']['template']
    drain_patch = {
        'spec': {
            'template': {
                'spec': {
                    'terminationGracePeriodSeconds':
                        150,
                    'containers': [{
                        'name': 'ml-pipeline-scheduledworkflow',
                        'lifecycle': {
                            'preStop': {
                                'exec': {
                                    'command': [
                                        '/bin/sh', '-c',
                                        'i=0; while [ ! -f ' + RELEASE_FILE +
                                        ' ] && [ $i -lt 120 ]; do sleep 1; i=$((i+1)); done'
                                    ]
                                }
                            }
                        }
                    }]
                }
            }
        }
    }
    forward = None
    held = False
    try:
        kube('-n', 'kubeflow', 'patch',
             'deployment/ml-pipeline-scheduledworkflow', '--type=strategic',
             '-p', json.dumps(drain_patch))
        kube('-n', 'kubeflow', 'rollout', 'status',
             'deployment/ml-pipeline-scheduledworkflow', '--timeout=180s')
        old = next(
            p for p in current_pods('ml-pipeline-scheduledworkflow')
            if ready(p))
        old_name = old['metadata']['name']
        old_api_uids = {
            p['metadata']['uid'] for p in current_pods('ml-pipeline')
        }
        held = True
        drain_started = time.monotonic()
        kube('-n', 'kubeflow', 'rollout', 'restart', 'deployment/ml-pipeline',
             'deployment/ml-pipeline-scheduledworkflow')
        wait_for(
            lambda: get('kubeflow', 'pod/' + old_name)['metadata'].get(
                'deletionTimestamp'), 'handoff_old_writer_not_terminating')
        require(
            get('kubeflow', 'pod/' + old_name)['status']['phase'] == 'Running',
            'handoff_retiring_writer_not_running')
        new = wait_for(
            lambda: next((p for p in current_pods('ml-pipeline')
                          if p['metadata']['uid'] not in old_api_uids and ready(
                              p)), None), 'handoff_replacement_api_not_ready')
        report['retiring_controller_uid'] = old['metadata']['uid']
        report['replacement_api_uid'] = new['metadata']['uid']
        with (state / 'handoff-port-forward.log').open('w+') as log:
            forward = subprocess.Popen([
                'kubectl', '--context', 'kind-kfp-readiness', '-n', 'kubeflow',
                'port-forward', '--address=127.0.0.1',
                'pod/' + new['metadata']['name'], '18888:8888'
            ],
                                       stdout=log,
                                       stderr=subprocess.STDOUT)

            def forwarded():
                require(forward.poll() is None, 'handoff_forward_exited')
                log.seek(0)
                return 'Forwarding from 127.0.0.1:18888' in log.read()

            wait_for(forwarded, 'handoff_forward_not_ready', timeout=30)
            payload = dict(
                display_name='handoff-wait-' + uuid.uuid4().hex,
                experiment_id=fixture['experiment_id'],
                pipeline_spec=read_object(state / 'pipeline.json'),
                max_concurrency='1',
                mode='DISABLE',
                no_catchup=True,
                trigger=dict(periodic_schedule=dict(interval_second='3600')))

            def unblock():
                nonlocal held
                retiring = get('kubeflow', 'pod/' + old_name)
                require(
                    retiring['status']['phase'] == 'Running' and
                    retiring['metadata'].get('deletionTimestamp'),
                    'handoff_retiring_writer_not_live_at_release')
                require(time.monotonic() - drain_started < 110,
                        'handoff_controlled_drain_budget_exhausted')
                kube('-n', 'kubeflow', 'exec', old_name, '-c',
                     'ml-pipeline-scheduledworkflow', '--', '/bin/sh', '-c',
                     'test ! -e ' + RELEASE_FILE)
                release(old_name)
                held = False

            require(
                get('kubeflow',
                    'pod/' + old_name)['status']['phase'] == 'Running',
                'handoff_retiring_writer_stopped_before_requests')
            report['requests'] = exercise(18888, token, payload, unblock)
            report['outcome'] = 'passed'
    finally:
        try:
            if held:
                release(old_name)
        finally:
            if forward is not None:
                forward.terminate()
                try:
                    forward.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    forward.kill()
                    forward.wait(timeout=5)
            # Restore the fixture's ordinary lifecycle; never force-delete a live writer.
            kube(
                '-n', 'kubeflow', 'patch',
                'deployment/ml-pipeline-scheduledworkflow', '--type=json', '-p',
                json.dumps([
                    dict(
                        op='replace',
                        path='/spec/template',
                        value=original_template)
                ]))
            write_object(report_path, report)


if __name__ == '__main__':
    main()
