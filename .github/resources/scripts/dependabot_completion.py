#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Complete bounded Dependabot proposals using trusted generators.

The generator runs without write credentials. The publisher consumes
data only, revalidates the live proposal and output scope, and uses an
atomic expected-head commit. Neither stage runs scripts from the
proposal.
"""

import argparse
import base64
import difflib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile

import complete_dependency_update as generators
import tomllib

ROOT = Path(__file__).resolve().parents[3]
SCRIPTS = Path(__file__).resolve().parent
PYTHON_SOURCES = {'pyproject.toml', 'sdk/python/pyproject.toml', 'uv.lock'}
PYTHON_EXPORTS = {'requirements.txt', 'sdk/python/requirements.txt'}
GH_AW_SOURCE = '.github/actions/setup-gh-aw/action.yml'
SHA = re.compile(r'[0-9a-f]{40}')
MAX_BUNDLE_BYTES = 10 * 1024 * 1024


def run(*args, cwd=ROOT, **kwargs):
    return subprocess.check_output(args, cwd=cwd, **kwargs)


def git(*args, cwd=ROOT):
    return run('git', '-c', 'core.fsmonitor=false', *args, cwd=cwd)


def api(endpoint, *, paginate=False, body=None):
    args = ['gh', 'api', endpoint]
    if paginate:
        args += ['--paginate', '--slurp']
    if body is not None:
        args += ['--input', '-']
    data = json.loads(
        run(*args,
            input=json.dumps(body).encode() if body is not None else None))
    return [item for page in data for item in page] if paginate else data


def changed_lines(before, after):
    return [
        line[1:]
        for line in difflib.unified_diff(
            before.splitlines(), after.splitlines(), n=0)
        if line[:1] in ('+', '-') and not line.startswith(('+++', '---'))
    ]


def only_lines(before, after, pattern):
    return all(
        re.fullmatch(pattern, line) for line in changed_lines(before, after))


def classify(before, after):
    """Return a completion plan, or None for an unrelated dependency update."""
    paths = set(after)
    if paths <= PYTHON_SOURCES | PYTHON_EXPORTS:
        if 'uv.lock' not in paths:
            if paths & PYTHON_EXPORTS:
                raise ValueError(
                    'Generated-only Python update: rebase/recreate against uv ownership; do not revert its intended bump.'
                )
            return None
        for path in paths & {'pyproject.toml', 'sdk/python/pyproject.toml'}:
            old, new = tomllib.loads(before[path]), tomllib.loads(after[path])
            # A bot may change dependency constraints, not build hooks or sources.
            for value in (old, new):
                project = value.get('project', {})
                project.pop('dependencies', None)
                project.pop('optional-dependencies', None)
            if old != new:
                raise ValueError(
                    f'{path}: dependency update also changes project configuration'
                )
        lock = tomllib.loads(after['uv.lock'])
        if not lock.get('package'):
            raise ValueError('uv.lock contains no packages')
        return {'kind': 'python', 'version': ''}
    go_docker = {str(pin.path) for pin in generators.go.MANAGED_DOCKERFILES}
    module_paths = {
        str(path)
        for path in generators.GO_OUTPUT_PATHS
        if str(path).endswith('go.mod')
    }
    go_change = any(
        re.match(r'(?:FROM golang:|go |toolchain go)', line)
        for path in paths
        for line in changed_lines(before[path], after[path]))
    if paths <= go_docker | module_paths and go_change:
        images, versions = [], set()
        for path in paths:
            if path in go_docker:
                pattern = r'FROM golang:(1\.\d+\.\d+(?:-[a-z0-9.-]+)?)@(sha256:[0-9a-f]{64}) AS [a-z0-9_.-]+'
                if not only_lines(before[path], after[path], pattern):
                    return None
                for line in changed_lines('', after[path]):
                    match = re.fullmatch(pattern, line)
                    if match:
                        images.append(f'{match[1]}@{match[2]}')
                        versions.add(match[1].split('-')[0])
            else:
                if not only_lines(
                        before[path], after[path],
                        r'(?:go 1\.\d+(?:\.\d+)?|toolchain go1\.\d+\.\d+|)'):
                    return None
                compiler = re.findall(r'(?m)^toolchain go(1\.\d+\.\d+)$',
                                      after[path])
                compiler += re.findall(r'(?m)^go (1\.\d+\.\d+)$', after[path])
                if compiler:
                    versions.add(compiler[0])
        if len(versions) != 1:
            raise ValueError(
                'Go proposal does not identify one compiler version')
        return {
            'kind': 'go',
            'version': versions.pop(),
            'go_images': sorted(set(images))
        }
    argo_manifest = 'manifests/kustomize/third-party/argo/base/workflow-controller-deployment-patch.yaml'
    if paths <= {'go.mod', 'go.sum', argo_manifest}:
        versions = set()
        if 'go.mod' in paths:
            pattern = r'\s*(?:require\s+)?github\.com/argoproj/argo-workflows/v\d+\s+(v\d+\.\d+\.\d+)(?:\s+//.*)?'
            if not only_lines(before['go.mod'], after['go.mod'], pattern):
                return None
            versions.update(
                re.findall(
                    r'github\.com/argoproj/argo-workflows/v\d+\s+(v\d+\.\d+\.\d+)',
                    after['go.mod']))
        if argo_manifest in paths:
            pattern = r'.*quay\.io/argoproj/(?:workflow-controller|argoexec):(v\d+\.\d+\.\d+).*'
            if not only_lines(before[argo_manifest], after[argo_manifest],
                              pattern):
                return None
            versions.update(
                re.findall(
                    r'quay\.io/argoproj/(?:workflow-controller|argoexec):(v\d+\.\d+\.\d+)',
                    '\n'.join(
                        line for line in difflib.ndiff(
                            before[argo_manifest].splitlines(),
                            after[argo_manifest].splitlines())
                        if line.startswith('+ '))))
        if not versions:
            return None
        if len(versions) != 1:
            raise ValueError(
                'Argo proposal contains conflicting target versions')
        return {'kind': 'argo', 'version': versions.pop()}
    if paths == {GH_AW_SOURCE}:
        pattern = r'[ \t]*uses: github/gh-aw/actions/setup-cli@([0-9a-f]{40}) # (v\d+\.\d+\.\d+)'
        if not only_lines(before[GH_AW_SOURCE], after[GH_AW_SOURCE], pattern):
            raise ValueError(
                'gh-aw proposal must update only the compiler action pin')
        matches = re.findall(pattern, after[GH_AW_SOURCE])
        if len(matches) != 1:
            raise ValueError('Expected one pinned gh-aw compiler')
        return {
            'kind': 'gh-aw',
            'version': matches[0][1],
            'compiler_sha': matches[0][0]
        }
    return None


def output_paths(plan):
    if plan['kind'] == 'python':
        return PYTHON_EXPORTS
    if plan['kind'] == 'go':
        return {str(path) for path in generators.GO_OUTPUT_PATHS}
    if plan['kind'] == 'argo':
        return {str(path) for path in generators.ARGO_OUTPUT_PATHS}
    if plan['kind'] == 'gh-aw':
        return {
            GH_AW_SOURCE, '.github/workflows/ai-analyzer.lock.yml',
            '.github/aw/actions-lock.json',
            '.github/resources/ci-workflow-inventory.json'
        }
    raise ValueError('Unsupported completion kind')


def validate_pr(pr, repository, head):
    if (pr['state'] != 'open' or pr['user']['login'] != 'dependabot[bot]' or
            pr['head']['repo']['full_name'] != repository or
            pr['base']['repo']['full_name'] != repository or
            pr['base']['ref'] != 'master' or
            not pr['head']['ref'].startswith('dependabot/') or
            pr['head']['sha'] != head):
        raise ValueError(
            'Expected one open same-repository Dependabot PR at the requested head targeting master'
        )
    if not SHA.fullmatch(head) or not SHA.fullmatch(pr['base']['sha']):
        raise ValueError('Invalid commit identity')


def proposal(repository, number, head):
    pr = api(f'repos/{repository}/pulls/{number}')
    validate_pr(pr, repository, head)
    peers = api(
        f'repos/{repository}/pulls?state=open&head={repository.split("/")[0]}:{pr["head"]["ref"]}&per_page=100',
        paginate=True)
    if len(peers) != 1 or peers[0]['number'] != number:
        raise ValueError('Head branch does not identify exactly one open PR')
    commits = api(
        f'repos/{repository}/pulls/{number}/commits?per_page=100',
        paginate=True)
    if not commits or commits[-1]['sha'] != head:
        raise ValueError('PR changed while reading commits')
    # Dependabot discards these reproducible commits on its next rebase. Never
    # recursively complete our own commit or take over a human-edited branch.
    if '[dependabot skip]' in commits[-1]['commit']['message']:
        return pr, None
    for commit in commits:
        if ((commit.get('author') or {}).get('login') != 'dependabot[bot]' or
            (commit.get('committer') or
             {}).get('login') not in ('dependabot[bot]', 'web-flow') or
                not commit['commit'].get('verification', {}).get('verified')):
            raise ValueError(
                'Only verified Dependabot commits can be completed automatically'
            )
    master = api(f'repos/{repository}/git/ref/heads/master')['object']['sha']
    if not SHA.fullmatch(master):
        raise ValueError('Invalid authoritative master identity')
    git('fetch', '--no-tags', 'origin', 'master')
    git('fetch', '--no-tags', 'origin', f'refs/pull/{number}/head')
    if git('rev-parse', 'FETCH_HEAD').decode().strip() != head:
        raise ValueError('PR head changed before fetching its source')
    base = git('merge-base', master, head).decode().strip()
    names = git('diff', '--name-status', '--no-renames', base,
                head).decode().splitlines()
    before, after = {}, {}
    for record in names:
        status, path = record.split('\t')
        if status != 'M':
            raise ValueError(
                'Completion does not support added, deleted or renamed inputs')
        for ref, dest in ((base, before), (head, after)):
            entry = git('ls-tree', ref, '--', path).decode().split()
            if not entry or entry[0] not in ('100644', '100755'):
                raise ValueError(
                    f'Completion input is not a regular file: {path}')
            dest[path] = git('show', f'{ref}:{path}').decode()
    plan = classify(before, after)
    if plan and plan['kind'] == 'gh-aw':
        target = api(f'repos/github/gh-aw/commits/{plan["version"]}')
        if target['sha'] != plan['compiler_sha']:
            raise ValueError(
                'gh-aw release tag does not match proposed compiler pin')
    if plan:
        plan.update(
            repository=repository,
            number=number,
            head=head,
            base=master,
            branch=pr['head']['ref'])
    return pr, plan


def prepare(event, destination):
    run_event = event['workflow_run']
    repository = event['repository']['full_name']
    if (run_event['event'] != 'pull_request' or run_event['path']
            != '.github/workflows/dependabot-completion-request.yml' or
            run_event['head_repository']['full_name'] != repository):
        raise ValueError('Unexpected completion signal workflow')
    candidates = api(
        f'repos/{repository}/pulls?state=open&head={repository.split("/")[0]}:{run_event["head_branch"]}&per_page=100',
        paginate=True)
    matches = [
        pr for pr in candidates
        if pr['head']['sha'] == run_event['head_sha'] and
        pr['user']['login'] == 'dependabot[bot]'
    ]
    if not matches:
        return None
    if len(matches) != 1:
        raise ValueError('Completion signal is ambiguous')
    _, plan = proposal(repository, matches[0]['number'], run_event['head_sha'])
    if plan:
        destination.write_text(json.dumps(plan, indent=2) + '\n')
    return plan


def snapshot(head, directory):
    # No PR checkout or scripts: extract a checked tree as generator data, with
    # tar's traversal/link guards. The clone supplies a clean tracked inventory.
    git('clone', '--shared', '--no-checkout', str(ROOT), str(directory))
    with tarfile.open(fileobj=io.BytesIO(git('archive', head))) as archive:
        archive.extractall(directory, filter='data')
    git('read-tree', head, cwd=directory)
    git('update-ref', 'HEAD', head, cwd=directory)


def generate(plan, directory, destination):
    snapshot(plan['head'], directory)
    if plan['kind'] == 'python':
        run('bash', str(SCRIPTS / 'export_python_requirements.sh'),
            str(directory))
    else:
        args = [
            'python3',
            str(SCRIPTS / 'complete_dependency_update.py'), '--repo',
            str(directory), '--kind', plan['kind'], '--version', plan['version']
        ]
        for image in plan.get('go_images', []):
            args += ['--go-image', image]
        run(*args)
    if plan['kind'] == 'gh-aw':
        import generate_ci_workflow_inventory as inventory
        records = [{
            'path': path.relative_to(directory).as_posix(),
            'content': path.read_text()
        }
                   for path in sorted((directory /
                                       '.github/workflows').glob('*'))
                   if path.suffix in ('.yaml', '.yml')]
        (directory / '.github/resources/ci-workflow-inventory.json').write_text(
            json.dumps(inventory.build_inventory(records), indent=2) + '\n')
    names = git('diff', '--name-only', cwd=directory).decode().splitlines()
    if set(names) - output_paths(plan):
        raise ValueError(
            'Generator modified files outside its declared output scope')
    files = {}
    for name in names:
        path = directory / name
        if path.is_symlink() or not path.is_file():
            raise ValueError('Generator output must be a regular file')
        files[name] = base64.b64encode(path.read_bytes()).decode()
    bundle = dict(plan=plan, files=files)
    serialized = json.dumps(bundle, indent=2) + '\n'
    if len(serialized.encode()) > MAX_BUNDLE_BYTES:
        raise ValueError('Generated completion exceeds bundle size limit')
    destination.write_text(serialized)
    return bool(files)


def validate_bundle(bundle):
    plan, files = bundle['plan'], bundle['files']
    if not isinstance(files,
                      dict) or not files or set(files) - output_paths(plan):
        raise ValueError('Unexpected or empty generated output set')
    for content in files.values():
        base64.b64decode(content, validate=True).decode('utf-8')
    return plan


def validate_event_binding(plan, event):
    signal = event['workflow_run']
    if (signal['event'] != 'pull_request' or
            signal['conclusion'] != 'success' or signal['path']
            != '.github/workflows/dependabot-completion-request.yml' or
            signal['head_repository']['full_name'] != plan['repository'] or
            event['repository']['full_name'] != plan['repository'] or
            signal['head_sha'] != plan['head'] or
            signal['head_branch'] != plan['branch']):
        raise ValueError(
            'Completion data does not belong to this workflow signal')


def publish(bundle, event):
    plan = validate_bundle(bundle)
    validate_event_binding(plan, event)
    _, current = proposal(plan['repository'], plan['number'], plan['head'])
    if current != plan:
        raise ValueError(
            'Proposal or base changed; regenerate completion for the live PR')
    name = git('config', 'user.name').decode().strip()
    email = git('config', 'user.email').decode().strip()
    if not name or not email or any(c in name + email for c in '\n\r'):
        raise ValueError(
            'Configure the publisher git identity before signing its commit')
    # Fetching and classifying a large proposal can take time. Repeat the live
    # state check immediately before the atomic append, including authoritative
    # master: the PR payload's base SHA may lag the branch ref.
    latest = api(f'repos/{plan["repository"]}/pulls/{plan["number"]}')
    validate_pr(latest, plan['repository'], plan['head'])
    master = api(
        f'repos/{plan["repository"]}/git/ref/heads/master')['object']['sha']
    if master != plan['base']:
        raise ValueError(
            'Master changed before publication; regenerate completion')
    mutation = '''mutation($input: CreateCommitOnBranchInput!) { createCommitOnBranch(input: $input) { commit { oid url } } }'''
    result = api(
        'graphql',
        body={
            'query': mutation,
            'variables': {
                'input': {
                    'branch': {
                        'repositoryNameWithOwner': plan['repository'],
                        'branchName': plan['branch']
                    },
                    'expectedHeadOid': plan['head'],
                    'message': {
                        'headline':
                            'chore(deps): complete generated dependency files [dependabot skip]',
                        'body':
                            f'Signed-off-by: {name} <{email}>'
                    },
                    'fileChanges': {
                        'additions': [{
                            'path': path,
                            'contents': content
                        } for path, content in sorted(bundle['files'].items())]
                    },
                }
            }
        })
    if result.get('errors'):
        raise ValueError(result['errors'])
    commit = result['data']['createCommitOnBranch']['commit']
    print(
        f'Published generated completion: {commit["url"]}; full CI and review are required on {commit["oid"]}.'
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        'operation', choices=('prepare', 'generate', 'validate', 'publish'))
    parser.add_argument(
        '--plan',
        type=Path,
        default=Path(os.environ.get('RUNNER_TEMP', '/tmp')) /
        'completion-plan.json')
    parser.add_argument(
        '--bundle',
        type=Path,
        default=Path(os.environ.get('RUNNER_TEMP', '/tmp')) / 'completion.json')
    args = parser.parse_args()
    if args.operation == 'prepare':
        plan = prepare(
            json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text()),
            args.plan)
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            output.write(f'ready={str(plan is not None).lower()}\n')
            if plan:
                output.write(
                    f'kind={plan["kind"]}\nversion={plan["version"]}\n')
    elif args.operation == 'generate':
        changed = generate(
            json.loads(args.plan.read_text()),
            args.plan.parent / 'completion-source', args.bundle)
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            output.write(f'changed={str(changed).lower()}\n')
    else:
        if args.bundle.stat().st_size > MAX_BUNDLE_BYTES:
            raise ValueError('Completion bundle exceeds size limit')
        bundle = json.loads(args.bundle.read_text())
        if args.operation == 'validate':
            validate_event_binding(
                validate_bundle(bundle),
                json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text()))
        else:
            publish(
                bundle,
                json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text()))


if __name__ == '__main__':
    main()
