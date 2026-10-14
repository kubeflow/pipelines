#!/usr/bin/env python3
"""One authenticated, non-retried CreateRun; bounded terminal/output validation."""
import argparse, importlib.util, json, os, re, subprocess, sys, time, urllib.request
from pathlib import Path
from coordinator import save, load, stop, endpoint_ready, capture_manifest

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise RuntimeError('redirect refused')

def request(endpoint, path, token, body=None):
    req=urllib.request.Request(endpoint+path, data=None if body is None else json.dumps(body).encode(), headers={'Authorization':'Bearer '+token,'Content-Type':'application/json'})
    with urllib.request.build_opener(NoRedirect()).open(req,timeout=10) as response:
        data=response.read((8<<20)+1)
    if len(data)>8<<20: raise RuntimeError('response size bound')
    return json.loads(data)

def full_run(endpoint,run_id,token):
    # DEFAULT deliberately omits tasks; output assertions require the FULL view.
    return request(endpoint,'/apis/v2beta1/runs/'+run_id+'?view=FULL',token)

def run(args):
    out=Path(args.output);out.mkdir(mode=0o700,parents=True,exist_ok=False)
    spec=importlib.util.spec_from_file_location('provision',Path(args.provision)/'provision.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
    config=load(args.config);owner=p.Provision(config,args.state);owner.guard()
    skeleton=load(Path(args.state)/'workload-config-skeleton.json');tenant=skeleton['tenants'][0]
    fixture=next(f for f in tenant['fixtures'] if f['name']=='flip_coin')
    token=Path(tenant['token_file']).read_text().strip();endpoint=skeleton['endpoint'];forward=None
    spec=importlib.util.spec_from_file_location('evaluate_fixture',Path(args.fixtures)/'evaluate.py');e=importlib.util.module_from_spec(spec);spec.loader.exec_module(e)
    try:
        forward=subprocess.Popen(['kubectl','--kubeconfig',str(owner.kubeconfig),'--context',owner.context,'-n','kubeflow','port-forward','svc/ml-pipeline',str(config['port'])+':8888','--address=127.0.0.1'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
        endpoint_ready(endpoint,forward)
        body={'display_name':'apf-hosted-smoke','namespace':tenant['namespace'],'experiment_id':tenant['experiment_id'],'service_account':tenant['service_account'],'pipeline_version_reference':{'pipeline_id':fixture['pipeline_id'],'pipeline_version_id':fixture['version_id']},'runtime_config':{'parameters':{}}}
        save(out/'create-dispatch.json',{'fixture':'flip_coin','namespace':tenant['namespace'],'retry_allowed':False})
        created=request(endpoint,'/apis/v2beta1/runs',token,body);rid=created['run_id']
        if not re.fullmatch('[a-zA-Z0-9-]{1,128}',rid):raise RuntimeError('invalid run identifier')
        save(out/'created.json',{'run_id':rid,'source_sha':config['source_sha']})
        deadline=time.monotonic()+900;terminal_deadline=None
        while time.monotonic()<deadline:
            if forward.poll() is not None:raise RuntimeError('forward exited')
            raw=full_run(endpoint,rid,token)
            save(out/'run.json',raw)
            if raw.get('state') in ('FAILED','CANCELED'):raise RuntimeError('pipeline failed')
            if raw.get('state')=='SUCCEEDED':
                terminal_deadline=terminal_deadline or time.monotonic()+60
                result=e.evaluate('flip_coin',raw);save(out/'output-validation.json',result)
                if result['passed']:
                    save(out/'result.json',{'smoke_passed':True,'overload_acceptance':False,'source_sha':config['source_sha'],'fixture':'flip_coin','run_id':rid});return
                if time.monotonic()>=terminal_deadline:raise RuntimeError('terminal output persistence incomplete')
            time.sleep(3)
        raise RuntimeError('pipeline deadline exceeded')
    finally:
        if forward is not None:stop(forward)
        capture_manifest(out)

def main():
    p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--state',required=True);p.add_argument('--provision',required=True);p.add_argument('--fixtures',required=True);p.add_argument('--output',required=True);args=p.parse_args();os.umask(0o077)
    try:run(args)
    except Exception as exc:
        out=Path(args.output)
        if out.is_dir():save(out/'failure.json',{'error_type':type(exc).__name__,'accepted_run_reconciliation_required':True});capture_manifest(out)
        print('Smoke failed; see bounded evidence. Raw error suppressed.',file=sys.stderr);return 1
    return 0
if __name__=='__main__':sys.exit(main())
