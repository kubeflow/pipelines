// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import { argoStaticView, type ArgoManifest } from 'src/lib/ArgoGraph';

const metadata = (label: string) => ({
  annotations: { 'workflows.argoproj.io/display-name': label },
});
const container = (name: string, label: string, artifacts: string[] = []) => ({
  name,
  metadata: metadata(label),
  container: { image: 'python:3.11' },
  outputs: { artifacts: artifacts.map((name) => ({ name })) },
});
const dag = (name: string, label: string, tasks: any[], artifacts: string[] = []) => ({
  name,
  metadata: metadata(label),
  dag: { tasks },
  outputs: { artifacts: artifacts.map((name) => ({ name })) },
});
const task = (name: string, template: string, dependencies: string[] = []) => ({
  name,
  template,
  dependencies,
});
const input = (source: string) => ({
  artifacts: [{ name: 'model', from: `{{tasks.${source}.outputs.artifacts.model}}` }],
});

export const nestedArtifactSpec: ArgoManifest = {
  kind: 'WorkflowTemplate',
  metadata: { name: 'nested-training' },
  spec: {
    entrypoint: 'root',
    templates: [
      container('train', 'Train model', ['model']),
      container('evaluate', 'Evaluate model'),
      container('prepare', 'Prepare data'),
      container('deploy', 'Deploy model'),
      dag(
        'body',
        'Training and evaluation',
        [task('train', 'train'), { ...task('evaluate', 'evaluate'), arguments: input('train') }],
        ['model'],
      ),
      dag(
        'workflow',
        'Training pipeline',
        [task('prepare', 'prepare'), task('fit', 'body', ['prepare'])],
        ['model'],
      ),
      dag('root', 'Root', [
        task('workflow', 'workflow'),
        { ...task('deploy', 'deploy'), arguments: input('workflow') },
      ]),
    ],
  },
};

export function manySubDagsSpec(count = 25, parallelBranches = 1): ArgoManifest {
  return {
    kind: 'WorkflowTemplate',
    spec: {
      entrypoint: 'root',
      templates: [
        container('step', 'Step'),
        ...Array.from({ length: count }, (_, index) =>
          dag(`body-${index}`, `Group ${index}`, [task('step', 'step')]),
        ),
        dag(
          'root',
          'Root',
          Array.from({ length: count }, (_, index) =>
            task(
              `group-${index}`,
              `body-${index}`,
              index >= parallelBranches ? [`group-${index - parallelBranches}`] : [],
            ),
          ).reverse(),
        ),
      ],
    },
  };
}

export function deeplyNestedSpec(): ArgoManifest {
  const templates: any[] = [dag('root', 'Root', [task('level-1', 'level-1')])];
  for (let depth = 1; depth <= 25; depth++) {
    const tasks: any[] = [];
    for (let index = 1; index <= 2; index++) {
      const number = (depth - 1) * 2 + index;
      const name = `component-${number}`;
      templates.push(container(name, `Component ${String(number).padStart(3, '0')}`));
      tasks.push(task(name, name));
    }
    if (depth < 25)
      tasks.push(
        task(
          `level-${depth + 1}`,
          `level-${depth + 1}`,
          tasks.map((item) => item.name),
        ),
      );
    templates.push(dag(`level-${depth}`, `Level ${String(depth).padStart(2, '0')}`, tasks));
  }
  return {
    kind: 'WorkflowTemplate',
    metadata: { name: 'nested-50-components-25-levels' },
    spec: { entrypoint: 'root', templates },
  };
}

export const resolveGraph = (spec: ArgoManifest, layers: string[]) =>
  argoStaticView(spec).resolve(layers);
