"""Separate scenario preparation/calibration; never edits an existing cluster."""
import argparse,datetime as dt,hashlib,json,subprocess,time
from pathlib import Path
SOURCE='6bea51e2003f9072a55068fb85e03cdb9391435d'
NODE='kindest/node@sha256:21c46cf61fd45873f89e6a1bfcba4b7904dffa84c2bec88aeeca9a0409af4725'
STEPS=((1,16),(4,64),(16,256),(64,1000))
#5s lead +33s timeline +12s observer context tail +10s durable output margin.
CALIBRATION_OBSERVER_TIMEOUT_SECONDS=60
SCENARIO={'name':'constrained-stock-apf-bulk-list-v2','source':SOURCE,'node_image':NODE,'apiserver_flags':{'max-requests-inflight':10,'max-mutating-requests-inflight':10},'apf_policy_mutations':False,'calibration_steps':[{'workers':w,'offered_rps':r} for w,r in STEPS],'measurement_client_qps':[20,-1],'burst':40,'minimum_observed_shared_seats':1,'saturation_consecutive_samples':10,'calibration_seconds_per_step':33,'pipeline_variant':'two-active-per-tenant-total-eight-v1','caller_budget':{'kfp_reads':64,'background':64,'quiet_kubernetes':64,'pipeline_lifecycle_serial_per_tenant':4},'background_pressure_phases':['sustained','burst1','burst2','burst3'],'background_max_offers':210000,'background_list':{'configmaps':1024,'payload_bytes_each':1024,'namespace_list_limit':2048,'named_get_update_unchanged':True},'acceptance_complete':False}
def digest(v):return hashlib.sha256(json.dumps(v,sort_keys=True,separators=(',',':')).encode()).hexdigest()
def save(p,v):
 p.write_text(json.dumps(v,indent=2)+'\n');p.chmod(0o600)
def policy_digest(p):
 return digest({key:sorted([{'name':v['metadata']['name'],'spec':v['spec']} for v in p[key]['items']],key=lambda v:v['name']) for key in ('flowschemas','priorities')})
def pressure_proof(events,summary):
 streak=best=0;last=None;capacity=[];failures=[];health_streak=health_worst=0;reset=False
 for r in events:
  if r.get('kind')!='snapshot' or r.get('index',-1)<0:continue
  if r.get('missing'):failures.append(r['index'])
  reset |= bool(r.get('counter_reset'))
  health_bad=not r.get('ready',False) or not r.get('live',False)
  health_streak=(health_streak+1 if last is not None and r['index']==last+1 else 1) if health_bad else 0
  health_worst=max(health_worst,health_streak)
  vals={}
  for m in r.get('metrics') or []:vals[m['name']]=vals.get(m['name'],0)+m['value']
  seats=vals.get('apiserver_flowcontrol_current_limit_seats')
  if seats is not None:capacity.append(seats)
  good=not r.get('missing') and not r.get('counter_reset') and seats is not None and seats>=1 and vals.get('apiserver_flowcontrol_current_inqueue_requests',0)>0 and vals.get('apiserver_flowcontrol_current_executing_seats',0)>=seats
  streak=(streak+1 if last is not None and r['index']==last+1 else 1) if good else 0;best=max(best,streak);last=r['index']
 classified=summary.get('classification_failures')==0 and summary.get('raw_http_attempts',0)>0
 durable=summary.get('snapshot_count')==34 and summary.get('baseline_captured_before_start') is True and not ({'event_write','snapshot_count','competitor_classification'} & set(summary.get('missing',[])))
 return {'proven':best>=10 and not reset and health_worst<=5 and bool(capacity) and min(capacity)>=1 and classified and durable,'longest_queued_at_capacity_seconds':best,'minimum_observed_shared_seats':min(capacity) if capacity else None,'missing_sample_indices':failures,'maximum_consecutive_missing_or_failed_health':health_worst,'counter_reset':reset,'competitor_headers_verified':classified,'kfp_live_classification_verified':False,'acceptance_complete':False,'observation_complete':summary.get('observation_complete',False)}
def config_for(template,directory,workers,rps):
 start=dt.datetime.now(dt.timezone.utc)+dt.timedelta(seconds=5);at=start;windows=[]
 for name,kind,seconds in [('calibration','load',30),('after_load','probe',1),('drain','drain',1),('after_drain','probe',1)]:
  end=at+dt.timedelta(seconds=seconds);windows.append({'name':name,'kind':kind,'start':at.isoformat(),'end':end.isoformat()});at=end
 save(directory/'timeline.json',windows)
 c=dict(template,pressure_only=False,start_at=start.isoformat(),timeline_file=str(directory/'timeline.json'),seconds=33,max_inflight=workers,rps=rps);save(directory/'config.json',c)
 return c
def execute(binary,template,root,stock):
 deadline=time.monotonic()+370;selected=None
 for number,(workers,rps) in enumerate(STEPS,1):
  d=root/f'calibration-{number}';d.mkdir(mode=0o700);config_for(template,d,workers,rps)
  remaining=deadline-time.monotonic()
  if remaining<CALIBRATION_OBSERVER_TIMEOUT_SECONDS:raise RuntimeError('bounded calibration exhausted')
  code=subprocess.run([binary,'--config',str(d/'config.json'),'--out',str(d/'capture')],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=CALIBRATION_OBSERVER_TIMEOUT_SECONDS).returncode
  captured=d/'capture';summary=json.loads((captured/'summary.json').read_text());policies=json.loads((captured/'policies.json').read_text())
  if policy_digest(policies)!=stock:raise RuntimeError('stock APF specs changed')
  with (captured/'events.jsonl').open() as f:proof=pressure_proof((json.loads(line) for line in f),summary)
  proof.update(observer_exit=code,workers=workers,offered_rps=rps,policy_spec_sha256=stock);save(d/'pressure-proof.json',proof)
  if proof['proven'] and code in (0,1):selected=proof;break
  if proof['maximum_consecutive_missing_or_failed_health']>5 or proof['counter_reset']:break
  # Calibration traffic stops completely between steps. No policies or seats change.
  time.sleep(30)
 result={'scenario':SCENARIO,'scenario_sha256':digest(SCENARIO),'stock_policy_spec_sha256':stock,'selected':selected,'measurements_authorized_by_calibration':selected is not None,'note':'Calibration verifies competitor flow only; actual KFP caller classification and runtime evidence remain required.'}
 save(root/'frozen-calibration.json',result)
 if selected is None:raise RuntimeError('No proven overload within64workers/1000RPS; do not measure or tune silently')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--out',required=True);p.add_argument('--observer-binary');p.add_argument('--observer-template');p.add_argument('--stock-policies');a=p.parse_args();out=Path(a.out);out.mkdir(mode=0o700)
 save(out/'scenario.json',dict(SCENARIO,scenario_sha256=digest(SCENARIO)))
 save(out/'kind-config.json',{'kind':'Cluster','apiVersion':'kind.x-k8s.io/v1alpha4','nodes':[{'role':'control-plane','kubeadmConfigPatches':['kind: ClusterConfiguration\napiServer:\n  extraArgs:\n    - name: max-requests-inflight\n      value: "10"\n    - name: max-mutating-requests-inflight\n      value: "10"\n']}]})
 if a.observer_binary:
  if not a.observer_template or not a.stock_policies:p.error('calibration requires owned observer template and archived stock policies')
  execute(a.observer_binary,json.loads(Path(a.observer_template).read_text()),out,policy_digest(json.loads(Path(a.stock_policies).read_text())))
