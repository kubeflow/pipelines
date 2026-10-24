# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise proxy registration and restart contracts without network access."""

import http.server
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile
import threading
import unittest


class RegistrationTest(unittest.TestCase):

    def setUp(self):
        if shutil.which("jq") is None:
            self.fail(
                "Registration contract tests require the real jq executable")
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.log = self.root / "commands.jsonl"
        source = Path(__file__).with_name(
            "attempt-register-vm-on-proxy.sh").read_text()
        self.script = self.root / "registration.sh"
        self.assertEqual(source.count("/opt/bin/proxy-forwarding-agent"), 1)
        self.script.write_text(
            source.replace(
                "/opt/bin/proxy-forwarding-agent",
                str(self.bin / "proxy-forwarding-agent"),
            ))
        fake = self.bin / "fake"
        fake.write_text("#!" + sys.executable + "\n" +
                        r"""import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ['COMMAND_LOG'], 'a') as log:
    print(json.dumps({'name': name, 'args': args,
                      'credentials': os.environ.get('GOOGLE_APPLICATION_CREDENTIALS')}), file=log)
if name == 'kubectl':
    if args == ['get', 'configmap', 'inverse-proxy-config', '-o', 'json']:
        print(os.environ['CONFIGMAP'])
    elif args[:4] != ['patch', 'configmap/inverse-proxy-config', '--type', 'merge']:
        sys.exit('Unexpected kubectl invocation')
elif name == 'gcloud':
    if args == ['auth', 'print-access-token']:
        print('test-access-token')
    elif args != ['auth', 'activate-service-account', '--key-file=/test-only/service-account.json']:
        sys.exit('Unexpected gcloud invocation')
elif name == 'curl':
    if 'http://metadata.google.internal/computeMetadata/v1/instance/zone' in args:
        print('projects/test/zones/us-central1-a')
    elif any('instance/service-accounts/default/identity?' in arg for arg in args):
        print('test-vm-identity')
    elif args[-1] == os.environ.get('EXPECTED_PROXY_URL', 'https://registration.invalid') + '/request-service-account-endpoint':
        print(json.dumps({'hostname': 'public.invalid', 'backendID': 'fresh-backend'}))
    else:
        sys.exit('Unexpected curl invocation')
elif name == 'wget':
    if args != ['https://storage.googleapis.com/ml-pipeline/proxy-agent-config.json']:
        sys.exit('Unexpected config URL')
    pathlib.Path('proxy-agent-config.json').write_text(json.dumps({
        'agent-docker-containers': {'latest': {'proxy-urls': {
            'us-central1': [os.environ['EXPECTED_PROXY_URL']]}}}}))
elif name == 'sleep':
    if args != ['20']:
        sys.exit('Unexpected retry delay')
elif name != 'proxy-forwarding-agent':
    sys.exit('Unexpected network/config fetch: ' + name)
""")
        fake.chmod(0o755)
        for name in (
                "kubectl",
                "gcloud",
                "curl",
                "wget",
                "sleep",
                "proxy-forwarding-agent",
        ):
            (self.bin / name).symlink_to(fake)
        self.environment = {
            "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
            "HOME": str(self.root),
            "COMMAND_LOG": str(self.log),
            "CONFIGMAP": "{}",
            "GOOGLE_APPLICATION_CREDENTIALS": "/test-only/service-account.json",
            "PROXY_URL": "https://registration.invalid",
            "DEBUG": "false",
            "PROXY_TIMEOUT": "45s",
            "ML_PIPELINE_UI_SERVICE_HOST": "pipeline-ui",
            "ML_PIPELINE_UI_SERVICE_PORT": "3000",
            "HEALTH_CHECK_PATH": "/healthz",
            "HEALTH_CHECK_INTERVAL_SECONDS": "7",
            "HEALTH_CHECK_UNHEALTHY_THRESHOLD": "4",
        }

    def run_registration(self):
        subprocess.run(
            ["bash", str(self.script)],
            env=self.environment,
            cwd=self.root,
            check=True,
            capture_output=True,
            text=True,
            timeout=10,
        )
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def assert_agent(self, calls, proxy, backend):
        agents = [
            call for call in calls if call["name"] == "proxy-forwarding-agent"
        ]
        self.assertEqual(len(agents), 1)
        self.assertEqual(
            agents[0]["args"],
            [
                "--debug=false",
                "--proxy=" + proxy + "/",
                "--proxy-timeout=45s",
                "--backend=" + backend,
                "--host=pipeline-ui:3000",
                "--shim-websockets=true",
                "--shim-path=websocket-shim",
                "--health-check-path=/healthz",
                "--health-check-interval-seconds=7",
                "--health-check-unhealthy-threshold=4",
            ],
        )
        self.assertEqual(agents[0]["credentials"],
                         "/test-only/service-account.json")
        self.assertEqual(calls[-1]["name"], "proxy-forwarding-agent")

    def test_existing_configmap_reuses_endpoint_without_registration(self):
        self.environment["CONFIGMAP"] = json.dumps({
            "data": {
                "Hostname": "existing.invalid",
                "ProxyUrl": "https://existing.invalid",
                "BackendId": "existing-backend",
            }
        })
        calls = self.run_registration()
        self.assert_agent(calls, "https://existing.invalid", "existing-backend")
        self.assertEqual(
            [call["name"] for call in calls],
            ["kubectl", "kubectl", "kubectl", "proxy-forwarding-agent"],
        )
        self.assertTrue(all(call["args"][0] == "get" for call in calls[:-1]))

    def test_fresh_registration_authenticates_patches_and_forwards_endpoint(
            self):
        calls = self.run_registration()
        self.assert_agent(calls, "https://registration.invalid",
                          "fresh-backend")
        gcloud = [call["args"] for call in calls if call["name"] == "gcloud"]
        self.assertEqual(
            gcloud,
            [
                [
                    "auth",
                    "activate-service-account",
                    "--key-file=/test-only/service-account.json",
                ],
                ["auth", "print-access-token"],
            ],
        )
        curls = [call["args"] for call in calls if call["name"] == "curl"]
        self.assertEqual(len(curls), 3)
        self.assertIn("Metadata-Flavor: Google", curls[0])
        self.assertIn("Metadata-Flavor: Google", curls[1])
        self.assertIn(
            "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?format=full&audience=https://registration.invalid/request-service-account-endpoint",
            curls[1],
        )
        self.assertEqual(
            curls[2],
            [
                "-H",
                "Authorization: Bearer test-access-token",
                "-H",
                "X-Inverting-Proxy-VM-ID: test-vm-identity",
                "-d",
                "",
                "https://registration.invalid/request-service-account-endpoint",
            ],
        )
        patches = [
            call for call in calls
            if call["name"] == "kubectl" and call["args"][0] == "patch"
        ]
        self.assertEqual(len(patches), 2)
        self.assertEqual(patches[0], patches[1])
        self.assertEqual(
            patches[0]["args"][:5],
            [
                "patch", "configmap/inverse-proxy-config", "--type", "merge",
                "--patch"
            ],
        )
        self.assertEqual(
            json.loads(patches[0]["args"][5]),
            {
                "data": {
                    "Hostname": "public.invalid",
                    "ProxyUrl": "https://registration.invalid",
                    "BackendId": "fresh-backend",
                }
            },
        )
        first, second = calls.index(patches[0]), len(calls) - 2
        self.assertEqual(calls[first + 1]["name"], "sleep")
        self.assertEqual(calls[second]["name"], "kubectl")
        self.assertFalse(any(call["name"] == "wget" for call in calls))

    def test_fresh_registration_discovers_regional_url_with_real_requests(self):
        received = []

        class Proxy(http.server.BaseHTTPRequestHandler):

            def do_HEAD(self):
                received.append(self.path)
                self.send_response(307)
                self.end_headers()

            def log_message(self, *args):
                pass

        server = http.server.HTTPServer(('127.0.0.1', 0), Proxy)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(thread.join, 2)
        self.addCleanup(server.shutdown)
        proxy = 'http://127.0.0.1:' + str(server.server_port)
        self.environment['PROXY_URL'] = ''
        self.environment['EXPECTED_PROXY_URL'] = proxy
        self.environment['NO_PROXY'] = '127.0.0.1'
        shutil.copyfile(
            Path(__file__).with_name('get_proxy_url.py'),
            self.root / 'get_proxy_url.py')
        python = self.bin / 'python3'
        python.write_text('#!/bin/sh\nexec ' + shlex.quote(sys.executable) +
                          ' "$@"\n')
        python.chmod(0o755)

        calls = self.run_registration()

        self.assertEqual(received, ['/'])
        self.assert_agent(calls, proxy, 'fresh-backend')
        self.assertEqual([
            call['args'] for call in calls if call['name'] == 'wget'
        ], [[
            'https://storage.googleapis.com/ml-pipeline/proxy-agent-config.json'
        ]])
        patches = [
            call for call in calls
            if call['name'] == 'kubectl' and call['args'][0] == 'patch'
        ]
        self.assertEqual(len(patches), 2)
        for patch in patches:
            self.assertEqual(
                json.loads(patch['args'][5])['data'], {
                    'Hostname': 'public.invalid',
                    'ProxyUrl': proxy,
                    'BackendId': 'fresh-backend'
                })
        registration = [call for call in calls if call['name'] == 'curl']
        self.assertEqual(registration[-1]['args'][-1],
                         proxy + '/request-service-account-endpoint')
        self.assertIn(
            'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?format=full&audience='
            + proxy + '/request-service-account-endpoint',
            registration[1]['args'])


if __name__ == "__main__":
    unittest.main()
