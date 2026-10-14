#!/usr/bin/env python3
"""Disposable image-import diagnostic; never starts application workloads."""
import argparse, hashlib, json, re, subprocess, tarfile
from pathlib import Path

DIGEST = re.compile(r'sha256:[0-9a-f]{64}')

def digest(raw):
    return 'sha256:' + hashlib.sha256(raw).hexdigest()

def project(raw):
    obj = json.loads(raw)
    result = {'digest': digest(raw), 'bytes': len(raw), 'keys': sorted(obj)}
    # Never emit Env, Cmd, labels, history or annotations; hash their exact values.
    if isinstance(obj.get('rootfs'), dict):
        result['kind'] = 'config'
        result['architecture'] = obj.get('architecture')
        result['os'] = obj.get('os')
        result['diff_ids'] = obj['rootfs'].get('diff_ids', [])
        result['fields_sha256'] = {k: digest(json.dumps(v, sort_keys=True, separators=(',', ':')).encode()) for k,v in obj.items()}
        result['runtime_fields_sha256'] = {k: digest(json.dumps(v, sort_keys=True, separators=(',', ':')).encode()) for k,v in obj.get('config', {}).items()}
    else:
        result['kind'] = 'index' if 'manifests' in obj else 'manifest' if 'config' in obj else 'unknown'
        result['config_digest'] = obj.get('config', {}).get('digest')
        result['children'] = sorted({m['digest'] for m in obj.get('manifests', []) if DIGEST.fullmatch(m.get('digest', ''))})
    return result

def run(args, limit=4 << 20):
    p = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=180)
    if p.returncode or len(p.stdout) > limit:
        raise RuntimeError('diagnostic command failed: ' + args[0] + ' exit=' + str(p.returncode))
    return p.stdout

def main():
    p=argparse.ArgumentParser();p.add_argument('--images',required=True);p.add_argument('--out',required=True);p.add_argument('--cluster',required=True);a=p.parse_args()
    if not re.fullmatch(r'kfp-apf-diag-[0-9-]+',a.cluster):raise ValueError('owned cluster name required')
    images=json.loads(Path(a.images).read_text()); image=images.get('images',images)['apiserver']
    out=Path(a.out);out.mkdir(parents=True,exist_ok=False)
    result={'expected_config':image['image_id'], 'stages':{}, 'acceptance_complete':False}
    def save(): (out/'image-import.json').write_text(json.dumps(result,indent=2)+'\n')
    with tarfile.open(image['archive']) as archive:
        manifest=json.load(archive.extractfile('manifest.json'))
        raw=archive.extractfile(manifest[0]['Config']).read(4<<20)
    if digest(raw)!=image['image_id']:raise RuntimeError('archive config mismatch')
    result['archive']=project(raw);save()
    run(['docker','load','-i',image['archive']]);ref='kfp-apf-diagnostic/apiserver:verified'
    before=json.loads(run(['docker','image','inspect',image['image_id']]))[0]
    if before['Id']!=image['image_id']:raise RuntimeError('Docker identity mismatch')
    result['stages']['docker']={'id':before['Id'],'rootfs':before.get('RootFS')};save()
    run(['docker','tag',image['image_id'],ref])
    created=False
    try:
        run(['kind','create','cluster','--name',a.cluster,'--image','kindest/node@sha256:21c46cf61fd45873f89e6a1bfcba4b7904dffa84c2bec88aeeca9a0409af4725','--wait','120s']);created=True
        run(['kind','load','docker-image','--name',a.cluster,ref])
        node=a.cluster+'-control-plane'
        cri=json.loads(run(['docker','exec',node,'crictl','inspecti',ref]))
        image_id=cri['status']['id'];result['stages']['cri']={'id':image_id};save()
        if not DIGEST.fullmatch(image_id):raise RuntimeError('CRI identity lacks digest')
        content=run(['docker','exec',node,'ctr','--namespace','k8s.io','content','get',image_id])
        if digest(content)!=image_id:raise RuntimeError('containerd content hash mismatch')
        result['containerd']=project(content)
        result['equal_json']=json.loads(raw)==json.loads(content)
        result['equal_rootfs']=json.loads(raw).get('rootfs')==json.loads(content).get('rootfs')
        result['equal_runtime_config']=json.loads(raw).get('config')==json.loads(content).get('config')
        save()
    finally:
        if created:run(['kind','delete','cluster','--name',a.cluster])
if __name__=='__main__':main()
