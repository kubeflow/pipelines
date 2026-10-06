#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Bounded, read-only kubectl transport for inventory collectors."""

import json
import os
import selectors
import signal
import subprocess
import time

MAX_BYTES = 16 * 1024 * 1024


def kill_process_group(process):
    """Stop kubectl and inherited exec-plugin processes in its private
    session."""
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def kubectl_get(context, namespace, resource, *, all_namespaces=False):
    """Bound time and buffered output; never print kubectl stderr or raw
    objects."""
    command = ['kubectl', '--context', context, '--request-timeout=20s']
    if all_namespaces and namespace:
        raise ValueError('namespace_and_all_namespaces_are_exclusive')
    if namespace:
        command += ['--namespace', namespace]
    command += ['get', resource, '--chunk-size=200', '-o', 'json']
    if all_namespaces:
        command += ['--all-namespaces']
    # Cap bytes while streaming, not after kubectl has filled memory or disk.
    chunks = bytearray()
    try:
        with subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                start_new_session=True) as process:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                deadline = time.monotonic() + 30
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        kill_process_group(process)
                        return None, 'collection_timed_out'
                    chunk = process.stdout.read1(
                        min(65536, MAX_BYTES + 1 - len(chunks)))
                    if not chunk:
                        break
                    chunks.extend(chunk)
                    if len(chunks) > MAX_BYTES:
                        kill_process_group(process)
                        return None, 'collection_exceeded_16_mib'
                try:
                    code = process.wait(
                        timeout=max(0.01, deadline - time.monotonic()))
                except subprocess.TimeoutExpired:
                    kill_process_group(process)
                    return None, 'collection_timed_out'
                if code:
                    return None, 'collection_failed'
    except OSError:
        return None, 'collection_failed'
    try:
        return json.loads(chunks), None
    except (ValueError, UnicodeError):
        return None, 'collection_invalid_json'
