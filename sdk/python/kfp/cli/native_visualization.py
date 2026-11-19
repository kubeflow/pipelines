# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Read-only projection of native Argo templates; never construct KFP IR."""

import re
from graphlib import CycleError, TopologicalSorter

KINDS = {'Workflow', 'WorkflowTemplate', 'CronWorkflow'}


def workflow_spec(manifest):
    spec = manifest.get('spec', {})
    return (
        spec.get('workflowSpec', {})
        if manifest.get('kind') == 'CronWorkflow'
        else spec
    )


def view(manifest):
    spec = workflow_spec(manifest)
    templates = {
        template['name']: template for template in spec.get('templates', [])
    }
    entrypoint = spec.get('entrypoint')
    if entrypoint not in templates:
        raise ValueError(
            'Argo entrypoint has no local template; resolve workflowTemplateRef before exporting.'
        )
    graphs, edges_by_template = [], {}
    pending = [(entrypoint, [])]
    remaining = 10000
    while pending:
        name, path = pending.pop()
        template = templates[name]
        tasks = template.get('dag', {}).get('tasks', [])
        if 'steps' in template:
            raise ValueError(
                'Offline native previews currently require DAG templates, not steps.'
            )
        if not tasks:
            tasks = [{'name': name, 'template': name}]
        ids = {task['name'] for task in tasks}
        dependencies = {}
        nodes, children = [], []
        for task in tasks:
            target = templates.get(task.get('template'))
            if target is None:
                raise ValueError(
                    f'Argo task {task["name"]!r} references a missing or external template.'
                )
            deps = set(task.get('dependencies', [])) | set(
                re.findall(
                    r'([a-zA-Z0-9_-]+)\.(?:Succeeded|Failed|Errored|Skipped|Omitted|AnySucceeded|AllFailed)',
                    task.get('depends', ''),
                )
            )
            if deps - ids or task['name'] in deps:
                raise ValueError(
                    f'Argo task {task["name"]!r} has an invalid dependency.'
                )
            dependencies[task['name']] = deps
            kind = (
                'Loop'
                if 'withParam' in task or 'withItems' in task
                else 'Condition'
                if 'when' in task
                else 'Sub-DAG'
                if 'dag' in target
                else 'Approval'
                if 'suspend' in target
                else 'Task'
            )
            nodes.append(
                {'id': task['name'], 'label': task['name'], 'kind': kind}
            )
            if 'dag' in target:
                children.append((task['template'], path + [task['name']]))
        try:
            TopologicalSorter(dependencies).prepare()
        except CycleError as error:
            raise ValueError(
                'Argo DAG has cyclic task dependencies.'
            ) from error
        edges_by_template[name] = {child[0] for child in children}
        try:
            TopologicalSorter(edges_by_template).prepare()
        except CycleError as error:
            raise ValueError(
                'Argo templates contain a recursive DAG reference.'
            ) from error
        edges = [
            {'source': source, 'target': target}
            for target, sources in dependencies.items()
            for source in sorted(sources)
        ]
        remaining -= 1 + len(nodes) + len(edges)
        if remaining < 0:
            raise ValueError(
                'Native preview exceeds 10,000 graph elements; use a smaller pipeline.'
            )
        graphs.append(
            {
                'label': ' / '.join(path) if path else 'Pipeline',
                'nodes': nodes,
                'edges': edges,
            }
        )
        pending.extend(reversed(children))
    metadata = manifest.get('metadata', {})
    name = (
        metadata.get('annotations', {}).get(
            'pipelines.kubeflow.org/display-name'
        )
        or metadata.get('name')
        or metadata.get('generateName', 'Argo pipeline')
    )
    return {'name': name, 'graphs': graphs}
