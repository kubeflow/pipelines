import gzip,json,tempfile,unittest
from pathlib import Path
from prepare_pair import constrained_flags
from coordinator import save
from run_pair import safe_next_arm
class Integration(unittest.TestCase):
 def test_live_flags_required_not_written_config(self):
  pod={'metadata':{'uid':'owned'},'spec':{'containers':[{'name':'kube-apiserver','command':['kube-apiserver','--max-requests-inflight=10','--max-mutating-requests-inflight=10']}]}}
  self.assertEqual(constrained_flags({'items':[pod]})['pod_uid'],'owned')
  pod['spec']['containers'][0]['command'][-1]='--max-mutating-requests-inflight=0'
  with self.assertRaises(RuntimeError):constrained_flags({'items':[pod]})
 def test_continue_only_completed_reconciled_no_auth_failure(self):
  with tempfile.TemporaryDirectory() as d:
   p=Path(d);(p/'workload').mkdir();(p/'observer').mkdir();(p/'background').mkdir()
   row=json.dumps({'kind':'snapshot','index':0,'ready':True,'live':True})+'\n';(p/'observer/events.jsonl').write_text(row)
   with gzip.open(p/'background/events.jsonl.gz','wt') as f:f.write(row)
   save(p/'auth-after.json',[{'actor':'a','complete':True}]);save(p/'health-result.json',{'passed':True});save(p/'workload/results.json',{'runs':{'a':[{'state':'SUCCEEDED'}]}})
   save(p/'process-results.json',{'completed_timeline':True,'auth_during_exit':0,'observer_exit':1,'background_exit':1});self.assertTrue(safe_next_arm(p))
   save(p/'process-results.json',{'completed_timeline':True,'auth_during_exit':1,'background_exit':1});self.assertFalse(safe_next_arm(p))
   save(p/'process-results.json',{'completed_timeline':True,'auth_during_exit':0,'background_exit':1});save(p/'workload/results.json',{'runs':{'a':[{'state':'RUNNING'}]}});self.assertFalse(safe_next_arm(p))
 def test_health_gaps_are_not_healthy(self):
  from run_pair import health_window_safe
  with tempfile.TemporaryDirectory() as d:
   p=Path(d)/'events.jsonl';p.write_text(''.join(json.dumps({'kind':'snapshot','index':i,'ready':False,'live':False})+'\n' for i in range(6)))
   self.assertFalse(health_window_safe(p))

class Wrapper(unittest.TestCase):
 def test_zero_exit_cannot_skip_safety_guard(self):
  import runpy,subprocess,sys
  from unittest.mock import patch
  with tempfile.TemporaryDirectory() as d:
   calls=[]
   config=Path(d)/'config.json';calibration=Path(d)/'calibration.json'
   save(config,{'background_calibration':str(calibration)});save(calibration,{'selected':{'workers':64,'offered_rps':1000,'proven':True},'measurements_authorized_by_calibration':True})
   def fake_run(argv,**kwargs):
    self.assertNotIn('timeout',kwargs);calls.append(argv)
    dest=Path(argv[argv.index('--out')+1]);dest.mkdir()
    save(dest/'process-results.json',{'completed_timeline':True,'auth_during_exit':0,'background_exit':0})
    save(dest/'health-result.json',{'passed':False})
    return subprocess.CompletedProcess(argv,0)
   with patch.object(sys,'argv',['run_pair.py','--config',str(config),'--out',d]),patch('subprocess.run',fake_run):
    with self.assertRaises(SystemExit) as e:runpy.run_path(str(Path(__file__).parent/'run_pair.py'),run_name='__main__')
   self.assertEqual(e.exception.code,1);self.assertEqual(len(calls),1)

 def test_final_unsafe_arm_is_nonzero_even_if_child_exits_zero(self):
  import runpy,subprocess,sys
  from unittest.mock import patch
  import coordinator
  with tempfile.TemporaryDirectory() as d:
   def fake_run(argv,**kwargs):
    dest=Path(argv[argv.index('--out')+1]);dest.mkdir()
    return subprocess.CompletedProcess(argv,0)
   # The sole diagnostic arm still fails when its post-load health is missing.
   def fake_load(path,*args):
    if str(path)=='unused':return {'background_calibration':'calibration'}
    if str(path)=='calibration':return {'selected':{'workers':64,'offered_rps':1000,'proven':True},'measurements_authorized_by_calibration':True}
    if 'arm-2' in str(path):raise OSError('missing capture')
    if path.name=='process-results.json':return {'completed_timeline':True,'auth_during_exit':0,'background_exit':0}
    if path.name=='results.json':return {'runs':{'a':[{'state':'SUCCEEDED'}]}}
    if path.name=='auth-after.json':return [{'actor':'a','complete':True}]
    if path.name=='health-result.json':return {'passed':True}
    raise AssertionError(path)
   real_open=open
   def fake_open(path,*args,**kwargs):
    if str(path).endswith('events.jsonl'):
     import io
     return io.StringIO(json.dumps({'kind':'snapshot','index':0,'ready':True,'live':True})+'\n')
    return real_open(path,*args,**kwargs)
   import io
   with patch.object(sys,'argv',['run_pair.py','--config','unused','--out',d]),patch('subprocess.run',fake_run),patch.object(coordinator,'load',fake_load),patch('builtins.open',fake_open),patch('gzip.open',lambda *a,**kw:io.StringIO(json.dumps({'kind':'snapshot','index':0,'ready':True,'live':True})+'\n')):
    with self.assertRaises(SystemExit) as e:runpy.run_path(str(Path(__file__).parent/'run_pair.py'),run_name='__main__')
   self.assertEqual(e.exception.code,1)
   self.assertEqual(len(json.loads((Path(d)/'pair-process-results.json').read_text())['arms']),1)

if __name__=='__main__':unittest.main()
