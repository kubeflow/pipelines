"""Publication must use the same sanitized source as candidate
qualification."""

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import yaml

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'release'))
import prepare_publication_context as publication


class PublicationContextTest(unittest.TestCase):
    def test_each_go_image_uses_sanitized_context_and_exact_sha(self) -> None:
        for image, dockerfile in publication.GO_IMAGES.items():
            with (
                self.subTest(image=image),
                tempfile.TemporaryDirectory() as temporary,
            ):
                destination = Path(temporary) / 'prepared'
                with patch.object(publication, 'prepare_context') as prepare:
                    prepare.return_value = {
                        'context': destination / 'source',
                        'dockerfile': destination / 'build.Dockerfile',
                    }
                    result = publication.prepare(
                        image, '.', dockerfile, 'a' * 40, destination
                    )
                prepare.assert_called_once_with(
                    Path.cwd(), 'a' * 40, dockerfile, destination
                )
                self.assertEqual(result['context'], str(destination / 'source'))
                self.assertEqual(
                    result['dockerfile'], str(destination / 'build.Dockerfile')
                )
                self.assertEqual(result['source_sha'], 'a' * 40)

    def test_wrong_go_context_or_recipe_is_rejected(self) -> None:
        for context, dockerfile in [
            ('backend', 'backend/Dockerfile'),
            ('.', 'frontend/Dockerfile'),
        ]:
            with (
                self.subTest(context=context, dockerfile=dockerfile),
                self.assertRaises(ValueError),
            ):
                publication.prepare(
                    'kfp-api-server',
                    context,
                    dockerfile,
                    'a' * 40,
                    Path('/unused'),
                )
        with self.assertRaises(ValueError):
            publication.prepare(
                'unknown-image',
                '.',
                'backend/Dockerfile',
                'a' * 40,
                Path('/unused'),
            )

    def test_unregistered_image_rejects_equivalent_go_recipe_paths(
        self,
    ) -> None:
        for dockerfile in publication.GO_IMAGES.values():
            for alias in (
                './' + dockerfile,
                'backend/../' + dockerfile,
                str(Path(dockerfile).resolve()),
            ):
                with (
                    self.subTest(alias=alias),
                    self.assertRaisesRegex(
                        ValueError, 'registered publication image'
                    ),
                ):
                    publication.prepare(
                        'custom-api', '.', alias, 'a' * 40, Path('/unused')
                    )
        with tempfile.TemporaryDirectory() as temporary:
            alias = Path(temporary) / 'Dockerfile'
            alias.symlink_to(Path('backend/Dockerfile').resolve())
            with self.assertRaisesRegex(
                ValueError, 'registered publication image'
            ):
                publication.prepare(
                    'custom-api', '.', str(alias), 'a' * 40, Path('/unused')
                )

    def test_non_go_context_is_unchanged(self) -> None:
        for image, context, dockerfile in [
            ('frontend', '.', 'frontend/Dockerfile'),
            ('kfp-profile-controller', 'profile', 'profile/Dockerfile'),
        ]:
            with (
                self.subTest(image=image),
                patch.object(publication, 'prepare_context') as prepare,
            ):
                result = publication.prepare(
                    image, context, dockerfile, 'a' * 40, Path('/unused')
                )
                prepare.assert_not_called()
                self.assertEqual(
                    result,
                    {
                        'context': context,
                        'dockerfile': dockerfile,
                        'source_sha': '',
                    },
                )

    def test_old_source_fails_with_release_workflow_guidance(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/build-and-push.yml').read_text()
        )
        step = next(
            step
            for step in workflow['jobs']['dry-run-images']['steps']
            if step.get('id') == 'source_context'
        )
        with tempfile.TemporaryDirectory() as directory:
            result = subprocess.run(
                ['bash', '-e', '-c', step['run']],
                cwd=directory,
                env=os.environ.copy(),
                capture_output=True,
                text=True,
            )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Use its release-branch workflow', result.stdout)
        self.assertNotIn('No such file', result.stderr)

    def test_push_and_dry_run_use_prepared_context_before_export(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/build-and-push.yml').read_text()
        )
        for job in workflow['jobs'].values():
            steps = job['steps']
            prepare = next(
                step for step in steps if step.get('id') == 'source_context'
            )
            build = next(step for step in steps if step.get('id') == 'push')
            self.assertLess(steps.index(prepare), steps.index(build))
            self.assertIn('prepare_publication_context.py', prepare['run'])
            self.assertEqual(
                prepare['env']['SOURCE_SHA'],
                '${{ steps.get_commit.outputs.sha }}',
            )
            for env, output in [
                ('INPUT_IMAGE_CONTEXT', 'context'),
                ('INPUT_DOCKER_FILE', 'dockerfile'),
                ('KFP_SOURCE_SHA', 'source_sha'),
            ]:
                self.assertEqual(
                    build['env'][env],
                    '${{ steps.source_context.outputs.' + output + ' }}',
                )
            self.assertIn(
                '--build-arg "KFP_SOURCE_SHA=${KFP_SOURCE_SHA}"', build['run']
            )


if __name__ == '__main__':
    unittest.main()
