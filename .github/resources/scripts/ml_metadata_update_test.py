"""Verify the retained MLMD updater targets protobuf generation only."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]


class MetadataUpdateTest(unittest.TestCase):

    def test_updater_calls_generation_and_propagates_failure(self):
        for status in (0, 17):
            with self.subTest(
                    status=status), tempfile.TemporaryDirectory() as directory:
                work = Path(directory)
                make = work / 'make'
                make.write_text(
                    '#!/bin/sh\nprintf "%s\\n" "$@" > "$OUTPUT"\nexit "$MAKE_EXIT"\n'
                )
                make.chmod(0o755)
                output = work / 'args'
                env = dict(
                    os.environ,
                    PATH=f'{work}:{os.environ["PATH"]}',
                    OUTPUT=str(output),
                    MAKE_EXIT=str(status))
                result = subprocess.run([
                    'bash',
                    str(ROOT / 'third_party/ml-metadata/update_version.sh')
                ],
                                        cwd=work,
                                        env=env,
                                        capture_output=True,
                                        text=True,
                                        check=False)
                self.assertEqual(result.returncode, status, result.stderr)
                self.assertEqual(
                    output.read_text().splitlines(),
                    ['-C',
                     str(ROOT / 'third_party/ml-metadata'), 'update'])


if __name__ == '__main__':
    unittest.main()
