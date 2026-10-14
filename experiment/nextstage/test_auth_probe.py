import tempfile,unittest
from pathlib import Path
from unittest.mock import patch
from auth_probe import probe
from coordinator import load
class Auth(unittest.TestCase):
    def setup_input(self,d):
        token=Path(d)/'token';token.write_text('secret')
        tenants=[{'name':n,'namespace':n,'token_file':str(token),'experiment_id':n,'service_account':'editor','fixtures':[{'pipeline_id':'p','version_id':'v'}]} for n in ('a','b')]
        return {'endpoint':'http://127.0.0.1','tenants':tenants}
    def test_exact_denials_owner_absence_pass(self):
        with tempfile.TemporaryDirectory() as d:
            out=Path(d)/'probe.json'
            with patch('auth_probe.list_names',return_value=set()),patch('auth_probe.call',return_value=(403,None)),patch('auth_probe.time.sleep'):
                probe(self.setup_input(d),{'a':'a','b':'b'},out,'during')
            self.assertTrue(all(r['complete'] for r in load(out)))
    def test_accepted_write_preserved_no_retry(self):
        with tempfile.TemporaryDirectory() as d:
            out=Path(d)/'probe.json'
            with patch('auth_probe.list_names',return_value=set()),patch('auth_probe.call',side_effect=[(200,{'run_id':'unexpected'}),(403,None),(403,None)]) as call,patch('auth_probe.time.sleep'):
                with self.assertRaises(RuntimeError):probe(self.setup_input(d),{'a':'a','b':'b'},out,'during')
            rows=load(out);self.assertEqual(rows[0]['unexpected_accepted_run_id'],'unexpected');self.assertFalse(rows[0]['complete']);self.assertEqual(call.call_count,3)
    def test_transport_error_not_denial(self):
        with tempfile.TemporaryDirectory() as d:
            out=Path(d)/'probe.json'
            with patch('auth_probe.list_names',return_value=set()),patch('auth_probe.call',side_effect=OSError):
                with self.assertRaises(OSError):probe(self.setup_input(d),{'a':'a','b':'b'},out,'during')
            self.assertFalse(load(out)[0]['complete'])
if __name__=='__main__':unittest.main()
