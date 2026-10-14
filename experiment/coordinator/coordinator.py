#!/usr/bin/env python3
"""Bounded disposable APF pairing coordinator. No operation runs on import."""
import argparse
import datetime as dt
import hashlib
import json
import math
import os
from pathlib import Path
import re
import selectors
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request

ORDER = (20, -1, -1, 20, 20, -1)
UTC = dt.timezone.utc
MAX_CAPTURE = 192 << 20


def utc():
    return dt.datetime.now(UTC)


def iso(value):
    return value.isoformat().replace('+00:00', 'Z')


def stamp(value):
    return dt.datetime.fromisoformat(value.replace('Z', '+00:00'))


def load(path, limit=2 << 20):
    with open(path, 'rb') as stream:
        data = stream.read(limit + 1)
    if len(data) > limit:
        raise RuntimeError('JSON input size bound exceeded')
    return json.loads(data)


def digest(path):
    with open(path, 'rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def save(path, value):
    path = Path(path)
    data = (json.dumps(value, sort_keys=True, indent=2) + '\n').encode()
    if len(data) > 32 << 20:
        raise RuntimeError('JSON output size bound exceeded')
    temporary = path.with_suffix(path.suffix + '.tmp')
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def bounded(argv, timeout=30, limit=1 << 20):
    """Never persist subprocess output; return only bounded successful stdout."""
    process = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                               start_new_session=True)
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    data = bytearray()
    deadline = time.monotonic() + timeout
    try:
        while selector.get_map():
            if time.monotonic() >= deadline:
                raise RuntimeError('bounded command deadline')
            for key, _ in selector.select(.1):
                chunk = os.read(key.fd, 65536)
                if not chunk:
                    selector.unregister(key.fileobj)
                    continue
                data.extend(chunk)
                if len(data) > limit:
                    raise RuntimeError('bounded command output limit')
        process.wait(timeout=max(.1, deadline - time.monotonic()))
        if process.returncode:
            raise RuntimeError(f'command failed with exit {process.returncode}')
        return bytes(data)
    finally:
        stop(process)
        selector.close()
        process.stdout.close()


def stop(process):
    if process is None or process.poll() is not None:
        return
    os.killpg(process.pid, signal.SIGTERM)
    try:
        process.wait(timeout=2)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGKILL)
        process.wait(timeout=2)


def spawn(argv):
    return subprocess.Popen(argv, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                            start_new_session=True)


def total_seconds(workload):
    return sum(p['seconds'] for p in workload['phases']) + 2 * workload['probe_seconds'] + workload['drain_seconds']


def validate(c):
    if not re.fullmatch('[0-9a-f]{40}', c['source_sha']):
        raise RuntimeError('exact source SHA required')
    if not re.fullmatch('kfp-apf-representative-[a-z0-9-]+', c['cluster']):
        raise RuntimeError('dedicated cluster name required')
    if not 1024 <= c['port'] <= 65535:
        raise RuntimeError('invalid local port')
    if c['api_image'] == '' or not c['api_image_ids']:
        raise RuntimeError('pinned API image and actual runtime image IDs required')
    workload = load(c['workload_template'])
    observer = load(c['observer_template'])
    names = [p['name'] for p in workload['phases']]
    if names != ['normal', 'sustained', 'burst1', 'rest1', 'burst2', 'rest2', 'burst3', 'recovery']:
        raise RuntimeError('freeze all planned load phases before pairing')
    if [p['seconds'] for p in workload['phases']] != [120, 180, 10, 30, 10, 30, 10, 120]:
        raise RuntimeError('phase durations differ from declared experiment')
    if len(workload['tenants']) != 4 or workload['request_seconds'] != 10 or workload['drain_seconds'] != 600:
        raise RuntimeError('four tenants, 10s requests and 600s drain required')
    for tenant in workload['tenants']:
        if {f['name'] for f in tenant['fixtures']} != {'flip_coin', 'collected_parameters', 'if_else_with_oneof_artifacts'}:
            raise RuntimeError('all three fixtures required per tenant')
    if total_seconds(workload) > 2700 or observer['rps'] * total_seconds(workload) > 90000:
        raise RuntimeError('observer workload budget exceeded')
    if not re.fullmatch('[0-9a-f]{64}', c['prewarm_evidence_sha256']) or digest(c['prewarm_evidence']) != c['prewarm_evidence_sha256']:
        raise RuntimeError('verified prewarm evidence missing')
    prewarm = load(c['prewarm_evidence'])
    if prewarm.get('source_sha') != c['source_sha'] or not prewarm.get('fixture_outputs_verified'):
        raise RuntimeError('prewarm output correctness evidence required before measurement')
    for program in ('workload_binary', 'observer_binary'):
        if not Path(c[program]).is_file():
            raise RuntimeError('compiled candidate binary unavailable')
    return workload, observer


class Owned:
    def __init__(self, config):
        self.c = config
        self.state = Path(config['state'])
        self.context = 'kind-' + config['cluster']
        self.kubeconfig = str(self.state / 'kubeconfig')

    def k(self, *args, timeout=30):
        return bounded(['kubectl', '--kubeconfig', self.kubeconfig, '--context', self.context,
                        '--request-timeout=20s', *args], timeout)

    def guard(self):
        owner = load(self.state / 'owner.json')
        if owner.get('cluster') != self.c['cluster'] or owner.get('source_sha') != self.c['source_sha']:
            raise RuntimeError('owned cluster marker mismatch')
        uid = self.k('get', 'namespace', 'kube-system', '-o', 'jsonpath={.metadata.uid}').decode()
        if not owner.get('namespace_uid') or uid != owner['namespace_uid']:
            raise RuntimeError('owned cluster UID mismatch')
        return uid

    def switch(self, qps, directory):
        self.guard()
        before = json.loads(self.k('-n', 'kubeflow', 'get', 'deployment', 'ml-pipeline', '-o', 'json'))
        containers = before['spec']['template']['spec']['containers']
        api = next(x for x in containers if x['name'] == 'ml-pipeline-api-server')
        if api['image'] != self.c['api_image']:
            raise RuntimeError('API image changed from frozen treatment image')
        save(directory / 'switch-dispatch.json', {'qps': qps, 'burst': 40, 'at': iso(utc()), 'image': api['image']})
        self.k('-n', 'kubeflow', 'set', 'env', 'deployment/ml-pipeline', f'CLIENTQPS={qps}', 'CLIENTBURST=40')
        self.k('-n', 'kubeflow', 'rollout', 'status', 'deployment/ml-pipeline', '--timeout=600s', timeout=620)
        self.guard()
        after = json.loads(self.k('-n', 'kubeflow', 'get', 'deployment', 'ml-pipeline', '-o', 'json'))
        api = next(x for x in after['spec']['template']['spec']['containers'] if x['name'] == 'ml-pipeline-api-server')
        env = {x['name']: x.get('value') for x in api.get('env', [])}
        if api['image'] != self.c['api_image'] or env.get('CLIENTQPS') != str(qps) or env.get('CLIENTBURST') != '40':
            raise RuntimeError('resolved treatment mismatch')
        labels = after['spec']['selector']['matchLabels']
        selector = ','.join(f'{k}={v}' for k, v in sorted(labels.items()))
        pods = json.loads(self.k('-n', 'kubeflow', 'get', 'pods', '-l', selector, '-o', 'json'))['items']
        identities = []
        for pod in pods:
            if pod['metadata'].get('deletionTimestamp'):
                continue
            for item in pod.get('status', {}).get('containerStatuses', []):
                if item['name'] == 'ml-pipeline-api-server':
                    identities.append({'pod_uid': pod['metadata']['uid'], 'image_id': item['imageID'], 'ready': item['ready'], 'restart_count': item['restartCount']})
        if not identities or any(not x['ready'] or x['image_id'] not in self.c['api_image_ids'] for x in identities):
            raise RuntimeError('ready API runtime image identity mismatch')
        save(directory / 'switch-result.json', {'qps': qps, 'burst': 40, 'at': iso(utc()), 'identities': identities})

    def forward(self):
        return spawn(['kubectl', '--kubeconfig', self.kubeconfig, '--context', self.context,
                      '-n', 'kubeflow', 'port-forward', '--address=127.0.0.1',
                      'svc/ml-pipeline', f'{self.c["port"]}:8888'])


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise urllib.error.HTTPError(args[0].full_url, 302, 'redirect rejected', None, None)


def endpoint_ready(endpoint, forward):
    opener = urllib.request.build_opener(NoRedirect)
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        if forward.poll() is not None:
            raise RuntimeError('owned port-forward exited')
        try:
            with opener.open(endpoint + '/apis/v2beta1/healthz', timeout=2) as response:
                if response.status == 200:
                    return
        except (OSError, urllib.error.HTTPError):
            pass
        time.sleep(.2)
    raise RuntimeError('KFP endpoint readiness deadline')


def wait_file(path, deadline, processes):
    while utc() < deadline:
        if any(p.poll() is not None for p in processes):
            raise RuntimeError('peer exited during launch')
        if path.is_file():
            return
        time.sleep(.025)
    raise RuntimeError('shared launch deadline missed')


def verify_baseline(marker, timeline, start):
    if marker.get('timeline_sha256') != digest(timeline) or stamp(marker['start_at']) != start or stamp(marker['captured_at']) >= start:
        raise RuntimeError('baseline does not bind shared timeline before start')


def capture_manifest(directory):
    files = []
    size = 0
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise RuntimeError('symlink in evidence')
        if not path.is_file() or path.name in {'evidence-captured.json'} or path.name.endswith('.tmp'):
            continue
        size += path.stat().st_size
        if size > MAX_CAPTURE:
            raise RuntimeError('sanitized evidence exceeds capture budget')
        files.append({'path': str(path.relative_to(directory)), 'sha256': digest(path), 'bytes': path.stat().st_size})
    save(directory / 'evidence-captured.json', {'files': files, 'bytes': size, 'captured_at': iso(utc()), 'cleanup_performed': False})


def arm(c, workload_template, observer_template, directory, index):
    directory.mkdir(mode=0o700)
    qps = ORDER[index]
    owner = Owned(c)
    processes = []
    failure = None
    try:
        owner.switch(qps, directory)
        forward = owner.forward()
        processes.append(forward)
        endpoint = f'http://127.0.0.1:{c["port"]}'
        endpoint_ready(endpoint, forward)
        # No accepted pipeline is created until all three child launch inputs exist.
        start = utc() + dt.timedelta(seconds=45)
        workload = dict(workload_template, endpoint=endpoint, start_at=iso(start), label=f'pair{index//2+1}-qps{qps}')
        observer = dict(observer_template, start_at=iso(start), seconds=total_seconds(workload),
                        kubeconfig=owner.kubeconfig, context=owner.context, cluster_uid=owner.guard(),
                        timeline_file=str(directory / 'workload' / 'timeline.json'))
        save(directory / 'workload-config.json', workload)
        save(directory / 'observer-config.json', observer)
        work = spawn([c['workload_binary'], '--config', str(directory / 'workload-config.json'), '--out', str(directory / 'workload')])
        processes.append(work)
        wait_file(directory / 'workload' / 'timeline.json', start-dt.timedelta(seconds=5), processes)
        watch = spawn([c['observer_binary'], '--config', str(directory / 'observer-config.json'), '--out', str(directory / 'observer')])
        processes.append(watch)
        marker = directory / 'observer' / 'baseline-ready.json'
        wait_file(marker, start, processes)
        verify_baseline(load(marker), directory / 'workload' / 'timeline.json', start)
        if utc() >= start:
            raise RuntimeError('baseline coordination completed after start')
        save(directory / 'launch.json', {'start_at': iso(start), 'timeline_sha256': digest(directory/'workload'/'timeline.json'), 'qps':qps})
        deadline = start + dt.timedelta(seconds=total_seconds(workload)+15)
        while utc() < deadline:
            if forward.poll() is not None:
                raise RuntimeError('port-forward died during measurement')
            codes = [work.poll(), watch.poll()]
            if any(x is not None and x != 0 for x in codes):
                raise RuntimeError('workload/observer peer failure; both canceled')
            if all(x == 0 for x in codes):
                break
            time.sleep(.1)
        else:
            raise RuntimeError('full timeline/drain watchdog expired')
        save(directory/'process-results.json', {'workload_exit':work.returncode,'observer_exit':watch.returncode,'qps':qps})
    except BaseException as error:
        failure = type(error).__name__
        # Only safe operation stage/type, never child stdout, tokens or raw API errors.
        save(directory/'interrupted.json', {'failure_type':failure,'at':iso(utc()),'requires_accepted_run_reconciliation':True})
    finally:
        for process in reversed(processes):
            stop(process)
        capture_manifest(directory)
    if failure:
        raise RuntimeError('arm interrupted; evidence retained, reconcile accepted runs before retry')


def read_events(path):
    used = 0
    with open(path) as stream:
        for line in stream:
            used += len(line.encode())
            if used > 128 << 20 or len(line) > 8 << 20:
                raise RuntimeError('event evidence exceeds bounds')
            yield json.loads(line)


def evidence_summary(directory):
    w = load(directory/'workload'/'results.json',32<<20)
    o = load(directory/'observer'/'summary.json',32<<20)
    longest = current = 0
    longest_health = health = 0
    snapshots = 0
    for event in read_events(directory/'observer'/'events.jsonl'):
        if event.get('kind') != 'snapshot' or event.get('index',-1) < 0:
            continue
        snapshots += 1
        priority = o['priority_name']
        queue = any(m['name']=='apiserver_flowcontrol_current_inqueue_requests' and m['labels'].get('priority_level')==priority and m['value']>0 for m in event.get('metrics',[]))
        current = current+1 if queue and event.get('positive_seats') and not event.get('missing') else 0
        longest = max(longest,current)
        health = health+1 if not event.get('ready') or not event.get('live') else 0
        longest_health = max(longest_health,health)
    return {'workflow_success':w['all_recorded_runs_succeeded'], 'post_load_permission_complete':w['permission_probe_coverage_complete'], 'observer_complete':o['observation_complete'], 'shared_priority_live_verified':o.get('kfp_live_classification_verified',False), 'longest_queued_seconds':longest,'longest_health_missing_or_failure_seconds':longest_health,'read_stats':w['read_stats'],'snapshots':snapshots}


def evaluate(root):
    gates = []
    for pair in range(3):
        dirs=[root/f'arm-{pair*2+1}',root/f'arm-{pair*2+2}']
        by_qps={ORDER[pair*2+i]:evidence_summary(path) for i,path in enumerate(dirs)}
        control,treatment=by_qps[20],by_qps[-1]
        checks={}
        for label,result in [('control',control),('treatment',treatment)]:
            checks[label+'_workflows']=result['workflow_success']
            checks[label+'_post_load_auth']=result['post_load_permission_complete']
            checks[label+'_observer']=result['observer_complete']
            checks[label+'_shared_priority']=result['shared_priority_live_verified']
            checks[label+'_sustained_queue']=result['longest_queued_seconds']>=10
            checks[label+'_health_observation']=result['longest_health_missing_or_failure_seconds']<=5
        for key,a in control['read_stats'].items():
            b=treatment['read_stats'].get(key)
            if b is None or a['count']==0 or b['count']==0:
                checks[key]=False
                continue
            if key.startswith('normal/'):
                checks[key+'_normal']=b['errors']/b['count']<=.001 and b['drops']==0 and b['p95_ms']<=1.1*a['p95_ms']+5
            if not key.endswith('/noisy'):
                checks[key+'_fairness']=sum(b['successful_reads_by_10s'].values())>=.9*sum(a['successful_reads_by_10s'].values()) and b['p95_ms']<=1.25*a['p95_ms']+10 and all(x>0 for x in b['successful_reads_by_10s'].values())
        gates.append({'pair':pair+1,'available_checks':checks,'available_checks_pass':all(checks.values())})
    missing=['during_load_read_and_write_denials','semantic_fixture_output_assertions','recurring_swf_lifecycle','actual_kfp_apf_headers','same_priority_competitor_paired_window_thresholds','direct_etcd_and_restart_oom_health','60second_recovery_evaluation','no_deadline_server_termination_evidence','minimum_supported_kubernetes','apf_disabled_override','full_ci_comparison']
    result={'pairs':gates,'acceptance_complete':False,'missing_gates':missing,'default_change_supported':False}
    save(root/'evaluation.json',result)
    return result


def disconnect_child(endpoint,run_id,token_file):
    token=Path(token_file).read_text().strip()
    request=urllib.request.Request(endpoint+'/apis/v2beta1/runs/'+run_id,headers={'Authorization':'Bearer '+token})
    # Intentionally no HTTP timeout; the parent owns a hard process watchdog.
    try:
        with urllib.request.build_opener(NoRedirect).open(request) as response:
            print(json.dumps({'http_status':response.status,'returned_before_disconnect':True}),flush=True)
    except Exception:
        print(json.dumps({'transport_or_http_error':True}),flush=True)


def main():
    parser=argparse.ArgumentParser();parser.add_argument('action',choices=['validate','run','evaluate']);parser.add_argument('--config');parser.add_argument('--out',required=True);args=parser.parse_args()
    if args.action=='evaluate':
        evaluate(Path(args.out));return
    c=load(args.config);workload,observer=validate(c)
    if args.action=='validate':
        print('Offline inputs validated; no cluster or workload operation executed.');return
    root=Path(args.out);root.mkdir(mode=0o700)
    save(root/'frozen-inputs.json',{'source_sha':c['source_sha'],'workload_template_sha256':digest(c['workload_template']),'observer_template_sha256':digest(c['observer_template']),'order':ORDER,'prewarm_evidence_sha256':c['prewarm_evidence_sha256']})
    for index in range(6):
        arm(c,workload,observer,root/f'arm-{index+1}',index)
    evaluate(root)


if __name__=='__main__':
    try:
        main()
    except Exception as error:
        print('Coordinator stopped: '+type(error).__name__+'; preserved evidence requires review.',file=sys.stderr)
        raise SystemExit(1)
