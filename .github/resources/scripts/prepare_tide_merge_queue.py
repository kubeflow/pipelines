#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Prepare an offline Prow config patch; never update the deployed configuration."""

import argparse
import copy
import difflib
from pathlib import Path

import yaml
from generate_ci_workflow_inventory import UniqueKeyLoader

REPOSITORY = 'kubeflow/pipelines'
BRANCHES = ('master', 'main')
CONFIG_PATH = 'prow/oss/config.yaml'


def covers_repository(query):
    if 'kubeflow' in query.get('orgs', []):
        return REPOSITORY not in query.get('excludedRepos', [])
    return REPOSITORY in query.get('repos', [])


def covers_branch(query, branch):
    return branch not in query.get('excludedBranches', []) and (
        'includedBranches' not in query or branch in query['includedBranches']
    )


def prepare_queries(queries):
    """Split matching queries so only development-branch direct merges stop."""
    if not isinstance(queries, list) or not queries:
        raise ValueError('Expected a nonempty Tide queries list')
    result = []
    for query in queries:
        if not isinstance(query, dict) or not (
            query.get('repos') or query.get('orgs')
        ):
            raise ValueError('Every Tide query must have an explicit scope')
        for key in (
            'repos',
            'orgs',
            'excludedRepos',
            'includedBranches',
            'excludedBranches',
        ):
            if key in query and (
                not isinstance(query[key], list)
                or any(not isinstance(value, str) for value in query[key])
            ):
                raise ValueError(f'Invalid Tide query field: {key}')
        if not covers_repository(query) or not any(
            covers_branch(query, branch) for branch in BRANCHES
        ):
            result.append(copy.deepcopy(query))
            continue
        remaining = copy.deepcopy(query)
        if 'repos' in remaining:
            remaining['repos'] = [
                repo for repo in remaining['repos'] if repo != REPOSITORY
            ]
            if not remaining['repos']:
                del remaining['repos']
        if 'kubeflow' in remaining.get('orgs', []):
            remaining['excludedRepos'] = [
                *remaining.get('excludedRepos', []),
                REPOSITORY,
            ]
        if remaining.get('repos') or remaining.get('orgs'):
            result.append(remaining)

        release = copy.deepcopy(query)
        for key in ('orgs', 'excludedRepos'):
            release.pop(key, None)
        release['repos'] = [REPOSITORY]
        release['excludedBranches'] = list(
            dict.fromkeys(
                [
                    *release.get('excludedBranches', []),
                    *BRANCHES,
                ]
            )
        )
        if 'includedBranches' in release:
            release['includedBranches'] = [
                branch
                for branch in release['includedBranches']
                if branch not in BRANCHES
            ]
            if not release['includedBranches']:
                continue
        if release not in result:
            result.append(release)
    if any(
        covers_repository(query) and covers_branch(query, branch)
        for query in result
        for branch in BRANCHES
    ):
        raise ValueError(
            'Development branch still has a direct Tide merge route'
        )
    return result


def prepare_config(source):
    """Preserve everything outside the queries block, including comments."""
    config = yaml.load(source, Loader=UniqueKeyLoader)
    if not isinstance(config, dict) or not isinstance(config.get('tide'), dict):
        raise ValueError('Expected a Tide configuration mapping')
    original = config['tide'].get('queries')
    prepared = prepare_queries(original)
    if prepared == original:
        return source
    root = yaml.compose(source, Loader=UniqueKeyLoader)
    tide = next(value for key, value in root.value if key.value == 'tide')
    queries = next(value for key, value in tide.value if key.value == 'queries')
    if queries.flow_style:
        raise ValueError('Expected block-style Tide queries')

    def last_content_line(node):
        if isinstance(node, yaml.ScalarNode):
            return node.end_mark.line
        children = (
            node.value
            if isinstance(node, yaml.SequenceNode)
            else [item for pair in node.value for item in pair]
        )
        return (
            max(last_content_line(child) for child in children)
            if children
            else node.end_mark.line
        )

    lines = source.splitlines(keepends=True)
    parts = [''.join(lines[: queries.start_mark.line])]
    seen = []
    for index, (query, node) in enumerate(zip(original, queries.value)):
        end_line = (
            queries.value[index + 1].start_mark.line
            if index + 1 < len(queries.value)
            else last_content_line(node) + 1
        )
        rewritten = [
            item for item in prepare_queries([query]) if item not in seen
        ]
        seen.extend(rewritten)
        if rewritten == [query]:
            parts.append(''.join(lines[node.start_mark.line : end_line]))
        elif rewritten:
            rendered = yaml.safe_dump(rewritten, sort_keys=False, width=1000)
            parts.append(
                ''.join(
                    ' ' * queries.start_mark.column + line + '\n'
                    for line in rendered.splitlines()
                )
            )
    parts.append(''.join(lines[end_line:]))
    updated = ''.join(parts)
    expected = copy.deepcopy(config)
    expected['tide']['queries'] = prepared
    if yaml.load(updated, Loader=UniqueKeyLoader) != expected:
        raise ValueError(
            'Prepared patch changed configuration outside Tide queries'
        )
    return updated


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'config', type=Path, help='Downloaded or checked-out Prow config.yaml'
    )
    parser.add_argument(
        '--patch',
        type=Path,
        required=True,
        help='Output reviewable patch; input is never modified',
    )
    args = parser.parse_args()
    source = args.config.read_text()
    prepared = prepare_config(source)
    args.patch.write_text(
        ''.join(
            difflib.unified_diff(
                source.splitlines(keepends=True),
                prepared.splitlines(keepends=True),
                fromfile='a/' + CONFIG_PATH,
                tofile='b/' + CONFIG_PATH,
            )
        )
    )


if __name__ == '__main__':
    main()
