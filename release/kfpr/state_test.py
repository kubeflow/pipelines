#!/usr/bin/env python3
"""Tests for release checkpoint state."""

import json
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from typer.testing import CliRunner

from kfpr import core
from kfpr.cli import app
from kfpr.core import ReleaseState

# A checkpoint written by an earlier kfpr release. Its bytes must not change.
SAVED_STATE = """{
  "answers": {
    "release_type": "patch",
    "version": "2.18.1",
    "fork_remote": "git@github.com:someone/pipelines.git",
    "include_backend": true,
    "include_sdk": false,
    "skip_local_review": true,
    "prepared_changelog": {
      "version": "2.18.1",
      "entry": "## 2.18.1"
    }
  },
  "completed_steps": [
    "preflight",
    "prepare-patch-branch"
  ]
}"""

SHAPE_ERRORS = {
    'not_object': ('[1, 2, 3]', ['state file must be an object']),
    'answers_not_object': (
        '{"answers": [], "completed_steps": []}',
        ['answers must be an object'],
    ),
    'completed_steps_not_list': (
        '{"answers": {"release_type": "patch", "version": "2.18.1"}, '
        '"completed_steps": "preflight"}',
        ['completed_steps must be a list'],
    ),
    'completed_step_not_string': (
        '{"answers": {"release_type": "patch", "version": "2.18.1"}, '
        '"completed_steps": [123]}',
        ['completed_steps must be a list of strings'],
    ),
    'both_invalid': (
        '{"answers": [], "completed_steps": [123]}',
        [
            'answers must be an object',
            'completed_steps must be a list of strings',
        ],
    ),
}


class StateCompatibilityTest(unittest.TestCase):
    """Pins checkpoint behaviour that existing release state files rely on."""

    def test_saved_state_round_trips_byte_for_byte(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(SAVED_STATE)

            ReleaseState.load(path).save()

            self.assertEqual(path.read_text(), SAVED_STATE)

    def test_non_string_answers_survive_round_trip(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(SAVED_STATE)

            state = ReleaseState.load(path)
            state.save()
            reloaded = ReleaseState.load(path)

            self.assertIs(reloaded.answers['include_backend'], True)
            self.assertIs(reloaded.answers['include_sdk'], False)
            self.assertEqual(
                reloaded.answers['prepared_changelog'],
                {'version': '2.18.1', 'entry': '## 2.18.1'},
            )
            self.assertEqual(reloaded, state)

    def test_extra_top_level_keys_are_ignored(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(
                '{"answers": {"version": "2.18.1"}, '
                '"completed_steps": ["preflight"], "extra": 1}'
            )

            state = ReleaseState.load(path)
            state.save()

            self.assertEqual(state.answers, {'version': '2.18.1'})
            self.assertEqual(state.completed_steps, ['preflight'])
            self.assertEqual(
                json.loads(path.read_text()),
                {
                    'answers': {'version': '2.18.1'},
                    'completed_steps': ['preflight'],
                },
            )

    def test_malformed_json_raises_value_error(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text('{"answers": ')

            with self.assertRaisesRegex(
                ValueError,
                r'Invalid state file: .* contains malformed JSON\. '
                r'Please fix or delete the file\. Details: ',
            ):
                ReleaseState.load(path)

    def test_validate_state_reports_shape_errors(self):
        for name, (content, messages) in SHAPE_ERRORS.items():
            with self.subTest(name), TemporaryDirectory() as tmpdir:
                path = Path(tmpdir) / 'state.json'
                path.write_text(content)

                errors = core.validate_state(ReleaseState.load(path))

                self.assertEqual(errors, messages)

    def test_validate_state_command_reports_shape_errors(self):
        for name, (content, messages) in SHAPE_ERRORS.items():
            with self.subTest(name), TemporaryDirectory() as tmpdir:
                path = Path(tmpdir) / 'state.json'
                path.write_text(content)

                result = CliRunner().invoke(
                    app, ['validate-state', '--state-file', str(path)]
                )

                self.assertEqual(result.exit_code, 1, result.output)
                self.assertEqual(result.output.splitlines(), messages)
                self.assertEqual(path.read_text(), content)

    def test_status_reports_shape_errors_without_changing_file(self):
        for name, (content, messages) in SHAPE_ERRORS.items():
            with self.subTest(name), TemporaryDirectory() as tmpdir:
                path = Path(tmpdir) / 'state.json'
                path.write_text(content)

                result = CliRunner().invoke(
                    app, ['status', '--state-file', str(path)]
                )

                self.assertEqual(result.exit_code, 0, result.output)
                self.assertIn(messages[0], result.output)
                self.assertEqual(path.read_text(), content)

    def test_status_keeps_valid_answers_when_completed_steps_are_invalid(self):
        content, messages = SHAPE_ERRORS['completed_step_not_string']
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(content)

            result = CliRunner().invoke(
                app, ['status', '--state-file', str(path)]
            )

            self.assertIn('  release_type: patch\n', result.output)
            self.assertIn('  version: 2.18.1\n', result.output)
            self.assertIn(f'errors:\n  {messages[0]}\n', result.output)


class LoadErrorsTest(unittest.TestCase):
    def test_shape_errors_are_recorded_without_placeholder_values(self):
        expected = {
            'not_object': ({}, []),
            'answers_not_object': ({}, []),
            'completed_steps_not_list': (
                {'release_type': 'patch', 'version': '2.18.1'},
                [],
            ),
            'completed_step_not_string': (
                {'release_type': 'patch', 'version': '2.18.1'},
                [123],
            ),
            'both_invalid': ({}, [123]),
        }
        for name, (answers, completed_steps) in expected.items():
            content, messages = SHAPE_ERRORS[name]
            with self.subTest(name), TemporaryDirectory() as tmpdir:
                path = Path(tmpdir) / 'state.json'
                path.write_text(content)

                state = ReleaseState.load(path)

                self.assertEqual(state.answers, answers)
                self.assertEqual(state.completed_steps, completed_steps)
                self.assertEqual(state.load_errors, messages)

    def test_valid_state_has_no_load_errors(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(SAVED_STATE)

            self.assertEqual(ReleaseState.load(path).load_errors, [])

    def test_load_errors_are_not_part_of_state_identity(self):
        state = ReleaseState(Path('state.json'), load_errors=['problem'])

        self.assertEqual(state, ReleaseState(Path('state.json')))
        self.assertNotIn('load_errors', repr(state))

    def test_status_reports_load_errors_instead_of_missing_answers(self):
        content, messages = SHAPE_ERRORS['not_object']
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(content)

            result = CliRunner().invoke(
                app, ['status', '--state-file', str(path)]
            )

            self.assertEqual(
                result.output,
                f'answers:\n  (none)\nerrors:\n  {messages[0]}\n',
            )


class InvalidStateUpdateTest(unittest.TestCase):
    """Commands that change a checkpoint must not overwrite an invalid one."""

    INVALID = tuple(SHAPE_ERRORS)
    COMMANDS = {
        'done': ['done', 'preflight'],
        'reset-step': ['reset-step', 'preflight'],
        'goto': ['goto', 'preflight'],
        'next': ['next'],
        'run': ['run'],
        'run --force': ['run', '--force'],
        'run preflight': ['run', 'preflight'],
    }

    def test_commands_refuse_invalid_state_before_prompting(self):
        for name in self.INVALID:
            content, messages = SHAPE_ERRORS[name]
            for label, args in self.COMMANDS.items():
                with (
                    self.subTest(state=name, command=label),
                    TemporaryDirectory() as tmpdir,
                ):
                    path = Path(tmpdir) / 'state.json'
                    path.write_text(content)

                    result = CliRunner().invoke(
                        app, [*args, '--state-file', str(path)], input=''
                    )

                    self.assertEqual(result.exit_code, 1, result.output)
                    self.assertEqual(
                        result.output.splitlines(),
                        [
                            *messages,
                            f'Fix {path} or delete it with '
                            f'`kfpr clear --state-file {path}` before '
                            'continuing.',
                        ],
                    )
                    self.assertEqual(path.read_text(), content)

    def test_clear_still_removes_invalid_state(self):
        content, _ = SHAPE_ERRORS['not_object']
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(content)

            result = CliRunner().invoke(
                app, ['clear', '--state-file', str(path)]
            )

            self.assertEqual(result.exit_code, 0, result.output)
            self.assertFalse(path.exists())

    def test_save_refuses_state_loaded_from_invalid_file(self):
        content, messages = SHAPE_ERRORS['answers_not_object']
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(content)
            state = ReleaseState.load(path)
            state.mark_done('preflight')

            with self.assertRaisesRegex(
                ValueError,
                f'Refusing to overwrite invalid state file .*: {messages[0]}',
            ):
                state.save()

            self.assertEqual(path.read_text(), content)
            self.assertFalse(path.with_suffix('.json.tmp').exists())

    def test_failed_write_removes_temporary_file(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'state.json'
            path.write_text(SAVED_STATE)
            state = ReleaseState.load(path)

            with (
                mock.patch.object(json, 'dump', side_effect=OSError('full')),
                self.assertRaises(OSError),
            ):
                state.save()

            self.assertEqual(path.read_text(), SAVED_STATE)
            self.assertFalse(path.with_suffix('.json.tmp').exists())


class TestReleaseState(unittest.TestCase):
    def test_load_nonexistent_path_returns_fresh_state(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'nonexistent' / 'state.json'
            loaded_state = ReleaseState.load(state_file)

            self.assertEqual(loaded_state.path, state_file)
            self.assertEqual(loaded_state.answers, {})
            self.assertEqual(loaded_state.completed_steps, [])

    def test_load_malformed_json_raises_value_error(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state_file.write_text('{ invalid json }')

            with self.assertRaises(ValueError) as cm:
                ReleaseState.load(state_file)

            error_msg = str(cm.exception)
            self.assertIn('Invalid state file', error_msg)
            self.assertIn(str(state_file), error_msg)
            self.assertIn('malformed JSON', error_msg)
            self.assertIn('fix or delete', error_msg)

    def test_load_save_roundtrip(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers['version'] = '1.0.0'
            state.save()

            loaded_state = ReleaseState.load(state_file)
            self.assertEqual(loaded_state.answers['version'], '1.0.0')

    def test_save_uses_tmp_file(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers['version'] = '1.0.0'
            state.save()

            self.assertTrue(state_file.exists())
            tmp_file = state_file.with_suffix(state_file.suffix + '.tmp')
            self.assertFalse(tmp_file.exists())

    def test_mark_done_and_is_done(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)

            self.assertFalse(state.is_done('step1'))
            state.mark_done('step1')
            self.assertTrue(state.is_done('step1'))
            state.save()

            loaded_state = ReleaseState.load(state_file)
            self.assertTrue(loaded_state.is_done('step1'))

    def test_multiple_steps(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)

            state.mark_done('step1')
            state.mark_done('step2')
            state.save()

            loaded_state = ReleaseState.load(state_file)
            self.assertTrue(loaded_state.is_done('step1'))
            self.assertTrue(loaded_state.is_done('step2'))
            self.assertFalse(loaded_state.is_done('step3'))


if __name__ == '__main__':
    unittest.main()
