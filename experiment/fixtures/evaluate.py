"""Strict checks on captured GetRun JSON and pre-captured artifact bytes; no I/O to cluster."""
import argparse,json,re
from pathlib import Path

def evaluate(name, run, artifact_bytes=None):
    errors=[]
    def check(ok,message):
        if not ok: errors.append(message)
    check(run.get('state')=='SUCCEEDED','run must succeed')
    tasks=run.get('tasks',[])
    # Executed leaves only: reject caching as evidence of actual executor traffic.
    leaves=[t for t in tasks if t.get('type')=='RUNTIME']
    check(bool(leaves),'no container task coverage')
    for t in leaves:
        if t.get('state')=='SKIPPED':continue
        check(t.get('state')=='SUCCEEDED','non-success leaf '+t.get('display_name',''))
        check(any(p.get('type')=='EXECUTOR' for p in t.get('pods',[])),'no executor '+t.get('display_name',''))
    active=[t for t in leaves if t.get('state')=='SUCCEEDED']
    def matching(prefix):return [t for t in active if re.fullmatch(re.escape(prefix)+r'(?:-\d+)?',t.get('display_name',''))]
    def param(t,direction,key):
        vals=[p.get('value') for p in (t.get(direction) or {}).get('parameters',[]) if p.get('parameter_key')==key]
        return vals[0] if len(vals)==1 else None
    if name=='flip_coin':
        flip=matching('flip-coin'); rand=matching('random-num'); msgs=matching('print-msg')
        check(len(flip)==len(rand)==len(msgs)==1,'expected one coin, random number, message')
        if len(flip)==len(rand)==len(msgs)==1:
            coin=param(flip[0],'outputs','Output'); n=param(rand[0],'outputs','Output')
            check(coin in ['heads','tails'],'invalid coin')
            numeric=isinstance(n,(int,float)) and not isinstance(n,bool) and int(n)==n
            check(numeric,'random output must be integer')
            if numeric and coin in ['heads','tails']:
                low,high,threshold=(0,9,5) if coin=='heads' else (10,19,15)
                check(low<=n<=high,'wrong branch random range')
                check(param(rand[0],'inputs','low')==low and param(rand[0],'inputs','high')==high,'wrong random inputs')
                check(param(msgs[0],'inputs','msg')==f'{coin} and {int(n)} {">" if n>threshold else "<="} {threshold}!','message/branch mismatch')
    elif name=='collected_parameters':
        prep=matching('prepend-id'); single=matching('consume-single-id'); consumers=matching('consume-ids'); split=matching('split-ids')
        check((len(prep),len(single),len(consumers),len(split))==(3,3,2,1),'missing loop/nested consumer coverage')
        expected=['model_id_s1','model_id_s2','model_id_s3']
        check(sorted(str(param(t,'outputs','Output')) for t in prep)==expected,'incorrect collected outputs')
        check(sorted(str(param(t,'inputs','id')) for t in single)==expected,'incorrect single consumer inputs')
        for t in single+consumers:check(param(t,'outputs','Output')=='completed','consumer output mismatch')
        for t in consumers:check(param(t,'inputs','ids')==expected,'collected propagation/order mismatch')
        if len(split)==1:check(param(split[0],'outputs','Output')==['s1','s2','s3'],'split output mismatch')
    elif name=='if_else_with_oneof_artifacts':
        coins=matching('flip-coin'); producers=matching('param-to-artifact'); consumers=matching('print-artifact')
        check((len(coins),len(producers),len(consumers))==(1,1,2),'missing oneof/nested artifact coverage')
        if len(coins)==1:check(param(coins[0],'outputs','Output')=='heads','deterministic coin mismatch')
        def artifacts(t,direction):return [a for group in (t.get(direction) or {}).get('artifacts',[]) for a in group.get('artifacts',[])]
        if len(producers)==1:
            check(param(producers[0],'inputs','val')=='heads','artifact source mismatch')
            produced=artifacts(producers[0],'outputs'); check(len(produced)==1,'missing produced artifact')
            if len(produced)==1:
                a=produced[0]; aid=a.get('artifact_id'); check(bool(aid) and bool(a.get('uri')),'missing artifact identity/URI')
                for t in consumers:
                    got=artifacts(t,'inputs');check(len(got)==1 and got[0].get('artifact_id')==aid and got[0].get('uri')==a.get('uri'),'oneof artifact identity not propagated')
                check(artifact_bytes is not None and artifact_bytes.get(aid)==b'heads','missing/incorrect captured artifact bytes')
    else:errors.append('unknown fixture')
    return {'fixture':name,'run_id':run.get('run_id'),'passed':not errors,'errors':errors,'executed_leaf_count':len(active)}

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('fixture');p.add_argument('run_json');p.add_argument('--artifact-index',help='JSON artifact ID to captured local filename, relative to index directory')
    a=p.parse_args(); data=None
    if a.artifact_index:
        index=Path(a.artifact_index);data={k:(index.parent/v).read_bytes() for k,v in json.loads(index.read_text()).items()}
    result=evaluate(a.fixture,json.loads(Path(a.run_json).read_text()),data);print(json.dumps(result,indent=2));raise SystemExit(not result['passed'])
