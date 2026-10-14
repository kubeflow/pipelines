import tempfile,unittest
from pathlib import Path
from unittest.mock import patch
from integration import scheduled_stage
class Integration(unittest.TestCase):
    def test_incomplete_schedule_prevents_load_start(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d);(p/'before.jsonl').write_text('{"tenant":"one","complete":false}\n')
            with patch('integration.bounded') as command:
                with self.assertRaises(RuntimeError):scheduled_stage({'scheduled_binary':'/candidate'}, {'label':'test','tenants':[{'name':'one'}]},p,'before')
            self.assertEqual(command.call_count,1)
            self.assertTrue((p/'before-result.json').exists())
    def test_success_records_actual_stage(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d);(p/'before.jsonl').write_text('{"tenant":"one","complete":true,"created":true,"disabled":true,"child_succeeded":true}\n')
            with patch('integration.bounded'):scheduled_stage({'scheduled_binary':'/candidate'}, {'label':'test','tenants':[{'name':'one'}]},p,'before')
            self.assertIn('true',(p/'before-result.json').read_text())
if __name__=='__main__':unittest.main()
