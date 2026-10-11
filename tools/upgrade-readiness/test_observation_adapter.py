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
import contextlib
import importlib.util
import io
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

ASSETS = Path(__file__).parent / 'observation'
SPEC = importlib.util.spec_from_file_location('observation_adapter',
                                              ASSETS / 'apply_adapter.py')
adapter = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(adapter)


class ObservationAdapterTest(unittest.TestCase):

    def test_exact_revision_and_clean_root_are_required(self):
        source = Path('/tmp/source')
        for answers, reason in ((['/tmp'], 'checkout root'),
                                ([str(source), 'a' * 40], 'exact pinned'), ([
                                    str(source), adapter.SOURCE_REVISION,
                                    ' M backend.go'
                                ], 'clean source'), ([
                                    str(source), adapter.SOURCE_REVISION,
                                    '?? untracked.go'
                                ], 'clean source')):
            with self.subTest(answers=answers), mock.patch.object(
                    adapter, 'git', side_effect=answers), mock.patch.object(
                        adapter.shutil, 'copyfile') as copy:
                with self.assertRaisesRegex(ValueError, reason):
                    adapter.apply(source, check_only=False)
                copy.assert_not_called()

    def test_default_only_checks_patch_without_writing(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory).resolve()
            with mock.patch.object(
                    adapter,
                    'git',
                    side_effect=[str(source), adapter.SOURCE_REVISION, '',
                                 '']) as git:
                adapter.apply(source)
            self.assertEqual([], list(source.iterdir()))
            self.assertEqual(('apply', '--check'), git.call_args.args[1:3])

    def test_apply_copies_only_recorder_and_uses_validated_patch(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory).resolve()
            (source / adapter.PACKAGE).parent.mkdir(parents=True)
            with mock.patch.object(
                    adapter,
                    'git',
                    side_effect=[
                        str(source), adapter.SOURCE_REVISION, '', '', ''
                    ]) as git:
                adapter.apply(source, check_only=False)
            self.assertEqual(
                {'recorder.go', 'recorder_test.go'},
                {file.name for file in (source / adapter.PACKAGE).iterdir()})
            for name in adapter.SOURCES:
                self.assertEqual((ASSETS / name).read_bytes(),
                                 (source / adapter.PACKAGE / name).read_bytes())
            self.assertEqual(('apply', '--check'),
                             git.call_args_list[-2].args[1:3])
            self.assertEqual('apply', git.call_args.args[1])

    def test_preexisting_package_is_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory).resolve()
            package = source / adapter.PACKAGE
            package.mkdir(parents=True)
            (package / 'custom.go').write_text('preserve')
            with mock.patch.object(
                    adapter,
                    'git',
                    side_effect=[str(source), adapter.SOURCE_REVISION, '']):
                with self.assertRaisesRegex(ValueError,
                                            'must not already exist'):
                    adapter.apply(source, check_only=False)
            self.assertEqual('preserve', (package / 'custom.go').read_text())

    def test_failed_apply_removes_only_its_new_package(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory).resolve()
            (source / adapter.PACKAGE).parent.mkdir(parents=True)
            with mock.patch.object(
                    adapter,
                    'git',
                    side_effect=[
                        str(source), adapter.SOURCE_REVISION, '', '',
                        subprocess.CalledProcessError(1, ['git', 'apply'])
                    ]):
                with self.assertRaises(subprocess.CalledProcessError):
                    adapter.apply(source, check_only=False)
            self.assertFalse((source / adapter.PACKAGE).exists())

    def test_cli_requires_explicit_apply_and_sanitizes_failures(self):
        for extra, check_only in (([], True), (['--apply'], False)):
            with self.subTest(extra=extra), mock.patch.object(
                    sys, 'argv', ['adapter', '--source', '/tmp/backend'] + extra), \
                    mock.patch.object(adapter, 'apply') as apply, \
                    contextlib.redirect_stdout(io.StringIO()):
                adapter.main()
                apply.assert_called_once_with(
                    Path('/tmp/backend'), check_only=check_only)
        error = subprocess.CalledProcessError(
            1, ['private-command'], stderr='private-file-data')
        output = io.StringIO()
        with mock.patch.object(sys, 'argv', ['adapter', '--source', '/tmp/backend']), \
                mock.patch.object(adapter, 'apply', side_effect=error), \
                contextlib.redirect_stderr(output):
            with self.assertRaises(SystemExit) as result:
                adapter.main()
        self.assertEqual(1, result.exception.code)
        self.assertNotIn('private-', output.getvalue())

    def test_patch_preserves_original_enforcement_statements(self):
        patch = (ASSETS / 'source-2.17.2.patch').read_text()
        removed = [
            line[1:]
            for line in patch.splitlines()
            if line.startswith('-') and not line.startswith('---')
        ]
        self.assertEqual(['\tuserIdentity := ""'], removed)
        self.assertIn('readinessobservation.Start(backgroundCtx)', patch)
        self.assertIn('defer readinessobservation.Shutdown()', patch)
        self.assertIn('allowed_with_evaluation_error', patch)
        self.assertNotIn('+\t\treturn ', patch)
        recorder = (ASSETS / 'recorder.go').read_text()
        self.assertIn(adapter.SOURCE_REVISION, recorder)

    def test_git_commands_have_a_deadline(self):
        with mock.patch.object(
                adapter.subprocess, 'check_output',
                return_value='source\n') as command:
            self.assertEqual('source',
                             adapter.git(Path('.'), 'rev-parse', 'HEAD'))
        self.assertEqual(30, command.call_args.kwargs['timeout'])

    def test_workflow_pins_source_and_runs_relevant_boundaries(self):
        workflow = (Path(__file__).resolve().parents[2] /
                    '.github/workflows/upgrade-readiness.yml').read_text()
        observer = workflow.split('  source-observation:', 1)[1]
        self.assertIn('ref: ' + adapter.SOURCE_REVISION, observer)
        self.assertIn('apply_adapter.py --source observation-source --apply',
                      observer)
        self.assertIn(
            'go test -race ./backend/src/apiserver/readinessobservation',
            observer)
        expression = re.search(r"-run '([^']+)'", observer).group(1)
        for test in ('TestUploadPipeline',
                     'TestUploadPipelineVersion_GetFromFileError',
                     'TestReadRunLogV1_MissingRunId',
                     'TestReadArtifactV1_Unauthorized',
                     'TestCreateRun_ThroughWorkflowSpec',
                     'TestCreateJob_Unauthorized'):
            self.assertRegex(test, expression)


if __name__ == '__main__':
    unittest.main()
