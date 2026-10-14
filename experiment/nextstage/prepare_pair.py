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
    proof = {'runtime_image_id': image_id, 'runtime_digest': digest, 'config_digest': expected, 'manifest_chain': [], 'observed_config_digests': [], 'candidate_manifest_digests': []}
    if digest == expected:
        return proof
    for _ in range(3):
        raw = read_content(digest)
        if len(raw) > 4 << 20 or 'sha256:' + hashlib.sha256(raw).hexdigest() != digest:
            raise RuntimeError('containerd manifest bytes do not match runtime digest')
        manifest = json.loads(raw)
        proof['manifest_chain'].append(digest)
        observed_config = manifest.get('config', {}).get('digest')
        if isinstance(observed_config, str) and re.fullmatch(r'sha256:[0-9a-f]{64}', observed_config):
            proof['observed_config_digests'].append(observed_config)
        if observed_config == expected:
            return proof
        candidates = [m for m in manifest.get('manifests', []) if not m.get('platform') or (m['platform'].get('os') == 'linux' and m['platform'].get('architecture') == 'amd64')]
        # An omitted platform does not weaken identity: the child chain must
        # still bind the exact verified amd64 config, and CRI must agree.
        # Docker save can list the same manifest once per tag (latest/ci/local).
        # Distinct descriptors are not distinct images when their digests match.
        candidate_digests = sorted({m.get('digest', '') for m in candidates})
        proof['candidate_manifest_digests'].append([d for d in candidate_digests if re.fullmatch(r'sha256:[0-9a-f]{64}', d)])
        if len(candidate_digests) != 1 or not re.fullmatch(r'sha256:[0-9a-f]{64}', candidate_digests[0]):
            break
        digest = candidate_digests[0]
    # Digest-only diagnostics survive hosted failure without leaking manifest
    # annotations, credentials, labels or raw config contents.
    diagnosis = {key: value for key, value in proof.items() if key != 'runtime_image_id'}
    raise RuntimeError('runtime image does not bind to verified config digest: ' + json.dumps(diagnosis, sort_keys=True))


def bind_cri_runtime(image_id, expected, inspect_image, read_content):
    cri = inspect_image(image_id)
    if cri.get('status', {}).get('id') != expected:
        raise RuntimeError('CRI selected config does not match verified config digest')
    proof = bind_runtime_image(image_id, expected, read_content)
    proof['cri_config_digest'] = expected
    return proof


def immutable_digest(value):
    value=value.rsplit('@',1)[-1]
    for prefix in ('containerd://','docker-pullable://','docker://'):
        if value.startswith(prefix): value=value[len(prefix):]
    if not re.fullmatch(r'sha256:[0-9a-f]{64}',value):
        raise RuntimeError('runtime reference lacks immutable digest')
    return value


def bind_actual_container(status, expected, inspect_container, inspect_image, read_content, emit):
    pod_digest=immutable_digest(status['imageID'])
    container_id=status.get('containerID','')
    if not re.fullmatch(r'containerd://[0-9a-f]{64}',container_id):
        raise RuntimeError('unsupported container runtime identity')
    cid=container_id.removeprefix('containerd://')
    evidence={'pod_image_digest':pod_digest,'container_id':cid,'expected_config_digest':expected,'content':[],'stage':'inspect_container'}
    emit(evidence)
    def capture_content(d):
        raw=read_content(d)
        if len(raw)>4<<20 or 'sha256:'+hashlib.sha256(raw).hexdigest()!=d:
            raise RuntimeError('containerd content digest mismatch')
        obj=json.loads(raw)
        descriptor={'digest':d,'kind':'index' if 'manifests' in obj else 'config' if 'rootfs' in obj else 'manifest'}
        cfg=obj.get('config',{})
        if isinstance(cfg,dict) and re.fullmatch(r'sha256:[0-9a-f]{64}',cfg.get('digest','')):descriptor['config_digest']=cfg['digest']
        descriptor['children']=[{'digest':m.get('digest'),'platform':{k:m.get('platform',{}).get(k) for k in ('os','architecture')}} for m in obj.get('manifests',[])[:8] if re.fullmatch(r'sha256:[0-9a-f]{64}',m.get('digest',''))]
        evidence['content'].append(descriptor);emit(evidence)
        return raw
    # Capture the pod's exact content chain before any CRI lookup can fail.
    capture_content(pod_digest)
    container=inspect_container(cid).get('status',{})
    actual_ref=container.get('imageRef','')
    actual_digest=immutable_digest(actual_ref)
    evidence.update(cri_container_image_digest=actual_digest,stage='inspect_image');emit(evidence)
    # The actual container must identify the same immutable image or config.
    if actual_digest not in (pod_digest,expected):
        raise RuntimeError('container and pod immutable image identities differ')
    # Container imageRef identifies an OCI index; CRI ImageStatus is indexed
    # by config ID. Establish the cryptographic runtime chain first, then query
    # its verified config ID (the successful import diagnostic exercises this).
    proof=bind_runtime_image(pod_digest,expected,capture_content)
    evidence.update(stage='inspect_verified_config',cri_lookup_digest=proof['config_digest']);emit(evidence)
    image=inspect_image(proof['config_digest'])
    config=image.get('status',{}).get('id','')
    if re.fullmatch(r'sha256:[0-9a-f]{64}',config):evidence['cri_config_digest']=config
    emit(evidence)
    if config!=expected:raise RuntimeError('CRI selected config does not match verified config digest')
    evidence['stage']='verified';emit(evidence)
    proof['cri_config_digest']=config
    return proof


def calibration_request(request, ca):
    opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ca), NoRedirect())
    return opener.open(request, timeout=10)


def main():
    p=argparse.ArgumentParser();p.add_argument('--provision-config',required=True);p.add_argument('--state',required=True);p.add_argument('--bundle',required=True);p.add_argument('--out',required=True);p.add_argument('--evidence-out',required=True);a=p.parse_args();os.umask(0o077)
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
            def inspect_image(image_id):
                return json.loads(bounded(['docker','exec',node,'crictl','inspecti',image_id],timeout=15,limit=4<<20))
            def inspect_container(container_id):
                return json.loads(bounded(['docker','exec',node,'crictl','inspect',container_id],timeout=15,limit=4<<20))
            evidence_path=Path(a.evidence_out)/('runtime-'+pod['metadata']['uid']+'.json')
            evidence_path.parent.mkdir(parents=True,exist_ok=True)
            bindings.append(bind_actual_container(status,expected,inspect_container,inspect_image,content,lambda record:save(evidence_path,record)))
            actual.append(status['imageID'])
    if not actual:raise RuntimeError('no ready API container identity')
    c['api_image_ids']=sorted(set(actual))
    c['api_image_bindings']=bindings
    save(out/'runtime-image-bindings.json',bindings)
    save(out/'coordinator.json',c)
if __name__=='__main__':main()
