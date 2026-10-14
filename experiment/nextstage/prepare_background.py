"""Create a distinct, namespace-scoped pressure identity in an owned cluster."""
import argparse,base64,hashlib,json,os,ssl,sys,urllib.request
from pathlib import Path

def resources(namespace='apf-background'):
    return [
        {'apiVersion':'v1','kind':'Namespace','metadata':{'name':namespace}},
        {'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':'pressure','namespace':namespace}},
        {'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':'probe','namespace':namespace},'data':{'fixed':'x'*1024}},
        {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':'pressure','namespace':namespace},'rules':[{'apiGroups':[''],'resources':['configmaps'],'resourceNames':['probe'],'verbs':['get','update']},{'apiGroups':[''],'resources':['configmaps'],'verbs':['list']}]},
        {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':'pressure','namespace':namespace},'subjects':[{'kind':'ServiceAccount','name':'pressure','namespace':namespace}],'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':'pressure'}}]

BULK_COUNT=1024
PAYLOAD='x'*1024
BULK_LABEL={'apf-experiment':'bulk-v2'}

def bulk_resources(namespace='apf-background'):
    return [{'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':f'bulk-{i:04d}','namespace':namespace,'labels':BULK_LABEL},'data':{'fixed':PAYLOAD}} for i in range(BULK_COUNT)]


def verify_seed(items):
    expected={f'bulk-{i:04d}' for i in range(BULK_COUNT)}
    names=[v['metadata']['name'] for v in items]
    if len(names)!=BULK_COUNT or set(names)!=expected:raise RuntimeError('bulk seed count or names differ')
    for item in items:
        if item.get('data')!={'fixed':PAYLOAD} or item.get('binaryData'):raise RuntimeError('bulk seed payload differs')
    return {'bulk_object_count':BULK_COUNT,'bulk_name_prefix':'bulk-','bytes_per_object':1024,'total_payload_bytes':BULK_COUNT*1024,
            'payload_sha256':hashlib.sha256(PAYLOAD.encode()).hexdigest(),'probe_excluded_from_bulk_count':True,'background_bulk_list':True}


def read_seed(owner,namespace):
    # The fixed 1MiB payload plus JSON/object metadata exceeds the usual 1MiB
    # command limit. Keep only this seed read at an explicit finite 4MiB bound.
    actual=json.loads(owner.k('-n',namespace,'get','configmaps','-l','apf-experiment=bulk-v2','-o','json',limit=4<<20))
    return verify_seed(actual['items'])


def verify_list(listed):
    items=listed.get('items',[])
    if not 1025<=len(items)<=2048 or listed.get('metadata',{}).get('continue'):
        raise RuntimeError('bulk LIST was incomplete or outside item bound')
    verify_seed([v for v in items if v.get('metadata',{}).get('labels',{}).get('apf-experiment')=='bulk-v2'])
    return len(items)


def classification(flow_uid,priority_uid,schemas,priorities,quiet):
    if not flow_uid or not priority_uid:raise RuntimeError('background APF response headers missing')
    flow=next((v for v in schemas['items'] if v['metadata']['uid']==flow_uid),None)
    priority=next((v for v in priorities['items'] if v['metadata']['uid']==priority_uid),None)
    if not flow or not priority or priority['spec']['type']!='Limited':raise RuntimeError('background is not observed nonexempt Limited traffic')
    if flow['spec']['priorityLevelConfiguration']['name']!=priority['metadata']['name']:raise RuntimeError('observed APF policy mapping differs')
    if priority_uid!=quiet['priority_uid']:raise RuntimeError('background and quiet competitor do not share priority')
    return {'background_flow_uid':flow_uid,'priority_uid':priority_uid,'actual_background_headers_verified':True,'quiet_shared_priority_verified':True,'actual_kfp_classification_verified':False}

def main():
    p=argparse.ArgumentParser();p.add_argument('--coordinator-dir',required=True);p.add_argument('--config',required=True);p.add_argument('--quiet-template',required=True);p.add_argument('--out',required=True);a=p.parse_args();os.umask(0o077)
    sys.path.insert(0,a.coordinator_dir)
    from coordinator import Owned,load,save,NoRedirect
    c=load(a.config);owner=Owned(c);uid=owner.guard();quiet=load(a.quiet_template)
    if quiet['rps']!=20 or quiet['max_inflight']>64:raise RuntimeError('quiet competitor must remain fixed20 RPS and bounded64 workers')
    out=Path(a.out);out.mkdir(mode=0o700);ns='apf-background'
    if any(v['metadata']['name']==ns for v in json.loads(owner.k('get','namespaces','-o','json'))['items']):raise RuntimeError('background namespace already exists; refusing overwrite')
    manifest=out/'resources.json';save(manifest,{'apiVersion':'v1','kind':'List','items':resources(ns)});owner.k('apply','-f',str(manifest))
    # Eight bounded batches; namespace is newly owned and was refused if preexisting.
    seeded=bulk_resources(ns)
    for start in range(0,BULK_COUNT,128):
        batch=out/f'bulk-{start:04d}.json';save(batch,{'apiVersion':'v1','kind':'List','items':seeded[start:start+128]});owner.k('apply','-f',str(batch))
    seed_proof=read_seed(owner,ns);save(out/'bulk-seed.json',seed_proof)
    token=owner.k('-n',ns,'create','token','pressure','--duration=3h').decode().strip();tokenfile=out/'background.token';tokenfile.write_text(token);tokenfile.chmod(0o600)
    kube=json.loads(owner.k('config','view','--minify','--raw','-o','json'))['clusters'][0]['cluster']
    ca=ssl.create_default_context(cadata=base64.b64decode(kube['certificate-authority-data']).decode())
    opener=urllib.request.build_opener(urllib.request.HTTPSHandler(context=ca),NoRedirect())
    request=urllib.request.Request(kube['server']+'/api/v1/namespaces/'+ns+'/configmaps/probe',headers={'Authorization':'Bearer '+token})
    with opener.open(request,timeout=10) as response:flow=response.headers.get('X-Kubernetes-PF-FlowSchema-UID');priority=response.headers.get('X-Kubernetes-PF-PriorityLevel-UID')
    schemas=json.loads(owner.k('get','flowschemas','-o','json'));priorities=json.loads(owner.k('get','prioritylevelconfigurations','-o','json'))
    proof=classification(flow,priority,schemas,priorities,quiet)
    list_request=urllib.request.Request(kube['server']+'/api/v1/namespaces/'+ns+'/configmaps?limit=2048',headers={'Authorization':'Bearer '+token})
    with opener.open(list_request,timeout=10) as response:
        list_proof=classification(response.headers.get('X-Kubernetes-PF-FlowSchema-UID'),response.headers.get('X-Kubernetes-PF-PriorityLevel-UID'),schemas,priorities,quiet)
        body=response.read((4<<20)+1)
        if len(body)>4<<20:raise RuntimeError('bulk LIST response bound exceeded')
        listed=json.loads(body)
    item_count=verify_list(listed)
    proof.update(bulk_list_headers_verified=True,bulk_list_flow_uid=list_proof['background_flow_uid'],bulk_list_page_count=item_count,bulk_list_response_bytes=len(body),bulk_list_has_continuation=bool(listed.get('metadata',{}).get('continue')))
    save(out/'classification.json',proof)
    save(out/'stock-policies.json',{'flowschemas':schemas,'priorities':priorities})
    template=dict(quiet,namespace=ns,configmap='probe',token_file=str(tokenfile.resolve()),competitor_flow_uid=flow,priority_uid=priority,cluster_uid=uid,background_bulk_list=True)
    save(out/'background-template.json',template)
if __name__=='__main__':main()
