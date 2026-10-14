"""Real bounded collection, separate from pure acceptance decisions."""
import importlib.util,json,time,urllib.request
from pathlib import Path
from coordinator import save,load,bounded,read_events
from acceptance import scheduled

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args,**kwargs):raise RuntimeError('redirect refused')

def get(endpoint,path,token):
    request=urllib.request.Request(endpoint+path,headers={'Authorization':'Bearer '+token})
    with urllib.request.build_opener(NoRedirect()).open(request,timeout=10) as response:data=response.read((8<<20)+1)
    if len(data)>8<<20:raise RuntimeError('GetRun response bound')
    return json.loads(data)

def scheduled_stage(c,workload,directory,name):
    config=dict(workload,label=workload['label']+'-'+name)
    path=directory/(name+'-config.json');output=directory/(name+'.jsonl');save(path,config)
    # Stage is outside the shared measurement timeline. No write retry.
    bounded([c['scheduled_binary'],'-config',str(path),'-out',str(output),'-seconds','300'],timeout=1300)
    rows=list(read_events(output));passed=scheduled(rows,[t['name'] for t in workload['tenants']])
    save(directory/(name+'-result.json'),{'passed':passed,'outside_measurement_window':True})
    if not passed:raise RuntimeError('recurring lifecycle incomplete')

HEALTH_TEMPLATE = ('{{range .items}}{{if or (eq .metadata.namespace "kube-system") (eq .metadata.namespace "kubeflow")}}'
    '{{$pod := .}}{{$uid := .metadata.uid}}{{range .status.containerStatuses}}'
    '{"key":{{printf "%q" (printf "%s/%s" $uid .name)}},'
    '"pod_phase":{{printf "%q" $pod.status.phase}},"controllers":['
    '{{$sep := ""}}{{range $pod.metadata.ownerReferences}}{{if .controller}}{{$sep}}'
    '{"api_version":{{printf "%q" .apiVersion}},"kind":{{printf "%q" .kind}},"uid":{{printf "%q" .uid}}}{{$sep = ","}}{{end}}{{end}}],'
    '"state":{{if .state.running}}"running"{{else if .state.terminated}}"terminated"{{else}}"other"{{end}},'
    '"exit_code":{{if .state.terminated}}{{.state.terminated.exitCode}}{{else}}null{{end}},'
    '"termination_reason":{{if .state.terminated}}{{printf "%q" .state.terminated.reason}}{{else}}null{{end}},'
    '"restart_count":{{.restartCount}},"ready":{{.ready}},"reason":'
    '{{if .lastState.terminated}}{{printf "%q" .lastState.terminated.reason}}{{else}}null{{end}}}'
    '{{"\\n"}}{{end}}{{end}}{{end}}')

def health_snapshot(owner):
    # kubectl projects before stdout capture: Pod args/env/spec never enter evidence.
    data=owner.k('get','pods','-A','-o','go-template','--template='+HEALTH_TEMPLATE,timeout=30)
    result={}
    for line in data.decode().splitlines():
        if not line.strip():continue
        row=json.loads(line);key=row.pop('key')
        if key in result:raise RuntimeError('duplicate health container identity')
        if not isinstance(row['ready'],bool) or not isinstance(row['restart_count'],int):raise RuntimeError('invalid projected container health')
        result[key]=row
    if not result:raise RuntimeError('empty projected container health')
    return result

def collect_semantics(c,workload,directory):
    spec=importlib.util.spec_from_file_location('fixture_evaluator',c['fixture_evaluator']);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    runs=load(directory/'workload'/'results.json',32<<20)['runs'];out=directory/'semantic';out.mkdir(mode=0o700);results=[]
    # Hard aggregate budget; missing late records remain explicit failures.
    deadline=time.monotonic()+180
    for tenant in workload['tenants']:
        token=Path(tenant['token_file']).read_text().strip()
        for run in runs.get(tenant['name'],[]):
            if time.monotonic()>=deadline:raise RuntimeError('semantic collection deadline')
            rid=run['id']
            if not rid or not all(ch.isalnum() or ch=='-' for ch in rid):raise RuntimeError('unsafe run ID')
            raw=get(workload['endpoint'],'/apis/v2beta1/runs/'+rid+'?view=FULL',token);save(out/(rid+'.json'),raw)
            artifact_bytes=None
            if run['fixture']=='if_else_with_oneof_artifacts' and c.get('artifact_gateway'):
                helper_spec=importlib.util.spec_from_file_location('owner_artifact_capture',c['artifact_capture_helper'])
                helper=importlib.util.module_from_spec(helper_spec);helper_spec.loader.exec_module(helper)
                ids={a['artifact_id'] for task in raw.get('tasks',[]) for group in (task.get('outputs') or {}).get('artifacts',[]) for a in group.get('artifacts',[]) if a.get('artifact_id')}
                if len(ids)!=1:raise RuntimeError('fixed oneof must expose one artifact identity')
                artifact_bytes={}
                for aid in ids:
                    destination=out/(rid+'-artifact')
                    proof=helper.capture(workload['endpoint'],c['artifact_gateway'],token,tenant['namespace'],tenant['experiment_id'],rid,aid,destination)
                    artifact_bytes[aid]=(destination/proof['file']).read_bytes()
            result=module.evaluate(run['fixture'],raw,artifact_bytes=artifact_bytes)
            # No gateway means no bytes and an explicit failed oneof assertion.
            results.append(result);save(out/'results.json',results)
    required={'flip_coin','collected_parameters','if_else_with_oneof_artifacts'}
    passed=bool(results) and required<={r['fixture'] for r in results} and all(r['passed'] for r in results)
    save(out/'summary.json',{'passed':passed,'count':len(results),'artifact_download_integrated':bool(c.get('artifact_gateway'))})
    return passed
