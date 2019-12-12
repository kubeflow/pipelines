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
    name: 'coinflip-recursive-q7dqb',
    generateName: 'coinflip-recursive-',
    namespace: 'default',
    selfLink: '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/coinflip-recursive-q7dqb',
    uid: '114660ca-4282-11e8-bba7-42010a8a0fc2',
    resourceVersion: '848911',
    creationTimestamp: '2018-04-17T20:58:23Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
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
              name: 'flip-coin',
              template: 'flip-coin',
              arguments: {},
            },
          ],
          [
            {
              name: 'heads',
              template: 'heads',
              arguments: {},
              when: '{{steps.flip-coin.outputs.result}} == heads',
            },
            {
              name: 'tails',
              template: 'coinflip',
              arguments: {},
              when: '{{steps.flip-coin.outputs.result}} == tails',
            },
          ],
        ],
      },
      {
        name: 'flip-coin',
        inputs: {},
        outputs: {},
        metadata: {},
        script: {
          name: '',
          image: 'python:alpine3.6',
          command: ['python'],
          resources: {},
          // tslint:disable-next-line:max-line-length
          source:
            'import random\nresult = "heads" if random.randint(0,1) == 0 else "tails"\nprint(result)\n',
        },
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
    arguments: {
      parameters: [
        {
          name: 'x',
          value: 10,
        },
        {
          name: 'y',
          value: 20,
        },
      ],
    },
  },
  status: {
    phase: 'Succeeded',
    startedAt: '2018-04-17T20:58:23Z',
    finishedAt: '2018-04-17T20:58:38Z',
    nodes: {
      'coinflip-recursive-q7dqb': {
        id: 'coinflip-recursive-q7dqb',
        name: 'coinflip-recursive-q7dqb',
        displayName: 'coinflip-recursive-q7dqb',
        type: 'Steps',
        templateName: 'coinflip',
        phase: 'Succeeded',
        startedAt: '2018-04-17T20:58:23Z',
        finishedAt: '2018-04-17T20:58:38Z',
        children: ['coinflip-recursive-q7dqb-1787723858', 'coinflip-recursive-q7dqb-1720466287'],
        outboundNodes: ['coinflip-recursive-q7dqb-3721646052'],
      },
      'coinflip-recursive-q7dqb-1720466287': {
        id: 'coinflip-recursive-q7dqb-1720466287',
        name: 'coinflip-recursive-q7dqb[1]',
        displayName: '[1]',
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'staging',
              },
            },
          ],
        },
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb',
        startedAt: '2018-04-17T20:58:28Z',
        finishedAt: '2018-04-17T20:58:38Z',
        children: ['coinflip-recursive-q7dqb-4011569486', 'coinflip-recursive-q7dqb-3266226990'],
      },
      'coinflip-recursive-q7dqb-1787723858': {
        id: 'coinflip-recursive-q7dqb-1787723858',
        name: 'coinflip-recursive-q7dqb[0]',
        displayName: '[0]',
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'analysis2',
              },
            },
          ],
        },
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb',
        startedAt: '2018-04-17T20:58:23Z',
        finishedAt: '2018-04-17T20:58:28Z',
        children: ['coinflip-recursive-q7dqb-311338607'],
      },
      'coinflip-recursive-q7dqb-2934726852': {
        id: 'coinflip-recursive-q7dqb-2934726852',
        name: 'coinflip-recursive-q7dqb[1].tails[1].tails',
        displayName: 'tails',
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'transform',
              },
            },
          ],
        },
        type: 'Skipped',
        phase: 'Skipped',
        boundaryID: 'coinflip-recursive-q7dqb-3266226990',
        message: "when 'heads == tails' evaluated false",
        startedAt: '2018-04-17T20:58:34Z',
        finishedAt: '2018-04-17T20:58:34Z',
      },
      'coinflip-recursive-q7dqb-311338607': {
        id: 'coinflip-recursive-q7dqb-311338607',
        name: 'coinflip-recursive-q7dqb[0].flip-coin',
        displayName: 'flip-coin',
        type: 'Pod',
        templateName: 'flip-coin',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb',
        startedAt: '2018-04-17T20:58:23Z',
        finishedAt: '2018-04-17T20:58:28Z',
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'model2',
              },
            },
          ],
          parameters: [
            {
              name: 'result',
              value: 'tails',
            },
          ],
        },
        children: ['coinflip-recursive-q7dqb-1720466287'],
      },
      'coinflip-recursive-q7dqb-3266226990': {
        id: 'coinflip-recursive-q7dqb-3266226990',
        name: 'coinflip-recursive-q7dqb[1].tails',
        displayName: 'tails',
        type: 'Steps',
        templateName: 'coinflip',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb',
        startedAt: '2018-04-17T20:58:28Z',
        finishedAt: '2018-04-17T20:58:38Z',
        children: ['coinflip-recursive-q7dqb-4010083248', 'coinflip-recursive-q7dqb-855846949'],
        outboundNodes: ['coinflip-recursive-q7dqb-3721646052'],
      },
      'coinflip-recursive-q7dqb-3466727817': {
        id: 'coinflip-recursive-q7dqb-3466727817',
        name: 'coinflip-recursive-q7dqb[1].tails[0].flip-coin',
        displayName: 'flip-coin-with-long-log',
        type: 'Pod',
        templateName: 'flip-coin',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb-3266226990',
        startedAt: '2018-04-17T20:58:28Z',
        finishedAt: '2018-04-17T20:58:33Z',
        outputs: {
          parameters: [
            {
              name: 'result',
              value: 'heads',
            },
          ],
        },
        children: ['coinflip-recursive-q7dqb-855846949'],
      },
      'coinflip-recursive-q7dqb-3721646052': {
        id: 'coinflip-recursive-q7dqb-3721646052',
        name: 'coinflip-recursive-q7dqb[1].tails[1].heads',
        displayName: 'heads',
        type: 'Pod',
        templateName: 'heads',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb-3266226990',
        startedAt: '2018-04-17T20:58:34Z',
        finishedAt: '2018-04-17T20:58:37Z',
      },
      'coinflip-recursive-q7dqb-4010083248': {
        id: 'coinflip-recursive-q7dqb-4010083248',
        name: 'coinflip-recursive-q7dqb[1].tails[0]',
        displayName: '[0]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb-3266226990',
        startedAt: '2018-04-17T20:58:28Z',
        finishedAt: '2018-04-17T20:58:34Z',
        children: ['coinflip-recursive-q7dqb-3466727817'],
      },
      'coinflip-recursive-q7dqb-4011569486': {
        id: 'coinflip-recursive-q7dqb-4011569486',
        name: 'coinflip-recursive-q7dqb[1].heads',
        displayName: 'heads',
        type: 'Skipped',
        phase: 'Skipped',
        boundaryID: 'coinflip-recursive-q7dqb',
        message: "when 'tails == heads' evaluated false",
        startedAt: '2018-04-17T20:58:28Z',
        finishedAt: '2018-04-17T20:58:28Z',
      },
      'coinflip-recursive-q7dqb-855846949': {
        id: 'coinflip-recursive-q7dqb-855846949',
        name: 'coinflip-recursive-q7dqb[1].tails[1]',
        displayName: '[1]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'coinflip-recursive-q7dqb-3266226990',
        startedAt: '2018-04-17T20:58:34Z',
        finishedAt: '2018-04-17T20:58:38Z',
        children: ['coinflip-recursive-q7dqb-3721646052', 'coinflip-recursive-q7dqb-2934726852'],
      },
    },
  },
};
