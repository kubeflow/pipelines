import datetime as dt,json,tempfile,unittest
from pathlib import Path
from coordinator import completed_measurement_capture,digest,iso,save

class CompletedCapture(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.p=Path(self.tmp.name)
  (self.p/'workload').mkdir();(self.p/'observer').mkdir()
  self.end=dt.datetime(2026,10,14,tzinfo=dt.timezone.utc)
  self.work={'phases':[{'seconds':1}],'probe_seconds':1,'drain_seconds':1,'tenants':[{'name':'quiet'},{'name':'noisy'}]}
  self.timeline=[{'name':'after_drain','end':iso(self.end)}]
  save(self.p/'workload/timeline.json',self.timeline)
  save(self.p/'workload/results.json',{'timeline':self.timeline,'all_recorded_runs_succeeded':False})
  save(self.p/'observer/summary.json',{'snapshot_count':5,'timeline_sha256':digest(self.p/'workload/timeline.json'),'observation_complete':False})
  (self.p/'workload/events.jsonl').write_text(''.join(json.dumps({'operation':'phase_end','phase':'after_drain','tenant':t['name']})+'\n' for t in self.work['tenants']))
 def ready(self,now=None):return completed_measurement_capture(self.p,self.work,self.end,now or self.end)
 def test_complete_failure_keeps_diagnostics_allowed_not_acceptance(self):
  self.assertTrue(self.ready());self.assertFalse(json.loads((self.p/'observer/summary.json').read_text())['observation_complete'])
 def test_running_timeline_still_failfast(self):self.assertFalse(self.ready(self.end-dt.timedelta(microseconds=1)))
 def test_missing_result_rejected(self):
  (self.p/'workload/results.json').unlink();self.assertFalse(self.ready())
 def test_partial_snapshots_rejected(self):
  save(self.p/'observer/summary.json',{'snapshot_count':4,'timeline_sha256':digest(self.p/'workload/timeline.json')});self.assertFalse(self.ready())
 def test_missing_phase_end_rejected(self):
  (self.p/'workload/events.jsonl').write_text('');self.assertFalse(self.ready())
 def test_mismatching_timeline_rejected(self):
  save(self.p/'workload/results.json',{'timeline':[]});self.assertFalse(self.ready())
 def test_workflow_runs_only_remaining_arm(self):
  workflow=(Path(__file__).parent/'hosted-pair.yml').read_text();self.assertIn('for index in 2; do',workflow);self.assertNotIn('for index in 1 2;',workflow)
if __name__=='__main__':unittest.main()
