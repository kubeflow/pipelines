"""Run a new constrained pair once, preserving failed gates and stopping unsafe continuation."""
import argparse,gzip,json,subprocess,sys
from pathlib import Path
from coordinator import load,save

def health_window_safe(path):
    opener=gzip.open if path.suffix=='.gz' else open
    used=count=worst=streak=0;last=None
    with opener(path,'rt') as f:
        for line in f:
            used+=len(line)
            if used>512<<20 or len(line)>8<<20:return False
            row=json.loads(line)
            if row.get('kind')!='snapshot':continue
            index=row['index'];count+=1
            if last is not None and index!=last+1:return False
            bad=row.get('ready') is not True or row.get('live') is not True
            streak=streak+1 if bad else 0;worst=max(worst,streak);last=index
            if row.get('counter_reset'):return False
    return count>0 and worst<=5

def safe_next_arm(directory):
    try:
        process=load(directory/'process-results.json');workload=load(directory/'workload/results.json',32<<20)
        runs=[r for values in workload['runs'].values() for r in values]
        auth=load(directory/'auth-after.json');health=load(directory/'health-result.json')
        auth_ok=len(auth)==len(workload['runs']) and {r.get('actor') for r in auth}==set(workload['runs']) and all(r.get('complete') is True for r in auth)
        return process.get('completed_timeline') is True and process.get('auth_during_exit')==0 and process.get('background_exit') in (0,1) and health.get('passed') is True and auth_ok and bool(runs) and all(r.get('state') in ('SUCCEEDED','FAILED','CANCELED','CANCELLED','SKIPPED') for r in runs) and health_window_safe(directory/'observer/events.jsonl') and health_window_safe(directory/'background/events.jsonl.gz')
    except (OSError,ValueError,KeyError,TypeError,EOFError):return False
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--out',required=True);a=p.parse_args();out=Path(a.out);out.mkdir(parents=True,exist_ok=True);outcomes=[]
 calibration=load(load(a.config)['background_calibration']); selected=calibration.get('selected') or {}
 if selected.get('workers')!=64 or selected.get('offered_rps')!=1000 or selected.get('proven') is not True or calibration.get('measurements_authorized_by_calibration') is not True:raise SystemExit('Diagnostic requires proven frozen64worker/1000RPS pressure')
 save(out/'diagnostic-continuation.json',{'prior_run_id':38525574937,'prior_head':'65c1e6fce','preserved_completed_arm':1,'run_indices':[2],'fresh_cluster':True,'order_confound':True,'attempt_limit':1,'note':'Negative-only diagnostic; prior20arm preserved, no complete paired/default acceptance claim.'})
 for index in (2,):
  dest=out/f'arm-{index}'
  if dest.exists():raise SystemExit('Arm directory already exists; no rerun or overwrite')
  save(out/f'arm-{index}-dispatch.json',{'index':index,'qps':20 if index==1 else -1,'scenario':'constrained-fixed-four-seat-bulk-list-v3'})
  # The coordinator owns a 45-minute SIGALRM watchdog and bounded child-group
  # cleanup in finally. Do not add subprocess.run timeout's abrupt parent kill.
  try:
   result=subprocess.run([sys.executable,str(Path(__file__).with_name('coordinator.py')),'arm','--config',a.config,'--out',str(dest),'--index',str(index)],check=False)
  except OSError as error:
   save(out/f'arm-{index}-outcome.json',{'index':index,'launch_error_type':type(error).__name__,'safe_next_arm':False})
   raise
  record={'index':index,'returncode':result.returncode,'safe_next_arm':safe_next_arm(dest)};outcomes.append(record);save(out/f'arm-{index}-outcome.json',record)
  if not record['safe_next_arm']:break
 save(out/'pair-process-results.json',{'arms':outcomes,'acceptance_complete':False,'default_change_supported':False})
 raise SystemExit(0 if len(outcomes)==1 and all(v['returncode']==0 and v['safe_next_arm'] for v in outcomes) else 1)
