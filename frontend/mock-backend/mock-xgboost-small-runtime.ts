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
    name: 'job-xgboosttrainingm7t2r-1-2537408167',
    namespace: 'default',
    selfLink:
      '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/job-xgboosttrainingm7t2r-1-2537408167',
    uid: '3333210c-cdef-11e8-8c17-42010a8a0078',
    resourceVersion: '24210',
    creationTimestamp: '2018-10-12T07:19:47Z',
    labels: {
      'scheduledworkflows.kubeflow.org/isOwnedByScheduledWorkflow': 'true',
      'scheduledworkflows.kubeflow.org/scheduledWorkflowName': 'job-xgboosttrainingm7t2r',
      'scheduledworkflows.kubeflow.org/workflowEpoch': '1539328777',
      'scheduledworkflows.kubeflow.org/workflowIndex': '1',
      'workflows.argoproj.io/phase': 'Running',
    },
    ownerReferences: [
      {
        apiVersion: 'kubeflow.org/v1beta1',
        kind: 'ScheduledWorkflow',
        name: 'job-xgboosttrainingm7t2r',
        uid: '2d3b0ed1-cdef-11e8-8c17-42010a8a0078',
        controller: true,
        blockOwnerDeletion: true,
      },
    ],
  },
  spec: {
    templates: [
      {
        name: 'analyze',
        inputs: {
          parameters: [
            {
              name: 'create-cluster-output',
            },
            {
              name: 'output',
            },
            {
              name: 'project',
            },
          ],
        },
        outputs: {
          parameters: [
            {
              name: 'analyze-output',
              valueFrom: {
                path: '/output.txt',
              },
            },
          ],
        },
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-analyze',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--cluster',
            '{{inputs.parameters.create-cluster-output}}',
            '--schema',
            'gs://ml-pipeline-playground/sfpd/schema.json',
            '--train',
            'gs://ml-pipeline-playground/sfpd/train.csv',
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
          ],
          resources: {},
        },
      },
      {
        name: 'confusion-matrix',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'predict-output',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix',
          args: [
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/confusionmatrix',
            '--predictions',
            '{{inputs.parameters.predict-output}}',
          ],
          resources: {},
        },
      },
      {
        name: 'create-cluster',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'project',
            },
          ],
        },
        outputs: {
          parameters: [
            {
              name: 'create-cluster-output',
              valueFrom: {
                path: '/output.txt',
              },
            },
          ],
        },
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--name',
            'xgb-{{workflow.name}}',
            '--staging',
            '{{inputs.parameters.output}}',
          ],
          resources: {},
        },
      },
      {
        name: 'delete-cluster',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--name',
            'xgb-{{workflow.name}}',
          ],
          resources: {},
        },
      },
      {
        name: 'exit-handler-1',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'project',
            },
          ],
        },
        outputs: {},
        metadata: {},
        dag: {
          tasks: [
            {
              name: 'analyze',
              template: 'analyze',
              arguments: {
                parameters: [
                  {
                    name: 'create-cluster-output',
                    value: '{{tasks.create-cluster.outputs.parameters.create-cluster-output}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                ],
              },
              dependencies: ['create-cluster'],
            },
            {
              name: 'confusion-matrix',
              template: 'confusion-matrix',
              arguments: {
                parameters: [
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'predict-output',
                    value: '{{tasks.predict.outputs.parameters.predict-output}}',
                  },
                ],
              },
              dependencies: ['predict'],
            },
            {
              name: 'create-cluster',
              template: 'create-cluster',
              arguments: {
                parameters: [
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                ],
              },
            },
            {
              name: 'predict',
              template: 'predict',
              arguments: {
                parameters: [
                  {
                    name: 'analyze-output',
                    value: '{{tasks.analyze.outputs.parameters.analyze-output}}',
                  },
                  {
                    name: 'create-cluster-output',
                    value: '{{tasks.create-cluster.outputs.parameters.create-cluster-output}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'train-output',
                    value: '{{tasks.train.outputs.parameters.train-output}}',
                  },
                  {
                    name: 'transform-eval',
                    value: '{{tasks.transform.outputs.parameters.transform-eval}}',
                  },
                ],
              },
              dependencies: ['analyze', 'create-cluster', 'train', 'transform'],
            },
            {
              name: 'roc',
              template: 'roc',
              arguments: {
                parameters: [
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'predict-output',
                    value: '{{tasks.predict.outputs.parameters.predict-output}}',
                  },
                ],
              },
              dependencies: ['predict'],
            },
            {
              name: 'train',
              template: 'train',
              arguments: {
                parameters: [
                  {
                    name: 'analyze-output',
                    value: '{{tasks.analyze.outputs.parameters.analyze-output}}',
                  },
                  {
                    name: 'create-cluster-output',
                    value: '{{tasks.create-cluster.outputs.parameters.create-cluster-output}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'transform-eval',
                    value: '{{tasks.transform.outputs.parameters.transform-eval}}',
                  },
                  {
                    name: 'transform-train',
                    value: '{{tasks.transform.outputs.parameters.transform-train}}',
                  },
                ],
              },
              dependencies: ['analyze', 'create-cluster', 'transform'],
            },
            {
              name: 'transform',
              template: 'transform',
              arguments: {
                parameters: [
                  {
                    name: 'analyze-output',
                    value: '{{tasks.analyze.outputs.parameters.analyze-output}}',
                  },
                  {
                    name: 'create-cluster-output',
                    value: '{{tasks.create-cluster.outputs.parameters.create-cluster-output}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                ],
              },
              dependencies: ['analyze', 'create-cluster'],
            },
          ],
        },
      },
      {
        name: 'predict',
        inputs: {
          parameters: [
            {
              name: 'analyze-output',
            },
            {
              name: 'create-cluster-output',
            },
            {
              name: 'output',
            },
            {
              name: 'project',
            },
            {
              name: 'train-output',
            },
            {
              name: 'transform-eval',
            },
          ],
        },
        outputs: {
          parameters: [
            {
              name: 'predict-output',
              valueFrom: {
                path: '/output.txt',
              },
            },
          ],
        },
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-predict',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--cluster',
            '{{inputs.parameters.create-cluster-output}}',
            '--predict',
            '{{inputs.parameters.transform-eval}}',
            '--analysis',
            '{{inputs.parameters.analyze-output}}',
            '--target',
            'resolution',
            '--package',
            'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            '--model',
            '{{inputs.parameters.train-output}}',
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/predict',
          ],
          resources: {},
        },
      },
      {
        name: 'roc',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'predict-output',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-local-roc',
          args: [
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/roc',
            '--predictions',
            '{{inputs.parameters.predict-output}}',
            '--trueclass',
            'ACTION',
          ],
          resources: {},
        },
      },
      {
        name: 'train',
        inputs: {
          parameters: [
            {
              name: 'analyze-output',
            },
            {
              name: 'create-cluster-output',
            },
            {
              name: 'output',
            },
            {
              name: 'project',
            },
            {
              name: 'transform-eval',
            },
            {
              name: 'transform-train',
            },
          ],
        },
        outputs: {
          parameters: [
            {
              name: 'train-output',
              valueFrom: {
                path: '/output.txt',
              },
            },
          ],
        },
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-train',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--cluster',
            '{{inputs.parameters.create-cluster-output}}',
            '--train',
            '{{inputs.parameters.transform-train}}',
            '--eval',
            '{{inputs.parameters.transform-eval}}',
            '--analysis',
            '{{inputs.parameters.analyze-output}}',
            '--target',
            'resolution',
            '--package',
            'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            '--workers',
            '2',
            '--rounds',
            '200',
            '--conf',
            'gs://ml-pipeline-playground/trainconfcla.json',
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/model',
          ],
          resources: {},
        },
      },
      {
        name: 'transform',
        inputs: {
          parameters: [
            {
              name: 'analyze-output',
            },
            {
              name: 'create-cluster-output',
            },
            {
              name: 'output',
            },
            {
              name: 'project',
            },
          ],
        },
        outputs: {
          parameters: [
            {
              name: 'transform-eval',
              valueFrom: {
                path: '/output_eval.txt',
              },
            },
            {
              name: 'transform-train',
              valueFrom: {
                path: '/output_train.txt',
              },
            },
          ],
        },
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-transform',
          args: [
            '--project',
            '{{inputs.parameters.project}}',
            '--region',
            'us-central1',
            '--cluster',
            '{{inputs.parameters.create-cluster-output}}',
            '--train',
            'gs://ml-pipeline-playground/sfpd/train.csv',
            '--eval',
            'gs://ml-pipeline-playground/sfpd/eval.csv',
            '--analysis',
            '{{inputs.parameters.analyze-output}}',
            '--target',
            'resolution',
            '--output',
            '{{inputs.parameters.output}}/{{workflow.name}}/transform',
          ],
          resources: {},
        },
      },
      {
        name: 'xgboosttrainer',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'project',
            },
          ],
        },
        outputs: {},
        metadata: {},
        dag: {
          tasks: [
            {
              name: 'exit-handler-1',
              template: 'exit-handler-1',
              arguments: {
                parameters: [
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}',
                  },
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                ],
              },
            },
          ],
        },
      },
    ],
    entrypoint: 'xgboosttrainer',
    arguments: {
      parameters: [
        {
          name: 'output',
          value: 'gs://yelsayed-2/xgboost',
        },
        {
          name: 'project',
          value: 'yelsayed-2',
        },
        {
          name: 'region',
          value: 'us-central1',
        },
        {
          name: 'train-data',
          value: 'gs://ml-pipeline-playground/sfpd/train.csv',
        },
        {
          name: 'eval-data',
          value: 'gs://ml-pipeline-playground/sfpd/eval.csv',
        },
        {
          name: 'schema',
          value: 'gs://ml-pipeline-playground/sfpd/schema.json',
        },
        {
          name: 'target',
          value: 'resolution',
        },
        {
          name: 'rounds',
          value: '200',
        },
        {
          name: 'workers',
          value: '2',
        },
        {
          name: 'true-label',
          value: 'ACTION',
        },
      ],
    },
    onExit: 'delete-cluster',
  },
  status: {
    phase: 'Running',
    startedAt: '2018-10-12T07:19:47Z',
    finishedAt: null,
    nodes: {
      'job-xgboosttrainingm7t2r-1-2537408167': {
        id: 'job-xgboosttrainingm7t2r-1-2537408167',
        name: 'job-xgboosttrainingm7t2r-1-2537408167',
        displayName: 'job-xgboosttrainingm7t2r-1-2537408167',
        type: 'DAG',
        templateName: 'xgboosttrainer',
        phase: 'Running',
        startedAt: '2018-10-12T07:19:47Z',
        finishedAt: null,
        inputs: {
          parameters: [
            {
              name: 'output',
              value: 'gs://yelsayed-2/xgboost',
            },
            {
              name: 'project',
              value: 'yelsayed-2',
            },
          ],
        },
        children: ['job-xgboosttrainingm7t2r-1-2537408167-3348277322'],
      },
      'job-xgboosttrainingm7t2r-1-2537408167-294182655': {
        id: 'job-xgboosttrainingm7t2r-1-2537408167-294182655',
        name: 'job-xgboosttrainingm7t2r-1-2537408167.exit-handler-1.create-cluster',
        displayName: 'create-cluster',
        type: 'Pod',
        templateName: 'create-cluster',
        phase: 'Pending',
        boundaryID: 'job-xgboosttrainingm7t2r-1-2537408167-3348277322',
        message:
          'ImagePullBackOff: Back-off pulling image "gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster"',
        startedAt: '2018-10-12T07:19:47Z',
        finishedAt: null,
        inputs: {
          parameters: [
            {
              name: 'output',
              value: 'gs://yelsayed-2/xgboost',
            },
            {
              name: 'project',
              value: 'yelsayed-2',
            },
          ],
        },
      },
      'job-xgboosttrainingm7t2r-1-2537408167-3348277322': {
        id: 'job-xgboosttrainingm7t2r-1-2537408167-3348277322',
        name: 'job-xgboosttrainingm7t2r-1-2537408167.exit-handler-1',
        displayName: 'exit-handler-1',
        type: 'DAG',
        templateName: 'exit-handler-1',
        phase: 'Running',
        boundaryID: 'job-xgboosttrainingm7t2r-1-2537408167',
        startedAt: '2018-10-12T07:19:47Z',
        finishedAt: null,
        inputs: {
          parameters: [
            {
              name: 'output',
              value: 'gs://yelsayed-2/xgboost',
            },
            {
              name: 'project',
              value: 'yelsayed-2',
            },
          ],
        },
        children: ['job-xgboosttrainingm7t2r-1-2537408167-294182655'],
      },
    },
  },
};
