import json,tempfile,unittest
from pathlib import Path
from coordinator import background_config,caller_budget,policy_spec_digest,verify_stock_policies,completed_measurement_capture,save,digest,UTC
import datetime as dt
class BackgroundCoordinatorTests(unittest.TestCase):
 def test_frozen_pressure_uses_shared_clock_and_distinct_identity(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=Path(tmp);(p/'frozen').write_text(json.dumps({'measurements_authorized_by_calibration':True,'selected':{'proven':True,'workers':64,'offered_rps':1000}}))
   (p/'template').write_text(json.dumps({'priority_uid':'p','token_file':'background-private-token'}))
   c={'background_calibration':str(p/'frozen'),'background_template':str(p/'template')}
   quiet={'rps':20,'max_inflight':64,'priority_uid':'p','token_file':'quiet-private-token','start_at':'start','seconds':1230,'timeline_file':'timeline','kubeconfig':'kube','context':'owned','cluster_uid':'uid'}
   bg=background_config(c,quiet)
   self.assertTrue(bg['pressure_only']);self.assertEqual(bg['timeline_file'],quiet['timeline_file']);self.assertEqual(bg['start_at'],quiet['start_at'])
   self.assertEqual(quiet['rps'],20)
   budget=caller_budget({'tenants':[{}]*4,'max_inflight_per_tenant':16},quiet,bg)
   self.assertEqual(budget['load_callers_max_inflight'],192);self.assertEqual(budget['background_max_offers'],210000);self.assertEqual(budget['total_harness_callers_max_inflight'],203)
   quiet['rps']=21
   with self.assertRaises(RuntimeError):background_config(c,quiet)
 def test_uncalibrated_never_launches(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=Path(tmp)/'frozen';p.write_text('{}')
   with self.assertRaisesRegex(RuntimeError,'not calibrated'):background_config({'background_calibration':str(p)}, {})
 def test_policy_guard_detects_spec_change_and_records_evidence(self):
  policies={'flowschemas':{'items':[{'metadata':{'name':'f'},'spec':{'x':1}}]},'priorities':{'items':[]}}
  with tempfile.TemporaryDirectory() as tmp:
   p=Path(tmp);save(p/'frozen',{'stock_policy_spec_sha256':policy_spec_digest(policies)})
   class Owner:
    def k(self,*args):return json.dumps(policies['flowschemas' if args[1]=='flowschemas' else 'priorities']).encode()
   verify_stock_policies(Owner(),{'background_calibration':str(p/'frozen')},p,'before')
   policies['flowschemas']['items'][0]['spec']['x']=2
   with self.assertRaisesRegex(RuntimeError,'specs changed'):verify_stock_policies(Owner(),{'background_calibration':str(p/'frozen')},p,'after')
   self.assertFalse(json.loads((p/'policy-after.json').read_text())['unchanged'])
 def test_background_capture_must_match_full_timeline_and_policies(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=Path(tmp);(p/'workload').mkdir();(p/'observer').mkdir();(p/'background').mkdir()
   end=dt.datetime(2026,1,1,tzinfo=UTC);timeline=[{'end':end.isoformat(),'name':'done'}]
   w={'phases':[{'seconds':1}],'probe_seconds':0,'drain_seconds':0,'tenants':[{'name':'one'}]}
   save(p/'workload'/'timeline.json',timeline);save(p/'workload'/'results.json',{'timeline':timeline})
   (p/'workload'/'events.jsonl').write_text(json.dumps({'operation':'phase_end','phase':'done','tenant':'one'})+'\n')
   summary={'snapshot_count':2,'timeline_sha256':digest(p/'workload'/'timeline.json')}
   save(p/'observer'/'summary.json',summary);save(p/'background-config.json',{})
   self.assertFalse(completed_measurement_capture(p,w,end,end))
   policies={'flowschemas':{'items':[]},'priorities':{'items':[]}}
   save(p/'background'/'summary.json',summary);save(p/'background'/'policies.json',policies);save(p/'policy-before.json',{'expected_spec_sha256':policy_spec_digest(policies)})
   self.assertTrue(completed_measurement_capture(p,w,end,end))
   summary['snapshot_count']=1;save(p/'background'/'summary.json',summary)
   self.assertFalse(completed_measurement_capture(p,w,end,end))
if __name__=='__main__':unittest.main()
