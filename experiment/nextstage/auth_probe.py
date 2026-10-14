#!/usr/bin/env python3
"""Bounded real owner checks and denied writes; never retry a write."""
import argparse,json,time,urllib.error,urllib.parse,urllib.request,uuid
from pathlib import Path
from coordinator import load,save,stamp,utc,iso,read_events
from integration import NoRedirect

def call(endpoint,path,token,body=None):
    request=urllib.request.Request(endpoint+path,data=None if body is None else json.dumps(body).encode(),headers={'Authorization':'Bearer '+token,'Content-Type':'application/json'})
    try:
        with urllib.request.build_opener(NoRedirect()).open(request,timeout=10) as response:
            data=response.read((8<<20)+1)
            if len(data)>8<<20:raise RuntimeError('response bound')
            return response.status,json.loads(data)
    except urllib.error.HTTPError as error:
        # Do not retain raw API error bodies.
        return error.code,None

def list_names(endpoint,tenant,token):
    result=set();page=''
    for _ in range(10):
        query=urllib.parse.urlencode({'namespace':tenant['namespace'],'experiment_id':tenant['experiment_id'],'page_size':100,'page_token':page})
        status,data=call(endpoint,'/apis/v2beta1/runs?'+query,token)
        if status!=200:raise RuntimeError('owner listing failed')
        for run in data.get('runs',[]):result.add(run.get('display_name',''))
        page=data.get('next_page_token','')
        if not page:return result
    raise RuntimeError('owner listing truncated')

def probe(workload,targets,out,phase):
    rows=[];tenants=workload['tenants'];endpoint=workload['endpoint'];output=Path(out)
    for i,actor in enumerate(tenants):
        target=tenants[(i+1)%len(tenants)];target_id=targets[target['name']];actor_id=targets[actor['name']]
        atoken=Path(actor['token_file']).read_text().strip();ttoken=Path(target['token_file']).read_text().strip()
        name='apf-denied-'+uuid.uuid4().hex
        before=list_names(endpoint,target,ttoken)
        if name in before:raise RuntimeError('unexpected probe name collision')
        fixture=target['fixtures'][0]
        body={'display_name':name,'namespace':target['namespace'],'experiment_id':target['experiment_id'],'service_account':target['service_account'],'pipeline_version_reference':{'pipeline_id':fixture['pipeline_id'],'pipeline_version_id':fixture['version_id']}}
        row={'started_at':iso(utc()),'phase':phase,'actor':actor['name'],'target':target['name'],'probe_name':name,'write_dispatched':True,'complete':False}
        rows.append(row);save(output,rows) # durable dispatch before possibly accepted write
        try:
            status,data=call(endpoint,'/apis/v2beta1/runs',atoken,body)
        except Exception as error:
            # A transport or decode failure is not a denial. Never resend this POST.
            row['write_outcome']='unknown'
            row['write_error_type']=type(error).__name__
            row['write_failed_at']=iso(utc())
            row['requires_accepted_run_reconciliation']=True
            save(output,rows)
            row['owner_absence_observations']=[]
            try:
                for delay in (0,2):
                    time.sleep(delay)
                    names=list_names(endpoint,target,ttoken)
                    row['owner_absence_observations'].append(name not in names)
                    save(output,rows)
                row['owner_reconciliation']='absent_in_two_observations' if all(row['owner_absence_observations']) else 'named_run_observed'
            except Exception as reconcile_error:
                row['owner_reconciliation']='incomplete'
                row['owner_reconciliation_error_type']=type(reconcile_error).__name__
            row['reconciliation_finished_at']=iso(utc())
            # Even two absent listings do not establish the POST's HTTP outcome,
            # nor rule out a request still executing after client disconnection.
            save(output,rows)
            raise RuntimeError('authorization write outcome unknown; no retry') from None
        row['write_status']=status
        if data and data.get('run_id'):row['unexpected_accepted_run_id']=data['run_id']
        save(output,rows)
        row['cross_read_status']=call(endpoint,'/apis/v2beta1/runs/'+target_id,atoken)[0]
        row['invalid_token_status']=call(endpoint,'/apis/v2beta1/runs/'+actor_id,atoken+'invalid')[0]
        # Two owner observations avoid accepting one stale listing as absence evidence.
        row['owner_absence_observations']=[]
        for delay in (0,2):
            time.sleep(delay);names=list_names(endpoint,target,ttoken)
            row['owner_absence_observations'].append(name not in names)
            save(output,rows)
        row['completed_at']=iso(utc())
        row['complete']=all(row[k] in (401,403) for k in ('write_status','cross_read_status','invalid_token_status')) and all(row['owner_absence_observations']) and 'unexpected_accepted_run_id' not in row
        save(output,rows)
        if not row['complete']:raise RuntimeError('authorization probe failure; preserve accepted IDs')
    return rows

def targets_from(path,tenants):
    latest={r['tenant']:r for r in read_events(Path(path)) if r.get('complete')}
    targets={t['name']:latest[t['name']]['child_run_id'] for t in tenants}
    if any(not rid or not all(c.isalnum() or c=='-' for c in rid) for rid in targets.values()):raise RuntimeError('unsafe target ID')
    return targets

def main():
    p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--targets',required=True);p.add_argument('--out',required=True);p.add_argument('--phase',required=True);p.add_argument('--at');p.add_argument('--until');a=p.parse_args();workload=load(a.config)
    if a.at:
        when=stamp(a.at)
        if (when-utc()).total_seconds()>300:raise RuntimeError('scheduled probe delay bound')
        while utc()<when:time.sleep(.1)
    started=utc()
    probe(workload,targets_from(a.targets,workload['tenants']),a.out,a.phase)
    if a.until and utc()>stamp(a.until):
        rows=load(a.out)
        for row in rows:row['complete']=False;row['outside_required_window']=True
        save(a.out,rows);raise RuntimeError('probes exceeded required load window')
if __name__=='__main__':
    try:main()
    except Exception as e:
        print('Authorization probe stopped: '+type(e).__name__);raise SystemExit(1)
