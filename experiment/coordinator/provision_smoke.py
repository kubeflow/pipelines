#!/usr/bin/env python3
"""Preserve sanitized stage status even when provisioning fails."""
import argparse,subprocess,sys,time,importlib.util,json
from pathlib import Path
from coordinator import save,capture_manifest
p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--state',required=True);p.add_argument('--provision',required=True);p.add_argument('--output',required=True);a=p.parse_args()
out=Path(a.output);out.mkdir(parents=True,mode=0o700,exist_ok=False)
rows=[]
try:
    for action in ['validate','create','deploy','tenants','upload']:
        rows.append({'stage':action,'status':'dispatched'});save(out/'provision-stages.json',rows);start=time.monotonic()
        result=subprocess.run([sys.executable,str(Path(a.provision)/'provision.py'),action,'--config',a.config,'--state',a.state],timeout=2700)
        rows[-1].update(status='success' if result.returncode==0 else 'failed',exit_code=result.returncode,elapsed_seconds=round(time.monotonic()-start,3));save(out/'provision-stages.json',rows)
        if result.returncode:raise SystemExit(result.returncode)
finally:
    try:
        spec=importlib.util.spec_from_file_location('provision_candidate',Path(a.provision)/'provision.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        owner=module.Provision(json.loads(Path(a.config).read_text()),a.state);owner.guard()
        pods=json.loads(owner.k('get','pods','-A','-o','json',timeout=30))
        summary=[]
        for pod in pods.get('items',[])[:200]:
            status=pod.get('status',{});containers=[]
            for c in status.get('initContainerStatuses',[])+status.get('containerStatuses',[]):
                states={k:{field:value for field,value in v.items() if field in ('reason','exitCode','signal','startedAt','finishedAt')} for k,v in c.get('state',{}).items()}
                containers.append({'name':c.get('name'),'ready':c.get('ready'),'restartCount':c.get('restartCount'),'state':states})
            summary.append({'namespace':pod['metadata']['namespace'],'name':pod['metadata']['name'],'phase':status.get('phase'),'reason':status.get('reason'),'containers':containers})
        save(out/'pod-status.json',{'pods':summary,'truncated':len(pods.get('items',[]))>200})
        events=json.loads(owner.k('get','events','-A','-o','json',timeout=30))
        save(out/'event-reasons.json',{'events':[{'namespace':e['metadata']['namespace'],'type':e.get('type'),'reason':e.get('reason'),'count':e.get('count'),'object':{k:v for k,v in e.get('involvedObject',{}).items() if k in ('kind','namespace','name')}} for e in events.get('items',[])[-200:]],'truncated':len(events.get('items',[]))>200,'messages_omitted':True})
    except Exception as exc:save(out/'diagnostic-gap.json',{'error_type':type(exc).__name__,'raw_error_omitted':True})
    capture_manifest(out)
