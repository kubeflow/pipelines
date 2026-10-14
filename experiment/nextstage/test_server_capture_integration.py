import datetime as dt,json,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
from coordinator import capture_server_classification,UTC
class Capture(unittest.TestCase):
 def test_finish_uses_explicit_measurement_window_and_bounded_process(self):
  with tempfile.TemporaryDirectory() as d:
   start=dt.datetime(2026,1,1,tzinfo=UTC);end=start+dt.timedelta(seconds=1230)
   with patch('coordinator.bounded') as command:
    capture_server_classification({'server_evidence_script':'capture.py','server_evidence_config':'private.json'},Path(d),'finish',start,end)
   args=command.call_args.args[0];self.assertIn('--start',args);self.assertIn('--end',args);self.assertEqual(command.call_args.kwargs['timeout'],120)
 def test_collection_failure_preserves_incomplete_without_changing_measurement(self):
  with tempfile.TemporaryDirectory() as d:
   with patch('coordinator.bounded',side_effect=RuntimeError('private detail')):
    capture_server_classification({'server_evidence_script':'capture.py','server_evidence_config':'private.json'},Path(d),'begin')
   result=json.loads((Path(d)/'server-classification/classification.json').read_text());self.assertFalse(result['acceptance_complete']);self.assertFalse(result['actual_kfp_records_proven']);self.assertNotIn('private detail',json.dumps(result))
