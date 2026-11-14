import collections
import hashlib
import json
import os
import subprocess
from pathlib import Path
from kfpr.osv_image_scan import scan_image, validate_report
from kfpr.osv_reports import findings
from third_party_upgrade_scan import compare_reports

component = os.environ['COMPONENT']
assert component in ('pilot', 'proxyv2', 'install-cni')
output = Path('istio-evidence')
output.mkdir()
reports = []
subjects = []
for version in ('1.31.4', '1.32.0'):
    reference = f'docker.io/istio/{component}:{version}'
    subprocess.run(['docker', 'pull', '--platform', 'linux/amd64', reference], check=True)
    metadata = json.loads(subprocess.check_output(['docker', 'image', 'inspect', reference], text=True))[0]
    assert (metadata['Os'], metadata['Architecture']) == ('linux', 'amd64')
    assert metadata['RepoDigests']
    directory = output / version
    directory.mkdir()
    subject = {'image': reference, 'image_id': metadata['Id'], 'repo_digests': metadata['RepoDigests'], 'platform': 'linux/amd64', 'workflow_sha': os.environ['GITHUB_SHA'], 'run_id': os.environ['GITHUB_RUN_ID'], 'run_attempt': os.environ['GITHUB_RUN_ATTEMPT']}
    (directory / 'subject.json').write_text(json.dumps(subject, indent=2)+'\n')
    scan_image(metadata['Id'], 'linux/amd64', directory, report_format='json')
    envelope = json.loads((directory / 'scan.json').read_text())
    validate_report(envelope, metadata['Id'], 'json')
    reports.append(envelope['report'])
    subjects.append(subject)
summary = {'component': component, 'subjects': subjects, 'comparison': compare_reports(*reports), 'counts': []}
for version, report in zip(('1.31.4', '1.32.0'), reports):
    rows = findings(report)
    summary['counts'].append({'version': version, 'fixable': sum(bool(r['fixable']) for r in rows), 'residual': sum(not r['fixable'] for r in rows), 'fixable_packages': dict(collections.Counter(r['package'] for r in rows if r['fixable']))})
(output / 'comparison.json').write_text(json.dumps(summary, indent=2)+'\n')
checksums = {str(p.relative_to(output)): hashlib.sha256(p.read_bytes()).hexdigest() for p in output.rglob('*') if p.is_file()}
(output / 'checksums.json').write_text(json.dumps(checksums, indent=2)+'\n')
print(json.dumps(summary, indent=2))
