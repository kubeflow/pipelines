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
import i18next from 'i18next';
export default {
  metadata: {
    name: 'xgboost-training-gzkm9',
    generateName: 'xgboost-training-',
    namespace: 'default',
    selfLink: '/apis/argoproj.io/v1alpha1/namespaces/default/workflows/xgboost-training-gzkm9',
    uid: '2c007465-41cf-11e8-bba7-42010a8a0fc2',
    resourceVersion: '696668',
    creationTimestamp: '2018-04-16T23:37:48Z',
    labels: {
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
    },
  },
  spec: {
    templates: [
      {
        name: 'exit-handler',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
          ],
        },
        outputs: {},
        metadata: {},
        steps: [
          [
            {
              name: 'deletecluster',
              template: 'deletecluster',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'name',
                    value: '{{inputs.parameters.cluster}}',
                  },
                ],
              },
            },
          ],
        ],
      },
      {
        name: 'xgboost-training',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
            {
              name: 'output',
            },
            {
              name: 'train',
            },
            {
              name: 'eval',
            },
            {
              name: 'schema',
            },
            {
              name: 'target',
            },
            {
              name: 'package',
            },
            {
              name: 'workers',
            },
            {
              name: 'rounds',
            },
            {
              name: 'conf',
            },
          ],
        },
        outputs: {},
        metadata: {},
        steps: [
          [
            {
              name: 'createcluster',
              template: 'createcluster',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'name',
                    value: '{{inputs.parameters.cluster}}',
                  },
                  {
                    name: 'staging',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/staging',
                  },
                ],
              },
            },
          ],
          [
            {
              name: 'analyze',
              template: 'analyze',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'cluster',
                    value: '{{inputs.parameters.cluster}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
                  },
                  {
                    name: 'train',
                    value: '{{inputs.parameters.train}}',
                  },
                  {
                    name: 'schema',
                    value: '{{inputs.parameters.schema}}',
                  },
                ],
              },
            },
          ],
          [
            {
              name: 'transform',
              template: 'transform',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'cluster',
                    value: '{{inputs.parameters.cluster}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/transform',
                  },
                  {
                    name: 'train',
                    value: '{{inputs.parameters.train}}',
                  },
                  {
                    name: 'eval',
                    value: '{{inputs.parameters.eval}}',
                  },
                  {
                    name: 'target',
                    value: '{{inputs.parameters.target}}',
                  },
                  {
                    name: 'analysis',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
                  },
                ],
              },
            },
          ],
          [
            {
              name: 'train',
              template: 'train',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'cluster',
                    value: '{{inputs.parameters.cluster}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/model',
                  },
                  {
                    name: 'train',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/transform/train/part-*',
                  },
                  {
                    name: 'eval',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/transform/eval/part-*',
                  },
                  {
                    name: 'target',
                    value: '{{inputs.parameters.target}}',
                  },
                  {
                    name: 'analysis',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
                  },
                  {
                    name: 'package',
                    value: '{{inputs.parameters.package}}',
                  },
                  {
                    name: 'workers',
                    value: '{{inputs.parameters.workers}}',
                  },
                  {
                    name: 'rounds',
                    value: '{{inputs.parameters.rounds}}',
                  },
                  {
                    name: 'conf',
                    value: '{{inputs.parameters.conf}}',
                  },
                ],
              },
            },
          ],
          [
            {
              name: 'batchpredict',
              template: 'batchpredict',
              arguments: {
                parameters: [
                  {
                    name: 'project',
                    value: '{{inputs.parameters.project}}',
                  },
                  {
                    name: 'region',
                    value: '{{inputs.parameters.region}}',
                  },
                  {
                    name: 'cluster',
                    value: '{{inputs.parameters.cluster}}',
                  },
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict',
                  },
                  {
                    name: 'eval',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/transform/eval/part-*',
                  },
                  {
                    name: 'target',
                    value: '{{inputs.parameters.target}}',
                  },
                  {
                    name: 'analysis',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
                  },
                  {
                    name: 'package',
                    value: '{{inputs.parameters.package}}',
                  },
                  {
                    name: 'model',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/model',
                  },
                ],
              },
            },
          ],
          [
            {
              name: 'confusionmatrix',
              template: 'confusionmatrix',
              arguments: {
                parameters: [
                  {
                    name: 'output',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/confusionmatrix',
                  },
                  {
                    name: 'predictions',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/batchpredict/part-*.csv',
                  },
                  {
                    name: 'analysis',
                    value: '{{inputs.parameters.output}}/{{workflow.name}}/analysis',
                  },
                  {
                    name: 'target',
                    value: '{{inputs.parameters.target}}',
                  },
                ],
              },
            },
          ],
        ],
      },
      {
        name: 'createcluster',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'name',
            },
            {
              name: 'staging',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            'python /ml/create_cluster.py --project {{inputs.parameters.project}} ' +
              '--region {{inputs.parameters.region}} --name {{inputs.parameters.name}} ' +
              '--staging {{inputs.parameters.staging}}',
          ],
          resources: {},
        },
      },
      {
        name: 'analyze',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
            {
              name: 'output',
            },
            {
              name: 'train',
            },
            {
              name: 'schema',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            'python /ml/analyze.py --project {{inputs.parameters.project}} ' +
              '--region {{inputs.parameters.region}} --cluster ' +
              '{{inputs.parameters.cluster}} --output ' +
              '{{inputs.parameters.output}} --train {{inputs.parameters.train}} ' +
              '--schema {{inputs.parameters.schema}}',
          ],
          resources: {},
        },
      },
      {
        name: 'transform',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
            {
              name: 'output',
            },
            {
              name: 'train',
            },
            {
              name: 'eval',
            },
            {
              name: 'target',
            },
            {
              name: 'analysis',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            'python /ml/transform.py --project {{inputs.parameters.project}} ' +
              '--region {{inputs.parameters.region}} --cluster ' +
              '{{inputs.parameters.cluster}} --output ' +
              '{{inputs.parameters.output}} --train {{inputs.parameters.train}} ' +
              '--eval {{inputs.parameters.eval}} --target ' +
              '{{inputs.parameters.target}} --analysis ' +
              '{{inputs.parameters.analysis}}',
          ],
          resources: {},
        },
      },
      {
        name: 'train',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
            {
              name: 'output',
            },
            {
              name: 'train',
            },
            {
              name: 'eval',
            },
            {
              name: 'target',
            },
            {
              name: 'analysis',
            },
            {
              name: 'package',
            },
            {
              name: 'workers',
            },
            {
              name: 'rounds',
            },
            {
              name: 'conf',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            // tslint:disable-next-line:max-line-length
            'python /ml/train.py --project {{inputs.parameters.project}} --region {{inputs.parameters.region}} --cluster {{inputs.parameters.cluster}} --output {{inputs.parameters.output}} --train {{inputs.parameters.train}} --eval {{inputs.parameters.eval}} --target {{inputs.parameters.target}} --analysis {{inputs.parameters.analysis}} --package {{inputs.parameters.package}} --workers {{inputs.parameters.workers}} --rounds {{inputs.parameters.rounds}} --conf {{inputs.parameters.conf}}',
          ],
          resources: {},
        },
      },
      {
        name: 'batchpredict',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'cluster',
            },
            {
              name: 'output',
            },
            {
              name: 'eval',
            },
            {
              name: 'model',
            },
            {
              name: 'target',
            },
            {
              name: 'package',
            },
            {
              name: 'analysis',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            // tslint:disable-next-line:max-line-length
            'python /ml/predict.py --project {{inputs.parameters.project}} --region {{inputs.parameters.region}} --cluster {{inputs.parameters.cluster}} --output {{inputs.parameters.output}} --predict {{inputs.parameters.eval}} --analysis {{inputs.parameters.analysis}} --target {{inputs.parameters.target}} --model {{inputs.parameters.model}} --package {{inputs.parameters.package}} ',
          ],
          resources: {},
        },
      },
      {
        name: 'confusionmatrix',
        inputs: {
          parameters: [
            {
              name: 'output',
            },
            {
              name: 'analysis',
            },
            {
              name: 'predictions',
            },
            {
              name: 'target',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-local',
          command: ['sh', '-c'],
          args: [
            // tslint:disable-next-line:max-line-length
            'python /ml/confusion_matrix.py --output {{inputs.parameters.output}} --predictions {{inputs.parameters.predictions}} --analysis {{inputs.parameters.analysis}} --target {{inputs.parameters.target}}',
          ],
          resources: {},
        },
      },
      {
        name: 'deletecluster',
        inputs: {
          parameters: [
            {
              name: 'project',
            },
            {
              name: 'region',
            },
            {
              name: 'name',
            },
          ],
        },
        outputs: {},
        metadata: {},
        container: {
          name: '',
          image: 'gcr.io/ml-pipeline/ml-pipeline-dataproc-xgboost',
          command: ['sh', '-c'],
          args: [
            // tslint:disable-next-line:max-line-length
            'python /ml/delete_cluster.py --project {{inputs.parameters.project}} --region {{inputs.parameters.region}} --name {{inputs.parameters.name}}',
          ],
          resources: {},
        },
      },
    ],
    entrypoint: 'xgboost-training',
    arguments: {
      parameters: [
        {
          name: 'project',
          value: 'ml-pipeline',
        },
        {
          name: 'region',
          value: 'us-central1',
        },
        {
          name: 'cluster',
          value: 'xgboost-spark-{{workflow.name}}',
        },
        {
          name: 'output',
          value: 'gs://sample-xgbbost-cm-output',
        },
        {
          name: 'train',
          value: 'gs://ml-pipeline-playground/newsgroup/train.csv',
        },
        {
          name: 'eval',
          value: 'gs://ml-pipeline-playground/newsgroup/eval.csv',
        },
        {
          name: 'schema',
          value: 'gs://ml-pipeline-playground/newsgroup/schema.json',
        },
        {
          name: 'target',
          value: 'news_label',
        },
        {
          name: 'package',
          // tslint:disable-next-line:max-line-length
          value:
            'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
        },
        {
          name: 'workers',
          value: '2',
        },
        {
          name: 'rounds',
          value: '200',
        },
        {
          name: 'conf',
          value: 'gs://ml-pipeline-playground/trainconfcla.json ',
        },
      ],
    },
    onExit: 'exit-handler',
  },
  status: {
    phase: 'Succeeded',
    startedAt: '2018-04-16T23:37:48Z',
    finishedAt: '2018-04-17T00:12:01Z',
    nodes: {
      'xgboost-training-gzkm9': {
        id: 'xgboost-training-gzkm9',
        name: 'xgboost-training-gzkm9',
        displayName: 'xgboost-training-gzkm9',
        type: 'Steps',
        templateName: 'xgboost-training',
        phase: 'Succeeded',
        startedAt: '2018-04-16T23:37:48Z',
        finishedAt: '2018-04-17T00:10:06Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'output',
              value: 'gs://sample-xgbbost-cm-output',
            },
            {
              name: 'train',
              value: 'gs://ml-pipeline-playground/newsgroup/train.csv',
            },
            {
              name: 'eval',
              value: 'gs://ml-pipeline-playground/newsgroup/eval.csv',
            },
            {
              name: 'schema',
              value: 'gs://ml-pipeline-playground/newsgroup/schema.json',
            },
            {
              name: 'target',
              value: 'news_label',
            },
            {
              name: 'package',
              // tslint:disable-next-line:max-line-length
              value:
                'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            },
            {
              name: 'workers',
              value: '2',
            },
            {
              name: 'rounds',
              value: '200',
            },
            {
              name: 'conf',
              value: 'gs://ml-pipeline-playground/trainconfcla.json ',
            },
          ],
        },
        children: [
          'xgboost-training-gzkm9-4204210601',
          'xgboost-training-gzkm9-916047540',
          'xgboost-training-gzkm9-915503087',
          'xgboost-training-gzkm9-982760658',
          'xgboost-training-gzkm9-4204798981',
          'xgboost-training-gzkm9-916635920',
        ],
        outboundNodes: ['xgboost-training-gzkm9-2203328319'],
      },
      'xgboost-training-gzkm9-1253553084': {
        id: 'xgboost-training-gzkm9-1253553084',
        name: 'xgboost-training-gzkm9.onExit',
        displayName: 'xgboost-training-gzkm9.onExit',
        type: 'Steps',
        templateName: 'exit-handler',
        phase: 'Pending',
        message:
          'ImagePullBackOff: Back-off pulling image "gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster"',
        startedAt: '2018-04-17T00:10:06Z',
        finishedAt: '2018-04-17T00:12:01Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
          ],
        },
        children: ['xgboost-training-gzkm9-3439262870'],
        outboundNodes: ['xgboost-training-gzkm9-3721733163'],
      },
      'xgboost-training-gzkm9-1761585008': {
        id: 'xgboost-training-gzkm9-1761585008',
        name: 'xgboost-training-gzkm9[4].batchpredict',
        displayName: 'batchpredict',
        type: 'Pod',
        templateName: 'batchpredict',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-17T00:08:06Z',
        finishedAt: '2018-04-17T00:08:59Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'output',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/batchpredict',
            },
            {
              name: 'eval',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/transform/eval/part-*',
            },
            {
              name: 'model',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/model',
            },
            {
              name: 'target',
              value: 'news_label',
            },
            {
              name: 'package',
              // tslint:disable-next-line:max-line-length
              value:
                'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            },
            {
              name: 'analysis',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/analysis',
            },
          ],
        },
        children: ['xgboost-training-gzkm9-916635920'],
      },
      'xgboost-training-gzkm9-2203328319': {
        id: 'xgboost-training-gzkm9-2203328319',
        name: 'xgboost-training-gzkm9[5].confusionmatrix',
        displayName: 'confusionmatrix',
        type: 'Pod',
        templateName: 'confusionmatrix',
        phase: 'Failed',
        message: 'Mock backend could not schedule pod; no k8s implementation in Javascript (yet).',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-17T00:08:59Z',
        finishedAt: '2018-04-17T00:10:06Z',
        inputs: {
          parameters: [
            {
              name: 'analysis',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/analysis',
            },
            {
              name: 'predictions',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/batchpredict/part-*.csv',
            },
            {
              name: 'target',
              value: 'news_label',
            },
          ],
        },
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'confusionmatrix',
              },
            },
          ],
        },
      },
      'xgboost-training-gzkm9-2365787662': {
        id: 'xgboost-training-gzkm9-2365787662',
        name: 'xgboost-training-gzkm9[3].train',
        displayName: 'train',
        type: 'Pod',
        templateName: 'train',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:49:28Z',
        finishedAt: '2018-04-17T00:08:05Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'train',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/transform/train/part-*',
            },
            {
              name: 'eval',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/transform/eval/part-*',
            },
            {
              name: 'target',
              value: 'news_label',
            },
            {
              name: 'analysis',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/analysis',
            },
            {
              name: 'package',
              // tslint:disable-next-line:max-line-length
              value:
                'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            },
            {
              name: 'workers',
              value: '2',
            },
            {
              name: 'rounds',
              value: '200',
            },
            {
              name: 'conf',
              value: 'gs://ml-pipeline-playground/trainconfcla.json ',
            },
          ],
        },
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'model',
              },
            },
          ],
        },
        children: ['xgboost-training-gzkm9-4204798981'],
      },
      'xgboost-training-gzkm9-2411879589': {
        id: 'xgboost-training-gzkm9-2411879589',
        name: 'xgboost-training-gzkm9[0].createcluster',
        displayName: 'createcluster',
        type: 'Pod',
        templateName: 'createcluster',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:37:49Z',
        finishedAt: '2018-04-16T23:39:56Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'name',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'staging',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/staging',
            },
          ],
        },
        children: ['xgboost-training-gzkm9-916047540'],
      },
      'xgboost-training-gzkm9-2457131397': {
        id: 'xgboost-training-gzkm9-2457131397',
        name: 'xgboost-training-gzkm9[2].transform',
        displayName: 'transform',
        type: 'Pod',
        templateName: 'transform',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:41:24Z',
        finishedAt: '2018-04-16T23:49:27Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'train',
              value: 'gs://ml-pipeline-playground/newsgroup/train.csv',
            },
            {
              name: 'eval',
              value: 'gs://ml-pipeline-playground/newsgroup/eval.csv',
            },
            {
              name: 'target',
              value: 'news_label',
            },
            {
              name: 'analysis',
              value: 'gs://sample-xgbbost-cm-output/xgboost-training-gzkm9/analysis',
            },
          ],
        },
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
        children: ['xgboost-training-gzkm9-982760658'],
      },
      'xgboost-training-gzkm9-3439262870': {
        id: 'xgboost-training-gzkm9-3439262870',
        name: 'xgboost-training-gzkm9.onExit[0]',
        displayName: '[0]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9-1253553084',
        startedAt: '2018-04-17T00:10:06Z',
        finishedAt: '2018-04-17T00:12:01Z',
        children: ['xgboost-training-gzkm9-3721733163'],
      },
      'xgboost-training-gzkm9-3636935406': {
        id: 'xgboost-training-gzkm9-3636935406',
        name: 'xgboost-training-gzkm9[1].analyze',
        displayName: 'analyze',
        type: 'Pod',
        templateName: 'analyze',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:39:56Z',
        finishedAt: '2018-04-16T23:41:24Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'cluster',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
            {
              name: 'train',
              value: 'gs://ml-pipeline-playground/newsgroup/train.csv',
            },
            {
              name: 'schema',
              value: 'gs://ml-pipeline-playground/newsgroup/schema.json',
            },
          ],
        },
        outputs: {
          artifacts: [
            {
              name: 'mlpipeline-ui-metadata',
              s3: {
                bucket: 'somebucket',
                key: 'analysis',
              },
            },
          ],
        },
        children: ['xgboost-training-gzkm9-915503087'],
      },
      'xgboost-training-gzkm9-3721733163': {
        id: 'xgboost-training-gzkm9-3721733163',
        name: 'xgboost-training-gzkm9.onExit[0].deletecluster',
        displayName: 'deletecluster',
        type: 'Pod',
        templateName: 'deletecluster',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9-1253553084',
        startedAt: '2018-04-17T00:10:06Z',
        finishedAt: '2018-04-17T00:12:01Z',
        inputs: {
          parameters: [
            {
              name: 'project',
              value: 'ml-pipeline',
            },
            {
              name: 'region',
              value: 'us-central1',
            },
            {
              name: 'name',
              value: 'xgboost-spark-xgboost-training-gzkm9',
            },
          ],
        },
      },
      'xgboost-training-gzkm9-4204210601': {
        id: 'xgboost-training-gzkm9-4204210601',
        name: 'xgboost-training-gzkm9[0]',
        displayName: '[0]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:37:48Z',
        finishedAt: '2018-04-16T23:39:56Z',
        children: ['xgboost-training-gzkm9-2411879589'],
      },
      'xgboost-training-gzkm9-4204798981': {
        id: 'xgboost-training-gzkm9-4204798981',
        name: 'xgboost-training-gzkm9[4]',
        displayName: '[4]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-17T00:08:06Z',
        finishedAt: '2018-04-17T00:08:59Z',
        children: ['xgboost-training-gzkm9-1761585008'],
      },
      'xgboost-training-gzkm9-915503087': {
        id: 'xgboost-training-gzkm9-915503087',
        name: 'xgboost-training-gzkm9[2]',
        displayName: '[2]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:41:24Z',
        finishedAt: '2018-04-16T23:49:28Z',
        children: ['xgboost-training-gzkm9-2457131397'],
      },
      'xgboost-training-gzkm9-916047540': {
        id: 'xgboost-training-gzkm9-916047540',
        name: 'xgboost-training-gzkm9[1]',
        displayName: '[1]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:39:56Z',
        finishedAt: '2018-04-16T23:41:24Z',
        children: ['xgboost-training-gzkm9-3636935406'],
      },
      'xgboost-training-gzkm9-916635920': {
        id: 'xgboost-training-gzkm9-916635920',
        name: 'xgboost-training-gzkm9[5]',
        displayName: '[5]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-17T00:08:59Z',
        finishedAt: '2018-04-17T00:10:06Z',
        children: ['xgboost-training-gzkm9-2203328319'],
      },
      'xgboost-training-gzkm9-982760658': {
        id: 'xgboost-training-gzkm9-982760658',
        name: 'xgboost-training-gzkm9[3]',
        displayName: '[3]',
        type: 'StepGroup',
        phase: 'Succeeded',
        boundaryID: 'xgboost-training-gzkm9',
        startedAt: '2018-04-16T23:49:28Z',
        finishedAt: '2018-04-17T00:08:06Z',
        children: ['xgboost-training-gzkm9-2365787662'],
      },
    },
  },
};
