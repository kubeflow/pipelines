"""Create only the separately declared positive-seat experiment policy."""
import argparse,json
from pathlib import Path
from coordinator import Owned,load,save,policy_spec_digest
NAME='kfp-experiment-four-seats'
def desired(api_namespace,api_sa):
    if not api_namespace or not api_sa:raise RuntimeError('resolved API service account missing')
    subjects=[{'kind':'ServiceAccount','serviceAccount':{'namespace':ns,'name':sa}} for ns,sa in [(api_namespace,api_sa),('apf-competitor','probe'),('apf-background','pressure')]]
    return {'apiVersion':'v1','kind':'List','items':[
      {'apiVersion':'flowcontrol.apiserver.k8s.io/v1','kind':'PriorityLevelConfiguration','metadata':{'name':NAME},'spec':{'type':'Limited','limited':{'nominalConcurrencyShares':60,'lendablePercent':0,'borrowingLimitPercent':0,'limitResponse':{'type':'Queue','queuing':{'queues':128,'handSize':6,'queueLengthLimit':50}}}}},
      {'apiVersion':'flowcontrol.apiserver.k8s.io/v1','kind':'FlowSchema','metadata':{'name':NAME},'spec':{'priorityLevelConfiguration':{'name':NAME},'matchingPrecedence':100,'distinguisherMethod':{'type':'ByUser'},'rules':[{'subjects':subjects,'resourceRules':[{'verbs':['*'],'apiGroups':['*'],'resources':['*'],'clusterScope':True,'namespaces':['*']}]}]}}]}
def verify_delta(before,after,objects):
    for key,kind in [('flowschemas','FlowSchema'),('priorities','PriorityLevelConfiguration')]:
        old={x['metadata']['name']:x['spec'] for x in before[key]['items']};new={x['metadata']['name']:x['spec'] for x in after[key]['items']}
        wanted=next(x['spec'] for x in objects['items'] if x['kind']==kind)
        if NAME in old or set(new)!=set(old)|{NAME} or any(new[k]!=v for k,v in old.items()) or new[NAME]!=wanted:raise RuntimeError('policy delta differs from exactly two declared experiment resources')
    return True
def snapshot(owner):return {'flowschemas':json.loads(owner.k('get','flowschemas','-o','json')),'priorities':json.loads(owner.k('get','prioritylevelconfigurations','-o','json'))}
def main():
    p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--state',required=True);p.add_argument('--stock-policies',required=True);p.add_argument('--out',required=True);a=p.parse_args()
    c=load(a.config);c['state']=a.state;owner=Owned(c);owner.guard();out=Path(a.out);out.mkdir(mode=0o700,parents=True)
    before=snapshot(owner)
    if policy_spec_digest(before)!=policy_spec_digest(load(a.stock_policies)):raise RuntimeError('bootstrap policy specs differ')
    deployment=json.loads(owner.k('-n','kubeflow','get','deployment','ml-pipeline','-o','json'))
    ns=deployment['metadata']['namespace'];sa=deployment['spec']['template']['spec'].get('serviceAccountName')
    objects=desired(ns,sa);save(out/'requested.json',objects);save(out/'before.json',before)
    owner.k('create','-f',str(out/'requested.json'))
    after=snapshot(owner);verify_delta(before,after,objects);save(out/'expected-policies.json',after)
    save(out/'identity.json',{'api_service_account':'system:serviceaccount:'+ns+':'+sa,'deployment_uid':deployment['metadata']['uid'],'priority_name':NAME,'expected_nominal_and_current_seats':4,'policy_delta_verified':True,'actual_kfp_classification_verified':False})
if __name__=='__main__':main()
