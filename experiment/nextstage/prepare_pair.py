#!/usr/bin/env python3
"""Prepare real least-privilege competitor and fixed exploratory offered rates."""
import argparse,base64,json,ssl,tempfile,urllib.request,os,hashlib,re
from pathlib import Path
from coordinator import Owned,load,save,bounded,NoRedirect

def bind_runtime_image(image_id, expected, read_content):
    """CRI may expose an OCI manifest ID, while Docker save pins config ID."""
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', expected):
        raise RuntimeError('invalid pinned config digest')
    digest = image_id.rsplit('@', 1)[-1]
    if digest.startswith('containerd://'):
        digest = digest[len('containerd://'):]
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest):
        raise RuntimeError('runtime image ID has no immutable digest')
    proof = {'runtime_image_id': image_id, 'config_digest': expected, 'manifest_chain': []}
    if digest == expected:
        return proof
    for _ in range(2):
        raw = read_content(digest)
        if len(raw) > 4 << 20 or 'sha256:' + hashlib.sha256(raw).hexdigest() != digest:
            raise RuntimeError('containerd manifest bytes do not match runtime digest')
        manifest = json.loads(raw)
        proof['manifest_chain'].append(digest)
        if manifest.get('config', {}).get('digest') == expected:
            return proof
        candidates = [m for m in manifest.get('manifests', []) if m.get('platform', {}).get('os') == 'linux' and m.get('platform', {}).get('architecture') == 'amd64']
        if len(candidates) != 1 or not re.fullmatch(r'sha256:[0-9a-f]{64}', candidates[0].get('digest', '')):
            break
        digest = candidates[0]['digest']
    raise RuntimeError('runtime image does not bind to verified config digest')


def calibration_request(request, ca):
    opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ca), NoRedirect())
    return opener.open(request, timeout=10)


def main():
    p=argparse.ArgumentParser();p.add_argument('--provision-config',required=True);p.add_argument('--state',required=True);p.add_argument('--bundle',required=True);p.add_argument('--out',required=True);a=p.parse_args();os.umask(0o077)
    out=Path(a.out).resolve();out.mkdir(mode=0o700);c=load(a.provision_config);c['state']=a.state;owner=Owned(c);uid=owner.guard();ns='apf-competitor'
    objects=[{'apiVersion':'v1','kind':'Namespace','metadata':{'name':ns}}, {'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':'probe','namespace':ns}}, {'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':'probe','namespace':ns},'data':{'fixed':'x'*1024}}, {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':'probe','namespace':ns},'rules':[{'apiGroups':[''],'resources':['configmaps'],'verbs':['get','list','update']}]}, {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':'probe','namespace':ns},'subjects':[{'kind':'ServiceAccount','name':'probe','namespace':ns}],'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':'probe'}}]
    manifest=out/'competitor-resources.json';save(manifest,{'apiVersion':'v1','kind':'List','items':objects});owner.k('apply','-f',str(manifest))
    token=owner.k('-n',ns,'create','token','probe','--duration=3h').decode().strip();tokenpath=out/'competitor.token';tokenpath.write_text(token);tokenpath.chmod(0o600)
    kube=json.loads(owner.k('config','view','--minify','--raw','-o','json'))['clusters'][0]['cluster'];ca=ssl.create_default_context(cadata=base64.b64decode(kube['certificate-authority-data']).decode())
    request=urllib.request.Request(kube['server']+'/api/v1/namespaces/'+ns+'/configmaps/probe',headers={'Authorization':'Bearer '+token})
    with calibration_request(request,ca) as response:
        fsuid=response.headers.get('X-Kubernetes-PF-FlowSchema-UID');pluid=response.headers.get('X-Kubernetes-PF-PriorityLevel-UID')
    schemas=json.loads(owner.k('get','flowschemas','-o','json'));priorities=json.loads(owner.k('get','prioritylevelconfigurations','-o','json'))
    save(out/'schemas.json',schemas);save(out/'priorities.json',priorities)
    fs=next(x for x in schemas['items'] if x['metadata']['uid']==fsuid);pl=next(x for x in priorities['items'] if x['metadata']['uid']==pluid)
    if pl['spec']['type']!='Limited' or fs['spec']['priorityLevelConfiguration']['name']!=pl['metadata']['name']:raise RuntimeError('competitor is not observed Limited priority')
    save(out/'classification-calibration.json',{'competitor_flow_uid':fsuid,'priority_uid':pluid,'actual_competitor_headers':True,'actual_kfp_classification_verified':False,'kfp_mapping_assumption':'serviceaccount bootstrap policy; must independently verify actual KFP audit observations'})
    # KFP identity is a serviceaccount; this candidate UID is not proof of actual KFP classification.
    workload=load(Path(a.state)/'workload-config-skeleton.json');workload.update(probe_seconds=60,drain_seconds=600,max_inflight_per_tenant=16)
    workload['phases']=[{'name':name,'seconds':seconds,'quiet_rps':2,'noisy_rps':rate} for name,seconds,rate in [('normal',120,2),('sustained',180,100),('burst1',10,300),('rest1',30,2),('burst2',10,300),('rest2',30,2),('burst3',10,300),('recovery',120,2)]]
    for tenant in workload['tenants']:
        fresh=owner.k('-n',tenant['namespace'],'create','token',tenant['service_account'],'--audience=pipelines.kubeflow.org','--duration=3h').decode().strip()
        Path(tenant['token_file']).write_text(fresh)
    save(out/'workload.json',workload)
    observer={'kubeconfig':owner.kubeconfig,'context':owner.context,'cluster_uid':uid,'namespace':ns,'configmap':'probe','token_file':str(tokenpath),'kfp_flow_uid':fsuid,'competitor_flow_uid':fsuid,'priority_uid':pluid,'rps':20,'max_inflight':64}
    save(out/'observer.json',observer)
    bundle=Path(a.bundle).resolve();c.update(api_image='kfp-apf-local/apiserver:'+c['source_sha'],api_image_ids=['docker-pullable://kfp-apf-local/apiserver@'+c['images']['apiserver']['image_id'],c['images']['apiserver']['image_id']],workload_template=str(out/'workload.json'),observer_template=str(out/'observer.json'),workload_binary=str(bundle/'bin/workload'),observer_binary=str(bundle/'bin/observer'),scheduled_binary=str(bundle/'bin/scheduled'),fixture_evaluator=str(bundle/'fixtures/evaluate.py'),measurement_only=True)
    pods=json.loads(owner.k('-n','kubeflow','get','pods','-l','app=ml-pipeline','-o','json'))
    expected=c['images']['apiserver']['image_id']
    actual=[];bindings=[]
    for pod in pods['items']:
        for status in pod.get('status',{}).get('containerStatuses',[]):
            if status['name']!='ml-pipeline-api-server' or not status.get('ready'):
                continue
            node=pod['spec']['nodeName']
            if node!=c['cluster']+'-control-plane':
                raise RuntimeError('runtime image inspection requires owned Kind control-plane node')
            def content(digest):
                return bounded(['docker','exec',node,'ctr','--namespace','k8s.io','content','get',digest],timeout=15,limit=4<<20)
            bindings.append(bind_runtime_image(status['imageID'],expected,content))
            actual.append(status['imageID'])
    if not actual:raise RuntimeError('no ready API container identity')
    c['api_image_ids']=sorted(set(actual))
    c['api_image_bindings']=bindings
    save(out/'runtime-image-bindings.json',bindings)
    save(out/'coordinator.json',c)
if __name__=='__main__':main()
