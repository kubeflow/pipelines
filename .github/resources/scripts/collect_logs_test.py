#!/usr/bin/env python3

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
"""Exercise agent container selection through collect-logs.sh, without a cluster."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).with_name('collect-logs.sh')


class CollectLogsTest(unittest.TestCase):

    def test_kubectl_failures_are_bounded_and_best_effort(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            call_log = temp_path / 'calls.txt'
            fake_kubectl = temp_path / 'kubectl'
            fake_kubectl.write_text(
                textwrap.dedent(f'''\
                    #!/usr/bin/env bash
                    echo "$*" >> "{call_log}"
                    case "$*" in
                      *"get namespace kubeflow"*) exit 0 ;;
                      *"get pods -n kubeflow -o custom-columns=:metadata.name --no-headers"*)
                        echo seaweedfs-0
                        exit 0
                        ;;
                      *"get pods -n kubeflow"*)
                        echo seaweedfs-0 Running
                        exit 0
                        ;;
                      *"describe pod seaweedfs-0"*) exit 1 ;;
                      *"logs seaweedfs-0"*) exit 1 ;;
                    esac
                    exit 1
                '''),
                encoding='utf-8',
            )
            fake_kubectl.chmod(0o755)
            output_file = temp_path / 'pod-logs.txt'
            environment = os.environ.copy()
            environment['PATH'] = f'{temp_path}:{environment["PATH"]}'

            result = subprocess.run(
                [str(SCRIPT), '--ns', 'kubeflow', '--output',
                 str(output_file)],
                check=False,
                capture_output=True,
                env=environment,
                text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            output = output_file.read_text(encoding='utf-8')
            self.assertIn('No events found for pod seaweedfs-0.', output)
            self.assertIn('No logs found for pod seaweedfs-0.', output)
            calls = call_log.read_text(encoding='utf-8').splitlines()
            self.assertGreaterEqual(len(calls), 5)
            self.assertTrue(
                all(
                    call.startswith('--request-timeout=20s ')
                    for call in calls),
                calls,
            )

    def test_component_label_not_agent_name_substring_selects_driver(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            calls = root / 'calls.jsonl'
            stub = root / 'kubectl'
            stub.write_text('''#!/usr/bin/env python3
import json
import os
import sys
args = sys.argv[1:]
with open(os.environ['KUBECTL_CALLS'], 'a') as output:
    output.write(json.dumps(args) + '\\n')
assert args.pop(0).startswith('--request-timeout=')
if args[:2] == ['get', 'namespace']:
    pass
elif args[:2] == ['get', 'pods']:
    print('my-agent-pipeline-task\\nopaque-agent-pod')
elif args[:2] == ['get', 'pod']:
    assert args[-1] == r'jsonpath={.metadata.labels.workflows\\.argoproj\\.io/component}'
    print('agent' if args[2] == 'opaque-agent-pod' else 'pod')
elif args[:2] == ['describe', 'pod']:
    print('Events: none')
elif args[0] == 'logs':
    if args[1] == 'my-agent-pipeline-task':
        assert '-c' not in args
        print('workload-log')
    else:
        assert args[-2:] == ['-c', 'driver-plugin']
        print('driver-log')
else:
    raise AssertionError(args)
''')
            stub.chmod(0o755)
            output = root / 'logs.txt'
            subprocess.run(
                ['bash',
                 str(SCRIPT), '--ns', 'test', '--output',
                 str(output)],
                env={
                    **os.environ, 'PATH':
                        str(root) + os.pathsep + os.environ['PATH'],
                    'KUBECTL_CALLS':
                        str(calls)
                },
                check=True,
                capture_output=True,
                text=True)
            logs = output.read_text()
            self.assertIn('workload-log', logs)
            self.assertIn('driver-log', logs)
            self.assertNotIn('No logs found', logs)
            requests = [
                json.loads(line) for line in calls.read_text().splitlines()
            ]
            labels = [args for args in requests if args[1:3] == ['get', 'pod']]
            self.assertEqual(len(labels), 2)


if __name__ == '__main__':
    unittest.main()
