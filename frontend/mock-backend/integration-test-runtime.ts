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
    name: 'job-cloneofhelloworldls94q-1-3667110102',
    namespace: 'kubeflow',
    selfLink:
      '/apis/argoproj.io/v1alpha1/namespaces/kubeflow/workflows/job-cloneofhelloworldls94q-1-3667110102',
    uid: '55dc2b6d-d688-11e8-83db-42010a800093',
    resourceVersion: '128069',
    creationTimestamp: '2018-10-23T05:56:07Z',
    labels: {
      'scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow': 'true',
      'scheduledworkflows.kubeflow.org/scheduledWorkflowName': 'job-cloneofhelloworldls94q',
      'scheduledworkflows.kubeflow.org/workflowEpoch': '1540274157',
      'scheduledworkflows.kubeflow.org/workflowIndex': '1',
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
    },
    ownerReferences: [
      {
        apiVersion: 'kubeflow.org/v1beta1',
        kind: 'ScheduledWorkflow',
        name: 'job-cloneofhelloworldls94q',
        uid: '4fac8e0f-d688-11e8-83db-42010a800093',
        controller: true,
        blockOwnerDeletion: true,
      },
    ],
  },
  spec: {
    templates: [
      {
        name: 'diamond',
        inputs: {},
        outputs: {},
        metadata: {},
        dag: {
          tasks: [
            {
              name: 'A',
              template: 'echo',
              arguments: {
                parameters: [
                  {
                    name: 'message',
                    value: '{{workflow.parameters.message}} from node: A',
                  },
                ],
              },
            },
            {
              name: 'B',
              template: 'echo',
              arguments: {
                parameters: [
                  {
                    name: 'message',
                    value: '{{workflow.parameters.message}} from node: B',
                  },
                ],
              },
              dependencies: ['A'],
            },
            {
              name: 'C',
              template: 'echo',
              arguments: {
                parameters: [
                  {
                    name: 'message',
                    value: '{{workflow.parameters.message}} from node: C',
                  },
                ],
              },
              dependencies: ['A'],
            },
            {
              name: 'D',
              template: 'echo',
              arguments: {
                parameters: [
                  {
                    name: 'message',
                    value: '{{workflow.parameters.message}} from node: D',
                  },
                ],
              },
              dependencies: ['B', 'C'],
            },
          ],
        },
      },
      {
        name: 'echo',
        inputs: {
          parameters: [
            {
              name: 'message',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'alpine:3.7',
          command: ['echo', '{{inputs.parameters.message}}'],
          resources: {},
        },
      },
    ],
    entrypoint: 'diamond',
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
    startedAt: '2018-10-23T05:56:07Z',
    finishedAt: '2018-10-23T05:56:25Z',
    nodes: {
      'job-cloneofhelloworldls94q-1-3667110102': {
        id: 'job-cloneofhelloworldls94q-1-3667110102',
        name: 'job-cloneofhelloworldls94q-1-3667110102',
        displayName: 'job-cloneofhelloworldls94q-1-3667110102',
        type: 'DAG',
        templateName: 'diamond',
        phase: 'Succeeded',
        startedAt: '2018-10-23T05:56:07Z',
        finishedAt: '2018-10-23T05:56:25Z',
        children: ['job-cloneofhelloworldls94q-1-3667110102-3867833025'],
        outboundNodes: ['job-cloneofhelloworldls94q-1-3667110102-3918165882'],
      },
      'job-cloneofhelloworldls94q-1-3667110102-3817500168': {
        id: 'job-cloneofhelloworldls94q-1-3667110102-3817500168',
        name: 'job-cloneofhelloworldls94q-1-3667110102.B',
        displayName: 'B',
        type: 'Pod',
        templateName: 'echo',
        phase: 'Succeeded',
        boundaryID: 'job-cloneofhelloworldls94q-1-3667110102',
        startedAt: '2018-10-23T05:56:10Z',
        finishedAt: '2018-10-23T05:56:12Z',
        inputs: {
          parameters: [
            {
              name: 'message',
              value: 'hello world from node: B',
            },
          ],
        },
        children: ['job-cloneofhelloworldls94q-1-3667110102-3918165882'],
      },
      'job-cloneofhelloworldls94q-1-3667110102-3834277787': {
        id: 'job-cloneofhelloworldls94q-1-3667110102-3834277787',
        name: 'job-cloneofhelloworldls94q-1-3667110102.C',
        displayName: 'C',
        type: 'Pod',
        templateName: 'echo',
        phase: 'Succeeded',
        boundaryID: 'job-cloneofhelloworldls94q-1-3667110102',
        startedAt: '2018-10-23T05:56:10Z',
        finishedAt: '2018-10-23T05:56:21Z',
        inputs: {
          parameters: [
            {
              name: 'message',
              value: 'hello world from node: C',
            },
          ],
        },
        children: ['job-cloneofhelloworldls94q-1-3667110102-3918165882'],
      },
      'job-cloneofhelloworldls94q-1-3667110102-3867833025': {
        id: 'job-cloneofhelloworldls94q-1-3667110102-3867833025',
        name: 'job-cloneofhelloworldls94q-1-3667110102.A',
        displayName: 'A',
        type: 'Pod',
        templateName: 'echo',
        phase: 'Succeeded',
        boundaryID: 'job-cloneofhelloworldls94q-1-3667110102',
        startedAt: '2018-10-23T05:56:07Z',
        finishedAt: '2018-10-23T05:56:09Z',
        inputs: {
          parameters: [
            {
              name: 'message',
              value: 'hello world from node: A',
            },
          ],
        },
        children: [
          'job-cloneofhelloworldls94q-1-3667110102-3817500168',
          'job-cloneofhelloworldls94q-1-3667110102-3834277787',
        ],
      },
      'job-cloneofhelloworldls94q-1-3667110102-3918165882': {
        id: 'job-cloneofhelloworldls94q-1-3667110102-3918165882',
        name: 'job-cloneofhelloworldls94q-1-3667110102.D',
        displayName: 'D',
        type: 'Pod',
        templateName: 'echo',
        phase: 'Succeeded',
        boundaryID: 'job-cloneofhelloworldls94q-1-3667110102',
        startedAt: '2018-10-23T05:56:22Z',
        finishedAt: '2018-10-23T05:56:23Z',
        inputs: {
          parameters: [
            {
              name: 'message',
              value: 'hello world from node: D',
            },
          ],
        },
      },
    },
  },
};
