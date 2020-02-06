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
    name: 'coinflip-error-nklng2',
    namespace: 'default',
    // tslint:disable-next-line:max-line-length
    selfLink:
      '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/coinflip-heads-c085010d-771a-4cdf-979c-257e991501b5',
    uid: '47a3d09c-7db4-4788-ac55-3f8d908574aa',
    resourceVersion: '10527150',
    creationTimestamp: '2018-06-11T22:49:26Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Failed',
    },
  },
  spec: {
    templates: [
      {
        name: 'coinflip',
        inputs: {},
        outputs: {},
        metadata: {},
        steps: [
          [
            {
              name: 'heads',
              template: 'heads',
              arguments: {},
              when: '{{steps.flip-coin.outputs.result}} == heads',
            },
          ],
        ],
      },
      {
        name: 'heads',
        inputs: {},
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'alpine:3.6',
          command: ['sh', '-c'],
          args: ['echo "it was heads"'],
          resources: {},
        },
      },
    ],
    entrypoint: 'coinflip',
    arguments: {},
  },
  status: {
    phase: 'Failed',
    startedAt: '2018-06-11T22:49:26Z',
    finishedAt: '2018-06-11T22:49:26Z',
    // tslint:disable-next-line:max-line-length
    message:
      'invalid spec: templates.coinflip.steps[0].heads failed to resolve {{steps.flip-coin.outputs.result}}',
  },
};
