import json,tempfile,unittest
from pathlib import Path
from unittest.mock import patch,MagicMock
import smoke
from coordinator import capture_manifest,load
class SmokeTests(unittest.TestCase):
    def test_full_view_required_for_task_output_evidence(self):
        with patch.object(smoke,'request',return_value={'tasks':[{'type':'RUNTIME'}]}) as request:
            self.assertTrue(smoke.full_run('http://127.0.0.1','r','private')['tasks'])
            request.assert_called_once_with('http://127.0.0.1','/apis/v2beta1/runs/r?view=FULL','private')

    def test_post_once_and_bounds(self):
        response=MagicMock();response.read.return_value=b'{"run_id":"safe"}'
        response.__enter__.return_value=response
        opener=MagicMock();opener.open.return_value=response
        with patch.object(smoke.urllib.request,'build_opener',return_value=opener):
            self.assertEqual(smoke.request('http://127.0.0.1:18888','/runs','private',{'name':'x'}),{'run_id':'safe'})
        self.assertEqual(opener.open.call_count,1)
        req=opener.open.call_args.args[0]
        self.assertEqual(req.get_method(),'POST');self.assertEqual(opener.open.call_args.kwargs['timeout'],10)
    def test_no_retry_failure(self):
        opener=MagicMock();opener.open.side_effect=OSError('secret')
        with patch.object(smoke.urllib.request,'build_opener',return_value=opener):
            with self.assertRaises(OSError):smoke.request('http://127.0.0.1','/runs','private',{})
        self.assertEqual(opener.open.call_count,1)
    def test_redirect_refused(self):
        with self.assertRaises(RuntimeError):smoke.NoRedirect().redirect_request(None,None,302,None,None,None)
    def test_partial_capture_and_symlink_guard(self):
        with tempfile.TemporaryDirectory() as temp:
            p=Path(temp);(p/'created.json').write_text('{"run_id":"accepted"}')
            capture_manifest(p);self.assertEqual(len(load(p/'evidence-captured.json')['files']),1)
            (p/'leak').symlink_to('/etc/passwd')
            with self.assertRaises(RuntimeError):capture_manifest(p)
if __name__=='__main__':unittest.main()
