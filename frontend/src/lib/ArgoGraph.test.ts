// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { expect, it } from 'vitest';
import { argoStaticView, type ArgoManifest } from './ArgoGraph';
import { getArtifactNodeKey, getTaskNodeKey, NodeTypeNames } from './v2/StaticFlow';

it('projects native tasks, artifacts and nested DAGs without constructing IR', () => {
  const manifest: ArgoManifest = {
    kind: 'WorkflowTemplate',
    metadata: { name: 'native' },
    spec: {
      entrypoint: 'root',
      templates: [
        {
          name: 'root',
          dag: {
            tasks: [
              { name: 'prepare', template: 'prepare' },
              {
                name: 'nested',
                template: 'inner',
                dependencies: ['prepare'],
                arguments: {
                  artifacts: [
                    { name: 'dataset', from: '{{tasks.prepare.outputs.artifacts.dataset}}' },
                  ],
                },
              },
            ],
          },
        },
        {
          name: 'prepare',
          container: { image: 'test' },
          outputs: {
            artifacts: [
              { name: 'dataset' },
              { name: 'dataset-metadata' },
              { name: 'kfp-metadata' },
            ],
          },
        },
        { name: 'inner', dag: { tasks: [{ name: 'train', template: 'train' }] } },
        { name: 'train', container: { image: 'test' } },
      ],
    },
  };
  const view = argoStaticView(manifest);
  expect(view.elements.find((element) => element.id === getTaskNodeKey('nested'))).toMatchObject({
    type: NodeTypeNames.SUB_DAG,
  });
  expect(view.elements.filter((element) => 'source' in element)).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        source: getArtifactNodeKey('prepare', 'dataset'),
        target: getTaskNodeKey('nested'),
      }),
    ]),
  );
  expect(view.resolve(['root', 'nested'])).toEqual(
    expect.arrayContaining([expect.objectContaining({ id: getTaskNodeKey('train') })]),
  );
  expect(view.elements.some((element) => element.data?.label === 'kfp-metadata')).toBe(false);
});
