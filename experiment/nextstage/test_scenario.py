import unittest
from scenario import pressure_proof,policy_digest,SCENARIO,STEPS
class Scenario(unittest.TestCase):
 def rows(self,n=10,seats=1,queue=1):
  return [{'kind':'snapshot','index':i,'missing':[],'ready':True,'live':True,'metrics':[{'name':'apiserver_flowcontrol_current_limit_seats','value':seats},{'name':'apiserver_flowcontrol_current_inqueue_requests','value':queue},{'name':'apiserver_flowcontrol_current_executing_seats','value':seats}]} for i in range(n)]
 def proof(self,r):return pressure_proof(r,{'classification_failures':0,'raw_http_attempts':20,'snapshot_count':34,'baseline_captured_before_start':True})
 def test_requires_ten_consecutive_queued_positive_samples(self):
  self.assertTrue(self.proof(self.rows())['proven']);self.assertFalse(self.proof(self.rows(9))['proven']);self.assertFalse(self.proof(self.rows(seats=0))['proven']);self.assertFalse(self.proof(self.rows(queue=0))['proven'])
 def test_gap_or_missing_fail(self):
  r=self.rows();r[5]['index']=30;self.assertFalse(self.proof(r)['proven']);r=self.rows();r[5]['missing']=['readyz'];self.assertFalse(self.proof(r)['proven'])
 def test_capacity_does_not_mean_saturation(self):self.assertFalse(self.proof(self.rows(queue=0))['proven'])
 def test_policy_hash_ignores_cluster_uids_not_policy_specs(self):
  p={'flowschemas':{'items':[]},'priorities':{'items':[{'metadata':{'name':'stock','uid':'a'},'spec':{'type':'Limited'}}]}};h=policy_digest(p);p['priorities']['items'][0]['metadata']['uid']='b';self.assertEqual(h,policy_digest(p));p['priorities']['items'][0]['spec']['type']='Exempt';self.assertNotEqual(h,policy_digest(p))
 def test_isolated_missing_does_not_claim_complete_acceptance(self):
  r=self.rows(12);r[11]['missing']=['sampling_lag'];p=self.proof(r);self.assertTrue(p['proven']);self.assertFalse(p['acceptance_complete']);self.assertEqual(p['missing_sample_indices'],[11])
 def test_long_health_missing_or_reset_blocks_selection(self):
  r=self.rows(20)
  for v in r[10:16]:v['ready']=False;v['missing']=['readyz']
  self.assertFalse(self.proof(r)['proven']);r=self.rows(20);r[19]['counter_reset']=True;self.assertFalse(self.proof(r)['proven'])
 def test_frozen_nonzero_bounded_scenario(self):
  self.assertEqual(SCENARIO['apiserver_flags'],{'max-requests-inflight':10,'max-mutating-requests-inflight':10});self.assertFalse(SCENARIO['apf_policy_mutations']);self.assertEqual(STEPS[-1],(64,1000))
 def test_parent_timeout_allows_terminal_request_drain_and_capture(self):
  import subprocess,tempfile
  from pathlib import Path
  from unittest.mock import patch
  from scenario import execute
  with tempfile.TemporaryDirectory() as d:
   def bounded_observer(argv,**kwargs):
    # Last offered call may take10s; parent previously killed it at45s.
    self.assertGreater(kwargs['timeout'],5+33+10)
    # Honor observer's full12s context tail and bounded output margin.
    self.assertEqual(kwargs['timeout'],5+33+12+10)
    self.assertLessEqual(4*kwargs['timeout']+4*30,370)
    raise subprocess.TimeoutExpired(argv,kwargs['timeout'])
   with patch('scenario.subprocess.run',bounded_observer):
    with self.assertRaises(subprocess.TimeoutExpired):execute('observer',{},Path(d),'stock')
if __name__=='__main__':unittest.main()
