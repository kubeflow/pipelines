"""Exercise the ordinary-image provenance boundary with real Git fixtures."""

import importlib.util
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'image_build_provenance',
    Path(__file__).with_name('image-build-provenance.py'),
)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ImageBuildProvenanceTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.source = self.root / 'source'
        self.source.mkdir()
        self.git('init', '--quiet')
        self.git('config', 'user.name', 'Fixture')
        self.git('config', 'user.email', 'fixture@example.test')
        (self.source / 'backend').mkdir()
        (self.source / 'backend/Dockerfile').write_text(
            'FROM scratch\nCOPY . .\n'
        )
        (self.source / '.dockerignore').write_text('.git\n')
        self.git('add', '.')
        self.git('commit', '--quiet', '-m', 'fixture')
        self.sha = self.git('rev-parse', 'HEAD')

    def git(self, *args: str) -> str:
        return subprocess.check_output(
            ['git', '-c', 'core.fsmonitor=false', *args],
            cwd=self.source,
            text=True,
        ).strip()

    def test_go_context_has_exact_clean_git_without_checkout_credentials(
        self,
    ) -> None:
        self.git('config', 'http.extraheader', 'authorization: fixture-secret')
        result = MODULE.prepare(
            self.source, self.sha, 'backend/Dockerfile', self.root / 'prepared'
        )
        context = Path(result['context'])
        self.assertTrue((context / '.git').is_dir())
        self.assertNotIn(
            'fixture-secret', (context / '.git/config').read_text()
        )
        self.assertEqual((self.source / '.dockerignore').read_text(), '.git\n')
        self.assertEqual(
            Path(str(result['dockerfile']) + '.dockerignore').read_text(), ''
        )
        self.assertEqual(
            MODULE.provenance._git(['rev-parse', 'HEAD'], context), self.sha
        )
        self.assertEqual(
            MODULE.provenance._git(['status', '--porcelain'], context), ''
        )

    def test_dirty_source_rejected(self) -> None:
        (self.source / 'backend/Dockerfile').write_text('FROM changed\n')
        with self.assertRaises(ValueError):
            MODULE.prepare(
                self.source,
                self.sha,
                'backend/Dockerfile',
                self.root / 'prepared',
            )

    def test_non_go_context_retains_normal_ignore_rules(self) -> None:
        for context, dockerfile in (
            ('.', 'frontend/Dockerfile'),
            ('backend/tensorboard', 'backend/tensorboard/Dockerfile'),
        ):
            with self.subTest(context=context):
                recipe = self.source / dockerfile
                recipe.parent.mkdir(parents=True, exist_ok=True)
                recipe.write_text('FROM scratch\nCOPY . .\n')
                ignores = recipe.with_name(recipe.name + '.dockerignore')
                ignores.write_text('ignored-secret\n')
                self.git('add', '.')
                self.git('commit', '--quiet', '-m', 'non-Go fixture')
                sha = self.git('rev-parse', 'HEAD')
                tooling = (
                    self.source
                    / '.publication-tooling/.github/actions/setup-go/action.yml'
                )
                tooling.parent.mkdir(parents=True, exist_ok=True)
                tooling.write_text('local action must survive until cleanup')
                result = MODULE.prepare(
                    self.source,
                    sha,
                    dockerfile,
                    self.root / ('prepared-' + recipe.parent.name),
                    context,
                )
                prepared = Path(result['context'])
                self.assertEqual(
                    prepared,
                    self.root
                    / ('prepared-' + recipe.parent.name)
                    / 'source'
                    / context,
                )
                self.assertNotEqual(prepared, self.source / context)
                self.assertFalse((prepared / '.publication-tooling').exists())
                self.assertTrue(tooling.is_file())
                self.assertEqual(
                    Path(
                        str(result['dockerfile']) + '.dockerignore'
                    ).read_text(),
                    'ignored-secret\n',
                )
                self.assertEqual(
                    Path(result['dockerfile']).read_text(), recipe.read_text()
                )
                # Do not accidentally commit the tooling fixture in the next case.
                shutil.rmtree(self.source / '.publication-tooling')

    def test_relative_dot_prefix_cannot_skip_go_validation(self) -> None:
        with mock.patch.object(
            MODULE.provenance, 'prepare_context', return_value={}
        ) as prepare:
            MODULE.prepare(
                self.source,
                self.sha,
                './backend/Dockerfile',
                self.root / 'prepared',
            )
        prepare.assert_called_once()

    def test_recipe_paths_rejected_before_classification(self) -> None:
        for recipe in (
            'backend/../backend/Dockerfile',
            str(self.source / 'backend/Dockerfile'),
        ):
            for operation in ('prepare', 'verify'):
                with self.subTest(recipe=recipe, operation=operation):
                    with self.assertRaisesRegex(
                        ValueError, 'relative.*traversal'
                    ):
                        if operation == 'prepare':
                            MODULE.prepare(
                                self.source,
                                self.sha,
                                recipe,
                                self.root / 'prepared',
                            )
                        else:
                            MODULE.verify(
                                self.source,
                                self.sha,
                                recipe,
                                'local:image',
                                'linux/amd64',
                                self.root / 'evidence',
                            )

    def test_go_context_rejects_subdirectory(self) -> None:
        with self.assertRaises(ValueError):
            MODULE.prepare(
                self.source,
                self.sha,
                'backend/Dockerfile',
                self.root / 'prepared',
                'backend',
            )

    def test_verifier_binds_both_architectures_to_resolved_image(self) -> None:
        image_id = 'sha256:' + 'a' * 64
        for platform in ('linux/amd64', 'linux/arm64'):
            with (
                self.subTest(platform=platform),
                mock.patch.object(
                    MODULE.subprocess,
                    'check_output',
                    return_value=image_id + '\n',
                ),
                mock.patch.object(
                    MODULE.provenance,
                    'source_identity',
                    return_value={'source_time': '2026-01-01T00:00:00Z'},
                ),
                mock.patch.object(
                    MODULE.provenance, 'collect', return_value={}
                ) as collect,
            ):
                MODULE.verify(
                    self.source,
                    self.sha,
                    'backend/Dockerfile',
                    'local:image',
                    platform,
                    self.root / 'evidence',
                )
                collect.assert_called_once_with(
                    image_id,
                    platform,
                    self.sha,
                    '2026-01-01T00:00:00Z',
                    ['/bin/apiserver'],
                    self.root / 'evidence',
                )

    def test_verification_failure_propagates(self) -> None:
        with (
            mock.patch.object(
                MODULE.subprocess,
                'check_output',
                return_value='sha256:' + 'a' * 64,
            ),
            mock.patch.object(
                MODULE.provenance,
                'source_identity',
                return_value={'source_time': '2026-01-01T00:00:00Z'},
            ),
            mock.patch.object(
                MODULE.provenance,
                'collect',
                side_effect=ValueError('wrong source'),
            ),
        ):
            with self.assertRaisesRegex(ValueError, 'wrong source'):
                MODULE.verify(
                    self.source,
                    self.sha,
                    'backend/Dockerfile',
                    'local:image',
                    'linux/amd64',
                    self.root / 'evidence',
                )

    def test_non_go_verification_records_explicit_skip(self) -> None:
        output = self.root / 'evidence'
        with mock.patch.object(MODULE.provenance, 'collect') as collect:
            MODULE.verify(
                self.source,
                self.sha,
                'frontend/Dockerfile',
                'local:image',
                'linux/amd64',
                output,
            )
        collect.assert_not_called()
        self.assertEqual(
            json.loads((output / 'go-provenance.json').read_text())['status'],
            'not-a-kfp-go-image',
        )


if __name__ == '__main__':
    unittest.main()
