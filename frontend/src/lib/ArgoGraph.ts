// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import type { PipelineFlowElement } from './v2/FlowTypes';
import {
  buildGraphLayout,
  getArtifactNodeKey,
  getTaskNodeKey,
  getTaskKeyFromNodeKey,
  getKeysFromArtifactNodeKey,
  isArtifactNode,
  isTaskNode,
  NodeTypeNames,
  TaskType,
} from './v2/StaticFlow';

export interface ArgoManifest {
  kind: string;
  metadata?: { name?: string; generateName?: string; annotations?: Record<string, string> };
  spec: {
    workflowSpec?: ArgoManifest['spec'];
    entrypoint?: string;
    templates?: Array<Record<string, any>>;
  };
}

/** A view model over native templates; never translate back into PipelineSpec IR. */
export function argoStaticView(manifest: ArgoManifest) {
  const spec = manifest.kind === 'CronWorkflow' ? manifest.spec.workflowSpec! : manifest.spec;
  const templates = new Map((spec.templates || []).map((template) => [template.name, template]));
  function lookup(layers: string[]) {
    let template = templates.get(spec.entrypoint);
    for (const task of layers.slice(1)) {
      const reference = template?.dag?.tasks?.find((item: any) => item.name === task)?.template;
      template = templates.get(reference);
    }
    if (!template) throw new Error('Native graph references a missing template.');
    return template;
  }
  function resolve(layers: string[]): PipelineFlowElement[] {
    const template = lookup(layers);
    const tasks = template.dag?.tasks || [{ name: template.name, template: template.name }];
    const elements: PipelineFlowElement[] = [];
    const artifacts = new Set<string>();
    for (const task of tasks) {
      const target = templates.get(task.template);
      if (!target) throw new Error(`Missing native template ${task.template}.`);
      elements.push({
        id: getTaskNodeKey(task.name),
        type: target.dag ? NodeTypeNames.SUB_DAG : NodeTypeNames.EXECUTION,
        position: { x: 0, y: 0 },
        data: {
          label: target.metadata?.annotations?.['workflows.argoproj.io/display-name'] || task.name,
          taskType: target.dag ? TaskType.DAG : TaskType.EXECUTOR,
          groupKind:
            task.withParam || task.withItems ? 'Loop' : task.when ? 'Condition' : 'Sub-DAG',
        },
      });
      for (const artifact of target.outputs?.artifacts || []) {
        if (artifact.name === 'kfp-metadata' || artifact.name.endsWith('-metadata')) continue;
        const id = getArtifactNodeKey(task.name, artifact.name);
        artifacts.add(id);
        elements.push({
          id,
          type: NodeTypeNames.ARTIFACT,
          position: { x: 0, y: 0 },
          data: { label: artifact.name, outputArtifactKey: artifact.name },
        });
        elements.push({
          id: `${getTaskNodeKey(task.name)}-${id}`,
          source: getTaskNodeKey(task.name),
          target: id,
        });
      }
    }
    for (const task of tasks) {
      const deps = new Set<string>(task.dependencies || []);
      for (const match of (task.depends || '').matchAll(/([a-zA-Z0-9_-]+)(?:\.[A-Za-z]+)?/g))
        if (!['true', 'false'].includes(match[1])) deps.add(match[1]);
      for (const source of deps)
        elements.push({
          id: `${source}-${task.name}`,
          source: getTaskNodeKey(source),
          target: getTaskNodeKey(task.name),
        });
      for (const artifact of task.arguments?.artifacts || []) {
        const match = artifact.from?.match(/^\{\{tasks\.([^.]+)\.outputs\.artifacts\.([^}]+)\}\}$/);
        if (!match) continue;
        const source = getArtifactNodeKey(match[1], match[2]);
        if (artifacts.has(source))
          elements.push({
            id: `${source}-${task.name}`,
            source,
            target: getTaskNodeKey(task.name),
          });
      }
    }
    return buildGraphLayout(elements);
  }
  return {
    name:
      manifest.metadata?.annotations?.['pipelines.kubeflow.org/display-name'] ||
      manifest.metadata?.name ||
      manifest.metadata?.generateName ||
      'Argo pipeline',
    elements: resolve(['root']),
    resolve,
    describe(layers: string[], nodeId: string) {
      if (!isArtifactNode(nodeId) && !isTaskNode(nodeId))
        return { task: undefined, template: undefined };
      const parent = lookup(layers);
      const taskName = isArtifactNode(nodeId)
        ? getKeysFromArtifactNodeKey(nodeId)[0]
        : getTaskKeyFromNodeKey(nodeId);
      const task = (parent.dag?.tasks || [{ name: parent.name, template: parent.name }]).find(
        (task: any) => task.name === taskName,
      );
      return { task, template: templates.get(task?.template) };
    },
  };
}
