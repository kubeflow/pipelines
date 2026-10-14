import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from prepare_pair import bind_runtime_image,calibration_request
from unittest.mock import patch
import urllib.request
from coordinator import NoRedirect
from coordinator import incomplete_acceptance

class RuntimeIdentityTests(unittest.TestCase):
    def test_calibration_blocks_bearer_redirects_and_preserves_ca(self):
        ca=object(); request=urllib.request.Request('https://127.0.0.1:6443/api/v1/namespaces/probe/configmaps/probe',headers={'Authorization':'Bearer secret'})
        with patch('prepare_pair.urllib.request.build_opener') as build:
            calibration_request(request,ca)
            handlers=build.call_args.args
            self.assertTrue(any(isinstance(h,NoRedirect) for h in handlers))
            tls=next(h for h in handlers if isinstance(h,urllib.request.HTTPSHandler))
            self.assertIs(tls._context,ca)
            build.return_value.open.assert_called_once_with(request,timeout=10)
        for status in (301,302,303,307,308):
            with self.assertRaises(urllib.error.HTTPError) as error:
                NoRedirect().redirect_request(request,None,status,'redirect',{},'https://other.invalid/')
            error.exception.close()

    def test_containerd_manifest_and_index_bind_pinned_config(self):
        expected='sha256:'+'a'*64
        raw=json.dumps({'config':{'digest':expected}}).encode()
        digest='sha256:'+hashlib.sha256(raw).hexdigest()
        index=json.dumps({'manifests':[{'digest':digest,'platform':{'os':'linux','architecture':'amd64'}}]}).encode()
        indexdigest='sha256:'+hashlib.sha256(index).hexdigest()
        blobs={digest:raw,indexdigest:index}
        for runtime in (digest,'docker-pullable://repo/image@'+digest,indexdigest):
            proof=bind_runtime_image(runtime,expected,blobs.__getitem__)
            self.assertEqual(proof['config_digest'],expected)
            self.assertTrue(proof['manifest_chain'])

    def test_wrong_config_and_forged_content_rejected(self):
        raw=json.dumps({'config':{'digest':'sha256:'+'b'*64}}).encode()
        digest='sha256:'+hashlib.sha256(raw).hexdigest()
        with self.assertRaises(RuntimeError):
            bind_runtime_image(digest,'sha256:'+'a'*64,lambda _:raw)
        with self.assertRaisesRegex(RuntimeError,'manifest bytes'):
            bind_runtime_image('sha256:'+'c'*64,'sha256:'+'a'*64,lambda _:raw)

    def test_direct_config_id_needs_no_manifest_guess(self):
        expected='sha256:'+'a'*64
        proof=bind_runtime_image(expected,expected,lambda _:self.fail('unexpected read'))
        self.assertEqual(proof['manifest_chain'],[])

    def test_measurements_keep_failed_semantics_and_default_gate_false(self):
        with tempfile.TemporaryDirectory() as tmp:
            arm=Path(tmp)/'arm-1';(arm/'semantic').mkdir(parents=True)
            (arm/'semantic'/'summary.json').write_text('{"passed":false}')
            incomplete_acceptance(arm,{'measurement_only':True})
            result=json.loads((arm/'acceptance.json').read_text())
            self.assertFalse(result['acceptance_complete']);self.assertFalse(result['default_change_supported'])
            self.assertFalse(result['semantic_result'])
            self.assertIn('fixture_semantic_assertions_failed',result['missing_gates'])
            self.assertFalse(json.loads((Path(tmp)/'acceptance.json').read_text())['default_change_supported'])

if __name__=='__main__':unittest.main()
