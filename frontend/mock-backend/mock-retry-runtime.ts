// Copyright 2020 Google LLC
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
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'Workflow',
  metadata: {
    annotations: {
      'pipelines.kubeflow.org/kfp_sdk_version': '0.5.1',
      'pipelines.kubeflow.org/pipeline_compilation_time': '2020-09-08T16:17:43.250289',
      'pipelines.kubeflow.org/pipeline_spec':
        '{"description": "The pipeline that include a retry step.", "name": "Retry random failures"}',
      'pipelines.kubeflow.org/run_name': 'Run of retry (835b8)',
    },
    creationTimestamp: '2020-09-08T08:19:12Z',
    generateName: 'retry-random-failures-',
    generation: 15,
    labels: {
      'pipeline/persistedFinalState': 'true',
      'pipeline/runid': '2f9a3d50-454c-4979-be8d-f222bc245bc1',
      'pipelines.kubeflow.org/kfp_sdk_version': '0.5.1',
      'workflows.argoproj.io/completed': 'true',
      'workflows.argoproj.io/phase': 'Succeeded',
    },
    name: 'retry-random-failures-p9snt',
    namespace: 'kubeflow',
    resourceVersion: '131647',
    selfLink:
      '/apis/argoproj.io/v1alpha1/namespaces/kubeflow/workflows/retry-random-failures-p9snt',
    uid: '3b673760-f297-42d4-a55d-32e7beba394a',
  },
  spec: {
    arguments: {},
    entrypoint: 'retry-random-failures',
    serviceAccountName: 'pipeline-runner',
    templates: [
      {
        arguments: {},
        container: {
          args: ['import sys; print(sys.argv[1])', 'some-op-1'],
          command: ['python', '-c'],
          image: 'python:alpine3.6',
          name: '',
          resources: {},
        },
        inputs: {},
        metadata: {
          annotations: {
            'sidecar.istio.io/inject': 'false',
          },
          labels: {
            'pipelines.kubeflow.org/cache_enabled': 'true',
            'pipelines.kubeflow.org/pipeline-sdk-type': 'kfp',
          },
        },
        name: 'print-msg',
        outputs: {},
      },
      {
        arguments: {},
        container: {
          args: ['import sys; print(sys.argv[1])', 'some-op-3'],
          command: ['python', '-c'],
          image: 'python:alpine3.6',
          name: '',
          resources: {},
        },
        inputs: {},
        metadata: {
          annotations: {
            'sidecar.istio.io/inject': 'false',
          },
          labels: {
            'pipelines.kubeflow.org/cache_enabled': 'true',
            'pipelines.kubeflow.org/pipeline-sdk-type': 'kfp',
          },
        },
        name: 'print-msg-2',
        outputs: {},
      },
      {
        arguments: {},
        container: {
          args: [
            'import random; import sys; exit_code = int(random.choice(sys.argv[1].split(","))); print(exit_code); sys.exit(exit_code)',
            '0,1,2,3',
          ],
          command: ['python', '-c'],
          image: 'python:alpine3.6',
          name: '',
          resources: {},
        },
        inputs: {},
        metadata: {
          annotations: {
            'sidecar.istio.io/inject': 'false',
          },
          labels: {
            'pipelines.kubeflow.org/cache_enabled': 'true',
            'pipelines.kubeflow.org/pipeline-sdk-type': 'kfp',
          },
        },
        name: 'random-failure',
        outputs: {},
        retryStrategy: {
          limit: 3,
        },
      },
      {
        arguments: {},
        dag: {
          tasks: [
            {
              arguments: {},
              name: 'print-msg',
              template: 'print-msg',
            },
            {
              arguments: {},
              dependencies: ['random-failure'],
              name: 'print-msg-2',
              template: 'print-msg-2',
            },
            {
              arguments: {},
              dependencies: ['print-msg'],
              name: 'random-failure',
              template: 'random-failure',
            },
          ],
        },
        inputs: {},
        metadata: {
          annotations: {
            'sidecar.istio.io/inject': 'false',
          },
          labels: {
            'pipelines.kubeflow.org/cache_enabled': 'true',
          },
        },
        name: 'retry-random-failures',
        outputs: {},
      },
    ],
  },
  status: {
    conditions: [
      {
        status: 'True',
        type: 'Completed',
      },
    ],
    finishedAt: '2020-09-08T08:19:25Z',
    nodes: {
      'retry-random-failures-p9snt': {
        children: ['retry-random-failures-p9snt-3292940923'],
        displayName: 'retry-random-failures-p9snt',
        finishedAt: '2020-09-08T08:19:25Z',
        id: 'retry-random-failures-p9snt',
        name: 'retry-random-failures-p9snt',
        outboundNodes: ['retry-random-failures-p9snt-1189960944'],
        phase: 'Succeeded',
        startedAt: '2020-09-08T08:19:12Z',
        templateName: 'retry-random-failures',
        type: 'DAG',
      },
      'retry-random-failures-p9snt-1189960944': {
        boundaryID: 'retry-random-failures-p9snt',
        displayName: 'print-msg-2',
        finishedAt: '2020-09-08T08:19:23Z',
        id: 'retry-random-failures-p9snt-1189960944',
        name: 'retry-random-failures-p9snt.print-msg-2',
        outputs: {
          artifacts: [
            {
              archiveLogs: true,
              name: 'main-logs',
              s3: {
                accessKeySecret: {
                  key: 'accesskey',
                  name: 'mlpipeline-minio-artifact',
                },
                bucket: 'mlpipeline',
                endpoint: 'minio-service.kubeflow:9000',
                insecure: true,
                key:
                  'artifacts/retry-random-failures-p9snt/retry-random-failures-p9snt-1189960944/main.log',
                secretKeySecret: {
                  key: 'secretkey',
                  name: 'mlpipeline-minio-artifact',
                },
              },
            },
          ],
        },
        phase: 'Succeeded',
        startedAt: '2020-09-08T08:19:21Z',
        templateName: 'print-msg-2',
        type: 'Pod',
      },
      'retry-random-failures-p9snt-310421839': {
        boundaryID: 'retry-random-failures-p9snt',
        children: ['retry-random-failures-p9snt-1189960944'],
        displayName: 'random-failure(0)',
        finishedAt: '2020-09-08T08:19:18Z',
        id: 'retry-random-failures-p9snt-310421839',
        name: 'retry-random-failures-p9snt.random-failure(0)',
        outputs: {
          artifacts: [
            {
              archiveLogs: true,
              name: 'main-logs',
              s3: {
                accessKeySecret: {
                  key: 'accesskey',
                  name: 'mlpipeline-minio-artifact',
                },
                bucket: 'mlpipeline',
                endpoint: 'minio-service.kubeflow:9000',
                insecure: true,
                key:
                  'artifacts/retry-random-failures-p9snt/retry-random-failures-p9snt-310421839/main.log',
                secretKeySecret: {
                  key: 'secretkey',
                  name: 'mlpipeline-minio-artifact',
                },
              },
            },
          ],
        },
        phase: 'Succeeded',
        startedAt: '2020-09-08T08:19:16Z',
        templateName: 'random-failure',
        type: 'Pod',
      },
      'retry-random-failures-p9snt-3193641316': {
        boundaryID: 'retry-random-failures-p9snt',
        children: ['retry-random-failures-p9snt-310421839'],
        displayName: 'random-failure',
        finishedAt: '2020-09-08T08:19:20Z',
        id: 'retry-random-failures-p9snt-3193641316',
        name: 'retry-random-failures-p9snt.random-failure',
        outputs: {
          artifacts: [
            {
              archiveLogs: true,
              name: 'main-logs',
              s3: {
                accessKeySecret: {
                  key: 'accesskey',
                  name: 'mlpipeline-minio-artifact',
                },
                bucket: 'mlpipeline',
                endpoint: 'minio-service.kubeflow:9000',
                insecure: true,
                key:
                  'artifacts/retry-random-failures-p9snt/retry-random-failures-p9snt-310421839/main.log',
                secretKeySecret: {
                  key: 'secretkey',
                  name: 'mlpipeline-minio-artifact',
                },
              },
            },
          ],
        },
        phase: 'Succeeded',
        startedAt: '2020-09-08T08:19:16Z',
        templateName: 'random-failure',
        type: 'Retry',
      },
      'retry-random-failures-p9snt-3292940923': {
        boundaryID: 'retry-random-failures-p9snt',
        children: ['retry-random-failures-p9snt-3193641316'],
        displayName: 'print-msg',
        finishedAt: '2020-09-08T08:19:14Z',
        id: 'retry-random-failures-p9snt-3292940923',
        name: 'retry-random-failures-p9snt.print-msg',
        outputs: {
          artifacts: [
            {
              archiveLogs: true,
              name: 'main-logs',
              s3: {
                accessKeySecret: {
                  key: 'accesskey',
                  name: 'mlpipeline-minio-artifact',
                },
                bucket: 'mlpipeline',
                endpoint: 'minio-service.kubeflow:9000',
                insecure: true,
                key:
                  'artifacts/retry-random-failures-p9snt/retry-random-failures-p9snt-3292940923/main.log',
                secretKeySecret: {
                  key: 'secretkey',
                  name: 'mlpipeline-minio-artifact',
                },
              },
            },
          ],
        },
        phase: 'Succeeded',
        startedAt: '2020-09-08T08:19:12Z',
        templateName: 'print-msg',
        type: 'Pod',
      },
    },
    phase: 'Succeeded',
    startedAt: '2020-09-08T08:19:12Z',
  },
};
