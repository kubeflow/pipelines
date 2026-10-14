import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import download_images as d


class DownloadTests(unittest.TestCase):
    def test_member_paths_are_fixed_tar_names(self):
        for value in ('../apiserver.tar', '/apiserver.tar', 'sub/apiserver.tar', 'apiserver.zip', None):
            self.assertFalse(d.safe_member(value))
        self.assertTrue(d.safe_member('viewer-crd-controller.tar'))

    def test_rejects_wrong_source_before_network_or_directory_creation(self):
        with tempfile.TemporaryDirectory() as temp:
            manifest = Path(temp) / 'manifest.json'
            manifest.write_text(json.dumps({'repository': 'other/project'}))
            with patch.object(d, 'fetch') as fetch:
                with self.assertRaises(RuntimeError):
                    d.download(manifest, Path(temp) / 'output')
                fetch.assert_not_called()
                self.assertFalse((Path(temp) / 'output').exists())

    def test_gh_receives_no_token_argument_and_failure_is_sanitized(self):
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(d.subprocess, 'run', return_value=subprocess.CompletedProcess([], 1, stderr=b'gh: Not Found (HTTP 404) SECRET')) as run:
                with self.assertRaisesRegex(RuntimeError, r'download failed: image.zip \(HTTP 404\)' ):
                    d.fetch('repos/kubeflow/pipelines/actions/artifacts/123/zip', Path(temp) / 'image.zip')
                self.assertEqual(run.call_args.args[0], ['gh', 'api', '--allow-escape-sequences', 'repos/kubeflow/pipelines/actions/artifacts/123/zip'])
                self.assertEqual(run.call_args.kwargs['stderr'], subprocess.PIPE)
                self.assertFalse((Path(temp) / 'image.zip').exists())

if __name__ == '__main__':
    unittest.main()
