"""Create a distinct, namespace-scoped pressure identity in an owned cluster."""
import argparse,base64,json,os,ssl,sys,urllib.request
from pathlib import Path

def resources(namespace='apf-background'):
    return [
        {'apiVersion':'v1','kind':'Namespace','metadata':{'name':namespace}},
        {'apiVersion':'v1','kind':'ServiceAccount','metadata':{'name':'pressure','namespace':namespace}},
        {'apiVersion':'v1','kind':'ConfigMap','metadata':{'name':'probe','namespace':namespace},'data':{'fixed':'x'*1024}},
        {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'Role','metadata':{'name':'pressure','namespace':namespace},'rules':[{'apiGroups':[''],'resources':['configmaps'],'resourceNames':['probe'],'verbs':['get','list','update']}]},
        {'apiVersion':'rbac.authorization.k8s.io/v1','kind':'RoleBinding','metadata':{'name':'pressure','namespace':namespace},'subjects':[{'kind':'ServiceAccount','name':'pressure','namespace':namespace}],'roleRef':{'apiGroup':'rbac.authorization.k8s.io','kind':'Role','name':'pressure'}}]

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
    token=owner.k('-n',ns,'create','token','pressure','--duration=3h').decode().strip();tokenfile=out/'background.token';tokenfile.write_text(token);tokenfile.chmod(0o600)
    kube=json.loads(owner.k('config','view','--minify','--raw','-o','json'))['clusters'][0]['cluster']
    ca=ssl.create_default_context(cadata=base64.b64decode(kube['certificate-authority-data']).decode())
    opener=urllib.request.build_opener(urllib.request.HTTPSHandler(context=ca),NoRedirect())
    request=urllib.request.Request(kube['server']+'/api/v1/namespaces/'+ns+'/configmaps/probe',headers={'Authorization':'Bearer '+token})
    with opener.open(request,timeout=10) as response:flow=response.headers.get('X-Kubernetes-PF-FlowSchema-UID');priority=response.headers.get('X-Kubernetes-PF-PriorityLevel-UID')
    schemas=json.loads(owner.k('get','flowschemas','-o','json'));priorities=json.loads(owner.k('get','prioritylevelconfigurations','-o','json'))
    proof=classification(flow,priority,schemas,priorities,quiet);save(out/'classification.json',proof)
    save(out/'stock-policies.json',{'flowschemas':schemas,'priorities':priorities})
    template=dict(quiet,namespace=ns,configmap='probe',token_file=str(tokenfile.resolve()),competitor_flow_uid=flow,priority_uid=priority,cluster_uid=uid)
    save(out/'background-template.json',template)
if __name__=='__main__':main()
