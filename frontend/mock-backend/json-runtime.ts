// Copyright 2019 Google LLC
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
    name: 'json-12abc',
    generateName: 'json-',
    namespace: 'default',
    selfLink: '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/json-7sm94',
    uid: 'dfc82af5-c5cb-43b1-822b-52487cb872d2',
    resourceVersion: '1322',
    creationTimestamp: '2018-06-06T00:04:49Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
    },
  },
  spec: {
    templates: [
      {
        name: 'whalesay1',
        inputs: {},
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'docker/whalesay:latest',
          command: ['cowsay'],
          args: ['{{workflow.parameters.message}}'],
          resources: {},
        },
      },
    ],
    entrypoint: 'whalesay1',
    arguments: {
      parameters: [
        {
          name: 'message',
          value: 'hello world',
        },
      ],
    },
  },
  status: {
    phase: 'Succeeded',
    startedAt: '2018-06-06T00:04:49Z',
    finishedAt: '2018-06-06T00:05:23Z',
    nodes: {
      'json-12abc': {
        id: 'json-12abc',
        name: 'json-12abc',
        displayName: 'json-12abc',
        type: 'Pod',
        templateName: 'whalesay1',
        phase: 'Succeeded',
        startedAt: '2018-06-06T00:04:49Z',
        finishedAt: '2018-06-06T00:05:23Z',
        inputs: {
          parameters: [
            {
              name: 'JSON Data',
              value: JSON.stringify({
                string1: 'a',
                string2: 'b',
                number1: 1,
                number2: 2.2,
                object: {
                  string: 'a',
                  number: 2,
                },
                array: ['a', 'b', 'c'],
              }),
            },
          ],
        },
      },
    },
  },
};
