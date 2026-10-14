"""Best-effort real Kubernetes caller/APF join. Never persists raw logs."""
import argparse,datetime as dt,hashlib,json,re,subprocess,sys,threading,time
from pathlib import Path
from project import project,MAX_LINE,LIMIT
from join import join

def stamp(value):return dt.datetime.fromisoformat(value.replace('Z','+00:00'))
def fingerprint(value):return hashlib.sha256(json.dumps(value,sort_keys=True,separators=(',',':')).encode()).hexdigest()

def observed_users(deployments,pods,namespaces):
    users=set()
    for obj in deployments.get('items',[]):
        ns=obj['metadata']['namespace'];sa=obj.get('spec',{}).get('template',{}).get('spec',{}).get('serviceAccountName','default')
        if ns in namespaces:users.add(f'system:serviceaccount:{ns}:{sa}')
    for obj in pods.get('items',[]):
        ns=obj['metadata']['namespace'];sa=obj.get('spec',{}).get('serviceAccountName','default')
        if ns in namespaces:users.add(f'system:serviceaccount:{ns}:{sa}')
    return users

def decode_line(raw,kind):
    # kubelet CRI log framing; retain only complete JSON records.
    if kind=='http':
        parts=raw.split(b' ',3)
        if len(parts)==4 and parts[1] in (b'stdout',b'stderr'):
            if parts[2]!=b'F':raise ValueError('partial CRI record')
            raw=parts[3]
    return json.loads(raw)

def project_file(command,kind,users,output,budget):
    remaining=budget[1]-time.monotonic()
    if remaining<=0:raise RuntimeError('capture deadline exceeded')
    proc=subprocess.Popen(command,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
    timer=threading.Timer(min(20,remaining),proc.kill);timer.start()
    projected=malformed=0
    try:
        for raw in iter(lambda:proc.stdout.readline(MAX_LINE+1),b''):
            budget[0]+=len(raw)
            if len(raw)>MAX_LINE or budget[0]>LIMIT:raise RuntimeError('raw input capture bound exceeded')
            try:record=project(decode_line(raw,kind),kind,users)
            except (ValueError,TypeError,AttributeError):malformed+=1;continue
            if record:
                output.write(json.dumps(record)+'\n');projected+=1
        if proc.wait(timeout=10):raise RuntimeError('node log read failed')
    finally:
        timer.cancel()
        if proc.poll() is None:proc.kill()
        proc.wait(timeout=10);proc.stdout.close()
    return projected,malformed

def snapshot(owner,namespaces):
    def identities(resource,sa_path):
        # Project at kubectl: never fetch/log workflow args, env or pod specs.
        expression='{range .items[*]}{.metadata.namespace}{"\\t"}{.metadata.name}{"\\t"}{'+sa_path+'}{"\\n"}{end}'
        text=owner.k('get',resource,'-A','-o','jsonpath='+expression,limit=1<<20).decode()
        items=[]
        for line in text.splitlines():
            parts=line.split('\t')
            if len(parts)!=3:raise RuntimeError('invalid identity projection')
            ns,name,sa=parts
            if ns not in namespaces:continue
            spec={'serviceAccountName':sa or 'default'}
            items.append({'metadata':{'namespace':ns,'name':name},'spec':{'template':{'spec':spec}} if resource=='deployments' else spec})
        return {'items':items}
    deployments=identities('deployments','.spec.template.spec.serviceAccountName')
    pods=identities('pods','.spec.serviceAccountName')
    policies={key:json.loads(owner.k('get',resource,'-o','json',limit=4<<20)) for key,resource in [('schemas','flowschemas'),('priorities','prioritylevelconfigurations')]}
    # UID and spec equality detect policy replacement or mutation across the window.
    policies={key:{o['metadata']['name']:{'metadata':{'name':o['metadata']['name'],'uid':o['metadata']['uid']},'spec':o['spec']} for o in value['items']} for key,value in policies.items()}
    api=[v for v in deployments['items'] if v['metadata']['namespace']=='kubeflow' and v['metadata']['name']=='ml-pipeline']
    if len(api)!=1:raise RuntimeError('actual API deployment identity unavailable')
    api_user='system:serviceaccount:kubeflow:'+api[0]['spec']['template']['spec'].get('serviceAccountName','default')
    return {'users':sorted(observed_users(deployments,pods,namespaces)),'api_user':api_user,'policies':policies}

def inventory(node,kind,bounded):
    root='/var/log/kubernetes' if kind=='audit' else '/var/log/pods'
    pattern='audit*' if kind=='audit' else '*kube-apiserver*/*.log*'
    args=['docker','exec',node,'find',root,'-type','f','-path' if kind=='http' else '-name',pattern]
    names=bounded(args,timeout=10,limit=65536).decode().splitlines()
    if len(names)>32:raise RuntimeError('node log file count bound exceeded')
    for name in names:
        if not name.startswith(root+'/') or not re.fullmatch(r'[A-Za-z0-9_./:-]+',name):raise RuntimeError('unexpected node log path')
    return sorted(names,key=lambda p:(not(p.endswith('/0.log') or p.endswith('/audit.log')),p))

def main():
    p=argparse.ArgumentParser();p.add_argument('mode',choices=['begin','finish']);p.add_argument('--coordinator-dir',required=True);p.add_argument('--config',required=True);p.add_argument('--out',required=True);p.add_argument('--start');p.add_argument('--end');a=p.parse_args()
    sys.path.insert(0,a.coordinator_dir)
    from coordinator import Owned,load,save,bounded
    out=Path(a.out);out.mkdir(parents=True,exist_ok=True)
    incomplete=[]
    try:
        c=load(a.config);owner=Owned(c);uid=owner.guard();namespaces={'kubeflow'}|set(load(Path(c['state'])/'tenants.json'))
        current=snapshot(owner,namespaces);current['cluster_uid']=uid
        if a.mode=='begin':save(out/'before.json',current);return
        if not a.start or not a.end:raise RuntimeError('measurement window absent')
        before=load(out/'before.json');save(out/'after.json',current)
        if before['cluster_uid']!=uid or before['policies']!=current['policies']:incomplete.append('cluster_or_policy_identity_changed')
        users=set(before['users'])|set(current['users']);node=c['cluster']+'-control-plane';budget=[0,time.monotonic()+60];projected={}
        for kind in ('audit','http'):
            paths=inventory(node,kind,bounded);save(out/(kind+'-inventory.json'),{'paths':paths})
            if not paths:incomplete.append(kind+'_logs_missing')
            count=bad=0
            with (out/(kind+'.jsonl')).open('w') as stream:
                for path in paths:
                    reader=['gzip','-cd',path] if path.endswith('.gz') else ['cat',path]
                    try:
                        n,b=project_file(['docker','exec',node,*reader],kind,users,stream,budget);count+=n;bad+=b
                    except RuntimeError:
                        incomplete.append(kind+'_read_or_capture_bound_failure')
                        break
            projected[kind]={'records':count,'malformed_or_partial':bad}
            if bad:incomplete.append(kind+'_malformed_or_partial_records')
        # Full retained audit set guards truncated-ID collisions across the capture;
        # filter the joined results to the actual completed measurement window.
        audit=[json.loads(v) for v in (out/'audit.jsonl').read_text().splitlines()]
        http=[json.loads(v) for v in (out/'http.jsonl').read_text().splitlines()]
        result=join(audit,http,before['policies']['schemas'],before['policies']['priorities'])
        start,end=stamp(a.start),stamp(a.end)
        selected=[v for v in result['records'] if v.get('received') and v.get('completed') and start<=stamp(v['received']) and stamp(v['completed'])<=end]
        window_audit=[v for v in audit if v.get('received') and v.get('completed') and start<=stamp(v['received']) and stamp(v['completed'])<=end]
        if len(selected)!=len(window_audit):incomplete.append('window_audit_http_join_incomplete')
        if not selected:incomplete.append('no_actual_kfp_request_classification')
        api_records=[v for v in selected if v['user']==before['api_user']]
        expected_priority=load(c['observer_template'])['priority_uid']
        api_shared=bool(api_records) and current['api_user']==before['api_user'] and before['policies']==current['policies'] and all(v['priority_uid']==expected_priority and v['priority_type']=='Limited' for v in api_records)
        coverage=sorted({(v['user'],v['group'],v['resource'],v['verb']) for v in selected})
        # Rotation/global audit completeness is not assumed from matching retained
        # rows. This artifact proves individual requests, not every tuned path.
        result.update(actual_api_user=before['api_user'],actual_api_record_count=len(api_records),actual_api_shared_priority_verified=api_shared,expected_priority_uid=expected_priority,records=selected,window_audit_records=len(window_audit),window_join_complete=bool(selected) and len(selected)==len(window_audit),coverage=coverage,projection_counts=projected,raw_bytes_processed=budget[0],missing=incomplete,actual_kfp_records_proven=bool(selected),global_capture_complete=False,acceptance_complete=False,logging_overhead_measured=False)
        save(out/'classification.json',result)
    except Exception as error:
        # Optional evidence collection must never abort the actual measurement;
        # no arbitrary exception text, subprocess stdout or secrets are retained.
        incomplete.append(type(error).__name__)
        save(out/'classification.json',{'missing':incomplete,'actual_kfp_records_proven':False,'acceptance_complete':False,'capture_failed':True})
if __name__=='__main__':main()
