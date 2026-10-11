# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Sample the API Service from inside the disposable cluster during rollout."""
import argparse
import json
from pathlib import Path
import time

from live_adoption_check import get
from live_adoption_check import kube
from live_adoption_check import require
from provision_live_schedules import write_object

NAME = 'readiness-adoption-availability'
PROBE = '''import json, pathlib, time, urllib.request
samples = failures = consecutive = longest = 0
started = time.monotonic()
while time.monotonic() - started < 3000:
    try:
        with urllib.request.urlopen('http://ml-pipeline.kubeflow:8888/apis/v2beta1/healthz', timeout=2) as response:
            if response.status != 200:
                raise ValueError('health')
        consecutive = 0
    except Exception:
        failures += 1
        consecutive += 1
        longest = max(longest, consecutive)
    samples += 1
    print(json.dumps(dict(samples=samples, failures=failures, longest_failure_streak=longest)), flush=True)
    if pathlib.Path('/tmp/stop').exists():
        break
    time.sleep(1)
'''


def validate_samples(evidence):
    require(evidence['samples'] >= 30, 'insufficient_availability_samples')
    # Publish failures honestly; one failed request can be a connection draining
    # during normal Pod replacement, but consecutive failures mean an outage.
    require(evidence['longest_failure_streak'] <= 1,
            'api_unavailable_during_rollout')
    require(evidence['failures'] / evidence['samples'] <= .01,
            'api_rollout_error_rate_exceeded')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('phase', choices=('start', 'stop'))
    parser.add_argument('--state', required=True)
    args = parser.parse_args()
    if args.phase == 'start':
        kube(
            'create',
            '-f',
            '-',
            value=dict(
                apiVersion='v1',
                kind='Pod',
                metadata=dict(name=NAME, namespace='kubeflow'),
                spec=dict(
                    restartPolicy='Never',
                    automountServiceAccountToken=False,
                    containers=[
                        dict(
                            name='probe',
                            image='python:3.11',
                            command=['python', '-u', '-c', PROBE])
                    ])))
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            if get('kubeflow', 'pod/' + NAME).get('status',
                                                  {}).get('phase') == 'Running':
                return
            time.sleep(2)
        raise ValueError('availability_probe_not_running')
    if get('kubeflow', 'pod/' + NAME).get('status',
                                          {}).get('phase') != 'Succeeded':
        kube('-n', 'kubeflow', 'exec', NAME, '--', 'touch', '/tmp/stop')
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        if get('kubeflow', 'pod/' + NAME).get('status',
                                              {}).get('phase') == 'Succeeded':
            break
        time.sleep(1)
    require(
        get('kubeflow', 'pod/' + NAME).get('status',
                                           {}).get('phase') == 'Succeeded',
        'availability_probe_did_not_finish')
    evidence = json.loads(kube('-n', 'kubeflow', 'logs', NAME).splitlines()[-1])
    evidence.update(
        scope='api_service_health_during_rolling_adoption',
        probe_path='/apis/v2beta1/healthz',
        outcome='inconclusive',
        policy='at most 1% failed probes and no consecutive failures')
    path = Path(args.state) / 'reports/adoption-availability.json'
    write_object(path, evidence)
    validate_samples(evidence)
    evidence['outcome'] = 'passed'
    write_object(path, evidence)


if __name__ == '__main__':
    main()
