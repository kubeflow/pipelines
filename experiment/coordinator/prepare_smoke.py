#!/usr/bin/env python3
import argparse,json
from pathlib import Path
from coordinator import save
SOURCE='6bea51e2003f9072a55068fb85e03cdb9391435d'
p=argparse.ArgumentParser();p.add_argument('--images',required=True);p.add_argument('--fixtures',required=True);p.add_argument('--source',required=True);p.add_argument('--output',required=True);p.add_argument('--suffix',required=True);a=p.parse_args()
manifest=json.loads((Path(a.fixtures)/'manifest.json').read_text())
if manifest['source_merge']!=SOURCE:raise SystemExit('fixture source mismatch')
images=json.loads(Path(a.images).read_text())
# Downloader returns the component map used by the provisioner.
if 'images' in images:images=images['images']
config={'source_sha':SOURCE,'source_worktree':str(Path(a.source).resolve()),'cluster':'kfp-apf-representative-'+a.suffix,'node_image':'kindest/node@sha256:21c46cf61fd45873f89e6a1bfcba4b7904dffa84c2bec88aeeca9a0409af4725','qps':20,'port':18888,'images':images,'fixtures':{f['name']:{'source_sha':SOURCE,'path':str((Path(a.fixtures)/(f['name']+'.yaml')).resolve()),'sha256':f['upload_sha256']} for f in manifest['fixtures']}}
save(a.output,config)
