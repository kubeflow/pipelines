// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
/** Engine-neutral graph layout and stable view keys. No pipeline IR or REST client. */
import dagre from 'dagre';
import { type Node, Position } from '@xyflow/react';
import type { FlowElementDataBase } from 'src/components/graph/Constants';
import type { PipelineFlowElement } from './FlowTypes';

export function isNode(element: PipelineFlowElement): element is Node<FlowElementDataBase> {
  return !('source' in element) && !('target' in element);
}

export function getNodeName(element: PipelineFlowElement | null): string {
  return typeof element?.data?.label === 'string' ? element.data.label : 'unknown';
}

export enum NodeTypeNames {
  EXECUTION = 'EXECUTION',
  ARTIFACT = 'ARTIFACT',
  SUB_DAG = 'SUB_DAG',
}
export enum TaskType {
  EXECUTOR,
  DAG,
}

export function buildGraphLayout(elements: PipelineFlowElement[]): PipelineFlowElement[] {
  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({ rankdir: 'TB' });
  for (const element of elements) {
    if (isNode(element))
      graph.setNode(element.id, { width: element.width ?? 224, height: element.height ?? 48 });
    else graph.setEdge(element.source, element.target);
  }
  dagre.layout(graph);
  return elements.map((element) => {
    if (isNode(element)) {
      const position = graph.node(element.id);
      element.sourcePosition = Position.Bottom;
      element.targetPosition = Position.Top;
      element.position = {
        x: position.x - (element.width ?? 224) / 2,
        y: position.y - (element.height ?? 48) / 2,
      };
    }
    return element;
  });
}

const TASK_PREFIX = 'task.';
const ARTIFACT_PREFIX = 'artifact.';
export const getTaskNodeKey = (key: string) => TASK_PREFIX + key;
export const isTaskNode = (key: string) => key.startsWith(TASK_PREFIX);
export function getTaskKeyFromNodeKey(key: string): string {
  if (!isTaskNode(key)) throw new Error(`Task nodeKey: ${key} doesn't start with ${TASK_PREFIX}`);
  return key.substring(TASK_PREFIX.length);
}
export const getArtifactNodeKey = (task: string, artifact: string) =>
  `${ARTIFACT_PREFIX}${task}.${artifact}`;
export const isArtifactNode = (key: string) => key.startsWith(ARTIFACT_PREFIX);
export function getKeysFromArtifactNodeKey(key: string): string[] {
  if (!isArtifactNode(key))
    throw new Error(`Artifact nodeKey: ${key} doesn't start with ${ARTIFACT_PREFIX}`);
  const parts = key.split('.');
  if (parts.length !== 3)
    throw new Error(`Artifact nodeKey: ${key} doesn't have format artifact.taskName.artifactName`);
  return parts.slice(1);
}
