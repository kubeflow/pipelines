// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// tslint:disable:object-literal-sort-keys
export default {
  metadata: {
    name: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
    namespace: 'default',
    // tslint:disable-next-line:max-line-length
    selfLink:
      '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
    uid: 'ef2a4a61-6e84-11e8-bba7-42010a8a0fc2',
    resourceVersion: '10690686',
    creationTimestamp: '2018-06-12T21:09:46Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
    },
  },
  spec: {
    templates: [
      {
        name: 'whalesay',
        inputs: {},
        outputs: {},
        metadata: {},
        steps: [
          [
            {
              name: 'say',
              template: 'say',
              arguments: {},
            },
          ],
        ],
      },
      {
        name: 'say',
        inputs: {},
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'docker/whalesay:latest',
          command: ['cowsay'],
          args: ['hello world'],
          resources: {},
        },
      },
    ],
    entrypoint: 'whalesay',
    arguments: {},
  },
  status: {
    phase: 'Succeeded',
    startedAt: '2018-06-12T21:09:46Z',
    finishedAt: '2018-06-12T21:09:47Z',
    nodes: {
      'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c': {
        id: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
        name: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
        displayName: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
        type: 'Steps',
        templateName: 'whalesay',
        phase: 'Succeeded',
        startedAt: '2018-06-12T21:09:46Z',
        finishedAt: '2018-06-12T21:09:47Z',
        children: ['hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-2303694156'],
      },
      'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-2303694156': {
        id: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-2303694156',
        name: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c[0]',
        displayName: '[0]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
        startedAt: '2018-06-12T21:09:46Z',
        finishedAt: '2018-06-12T21:09:47Z',
        children: ['hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-3584189705'],
      },
      'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-3584189705': {
        id: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c-3584189705',
        name: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c[0].say',
        displayName: 'say',
        type: 'Pod',
        templateName: 'say',
        phase: 'Succeeded',
        boundaryID: 'hello-world-61985dbf-4299-458b-a183-1f2c2436c21c',
        startedAt: '2018-06-12T21:09:46Z',
        finishedAt: '2018-06-12T21:09:47Z',
      },
    },
  },
};
