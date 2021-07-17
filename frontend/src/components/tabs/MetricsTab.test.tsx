/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, waitFor } from '@testing-library/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import React from 'react';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import * as mlmdUtils from 'src/mlmd/MlmdUtils';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { Artifact, ArtifactType, Event, Execution, Value } from 'src/third_party/mlmd';
import { ConfusionMatrixConfig } from '../viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../viewers/MarkdownViewer';
import { ROCCurveConfig } from '../viewers/ROCCurve';
import { TensorboardViewerConfig } from '../viewers/Tensorboard';
import { PlotType } from '../viewers/Viewer';
import { MetricsTab } from './MetricsTab';

testBestPractices();

const namespace = 'kubeflow';

describe('MetricsTab common case', () => {
  beforeEach(() => {});
  it('has no metrics', async () => {
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText('There is no metrics artifact available in this step.'));
  });
  it('shows info banner when execution is in UNKNOWN', async () => {
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.UNKNOWN);
    const { getByText, queryByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task is in unknown state.'));
    await waitFor(() => queryByText('Task has not completed.') === null);
  });

  it('shows info banner when execution is in NEW', async () => {
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.NEW);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task has not completed.'));
  });

  it('shows info banner when execution is in RUNNING', async () => {
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockReturnValue(Promise.resolve([]));
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockReturnValue(Promise.resolve([]));

    const execution = buildBasicExecution().setLastKnownState(Execution.State.RUNNING);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    await waitFor(() => getByText('Task has not completed.'));
  });
});

describe('MetricsTab with confidenceMetrics', () => {
  it('shows ROC curve', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          list: [
            {
              confidenceThreshold: 2,
              falsePositiveRate: 0,
              recall: 0,
            },
            {
              confidenceThreshold: 1,
              falsePositiveRate: 0,
              recall: 0.33962264150943394,
            },
            {
              confidenceThreshold: 0.9,
              falsePositiveRate: 0,
              recall: 0.6037735849056604,
            },
          ],
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };

    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValueOnce([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValueOnce([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    getByText('Metrics is loading.');
    await waitFor(() => getByText('ROC Curve: metrics'));
  });

  it('shows error banner when confidenceMetric type is wrong', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          list: [
            {
              invalidType: 2,
              falsePositiveRate: 0,
              recall: 0,
            },
            {
              confidenceThreshold: 1,
              falsePositiveRate: 0,
              recall: 0.33962264150943394,
            },
          ],
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };

    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confidenceMetrics data format."));
  });

  it('shows error banner when confidenceMetric is not array', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          variable: 'i am not a list',
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };

    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confidenceMetrics data format."));
  });
});

describe('MetricsTab with confusionMatrix', () => {
  it('shows confusion matrix', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confusionMatrix',
      new Value().setStructValue(
        Struct.fromJavaScript({
          struct: {
            annotationSpecs: [
              { displayName: 'Setosa' },
              { displayName: 'Versicolour' },
              { displayName: 'Virginica' },
            ],
            rows: [{ row: [31, 0, 0] }, { row: [1, 8, 12] }, { row: [0, 0, 23] }],
          },
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValueOnce([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValueOnce([artifactType]);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Confusion Matrix: metrics'));
  });

  it('shows error banner when confusionMatrix type is wrong', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confusionMatrix',
      new Value().setStructValue(
        Struct.fromJavaScript({
          struct: {
            annotationSpecs: [
              { displayName: 'Setosa' },
              { displayName: 'Versicolour' },
              { displayName: 'Virginica' },
            ],
            rows: [{ row: 'I am not an array' }, { row: [1, 8, 12] }, { row: [0, 0, 23] }],
          },
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };

    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confusionMatrix data format."));
  });

  it("shows error banner when confusionMatrix annotationSpecs length doesn't match rows", async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifactType = buildClassificationMetricsArtifactType();
    const artifact = buildClassificationMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set(
      'confusionMatrix',
      new Value().setStructValue(
        Struct.fromJavaScript({
          struct: {
            annotationSpecs: [{ displayName: 'Setosa' }, { displayName: 'Versicolour' }],
            rows: [{ row: [31, 0, 0] }, { row: [1, 8, 12] }, { row: [0, 0, 23] }],
          },
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };

    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([artifactType]);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );

    await waitFor(() => getByText("Error in metrics artifact's confusionMatrix data format."));
  });
});

describe('MetricsTab with Scalar Metrics', () => {
  it('shows Scalar Metrics', async () => {
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const artifact = buildMetricsArtifact();
    artifact.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    artifact.getCustomPropertiesMap().set('double', new Value().setDoubleValue(123.456));
    artifact.getCustomPropertiesMap().set('int', new Value().setIntValue(123));
    artifact.getCustomPropertiesMap().set(
      'struct',
      new Value().setStructValue(
        Struct.fromJavaScript({
          struct: {
            field: 'a string value',
          },
        }),
      ),
    );
    const linkedArtifact = { event: new Event(), artifact: artifact };
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValueOnce([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValueOnce([buildMetricsArtifactType()]);
    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Scalar Metrics: metrics'));
    await waitFor(() => getByText('double'));
    await waitFor(() => getByText('int'));
    await waitFor(() => getByText('struct'));
  });
});

describe('MetricsTab with V1 metrics', () => {
  it('shows Confusion Matrix', async () => {
    const linkedArtifact = buildV1LinkedSytemArtifact();
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([buildSystemArtifactType()]);
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const confusionViewerConfigs: ConfusionMatrixConfig[] = [
      {
        axes: ['target', 'predicted'],
        data: [
          [100, 20, 5],
          [40, 200, 33],
          [0, 10, 200],
        ],
        labels: ['rose', 'lily', 'iris'],
        type: PlotType.CONFUSION_MATRIX,
      },
    ];
    jest
      .spyOn(OutputArtifactLoader, 'load')
      .mockReturnValue(Promise.resolve(confusionViewerConfigs));

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Confusion matrix'));
  });

  it('shows ROC Curve', async () => {
    const linkedArtifact = buildV1LinkedSytemArtifact();
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([buildSystemArtifactType()]);
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const rocViewerConfig: ROCCurveConfig[] = [
      {
        data: [
          { label: '0.999972701073', x: 0, y: 0.00265957446809 },
          { label: '0.996713876724', x: 0, y: 0.226063829787 },
        ],
        type: PlotType.ROC,
      },
    ];
    jest.spyOn(OutputArtifactLoader, 'load').mockReturnValue(Promise.resolve(rocViewerConfig));

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('ROC Curve'));
  });

  it('shows HTML view', async () => {
    const linkedArtifact = buildV1LinkedSytemArtifact();
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([buildSystemArtifactType()]);
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const htmlViewerConfig: HTMLViewerConfig[] = [
      { htmlContent: '<h1>Hello, World!</h1>', type: PlotType.WEB_APP },
    ];
    jest.spyOn(OutputArtifactLoader, 'load').mockResolvedValue(htmlViewerConfig);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Static HTML'));
  });

  it('shows Markdown view', async () => {
    const linkedArtifact = buildV1LinkedSytemArtifact();
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([buildSystemArtifactType()]);
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const markdownViewerConfig: MarkdownViewerConfig[] = [
      {
        markdownContent:
          '# Inline Markdown\n\n* [Kubeflow official doc](https://www.kubeflow.org/).\n',
        type: PlotType.MARKDOWN,
      },
    ];
    jest.spyOn(OutputArtifactLoader, 'load').mockResolvedValue(markdownViewerConfig);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Markdown'));
    await waitFor(() => getByText('Kubeflow official doc'));
  });

  it('shows Tensorboard view', async () => {
    const linkedArtifact = buildV1LinkedSytemArtifact();
    jest
      .spyOn(mlmdUtils, 'getOutputLinkedArtifactsInExecution')
      .mockResolvedValue([linkedArtifact]);
    jest.spyOn(mlmdUtils, 'getArtifactTypes').mockResolvedValue([buildSystemArtifactType()]);
    const execution = buildBasicExecution().setLastKnownState(Execution.State.COMPLETE);
    const tensorboardViewerConfig: TensorboardViewerConfig[] = [
      {
        type: PlotType.TENSORBOARD,
        url: 's3://mlpipeline/tensorboard/logs/8d0e4b0a-466a-4c56-8164-30ed20cbc4a0',
        namespace: 'kubeflow',
        image: 'gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest',
        podTemplateSpec: {
          spec: {
            containers: [
              {
                env: [
                  {
                    name: 'AWS_ACCESS_KEY_ID',
                    valueFrom: {
                      secretKeyRef: { name: 'mlpipeline-minio-artifact', key: 'accesskey' },
                    },
                  },
                  {
                    name: 'AWS_SECRET_ACCESS_KEY',
                    valueFrom: {
                      secretKeyRef: {
                        name: 'mlpipeline-minio-artifact',
                        key: 'secretkey',
                      },
                    },
                  },
                  { name: 'AWS_REGION', value: 'minio' },
                  { name: 'S3_ENDPOINT', value: 'minio-service:9000' },
                  { name: 'S3_USE_HTTPS', value: '0' },
                  { name: 'S3_VERIFY_SSL', value: '0' },
                ],
              },
            ],
          },
        },
      },
    ];
    jest.spyOn(OutputArtifactLoader, 'load').mockResolvedValue(tensorboardViewerConfig);

    const { getByText } = render(
      <CommonTestWrapper>
        <MetricsTab execution={execution} namespace={namespace}></MetricsTab>
      </CommonTestWrapper>,
    );
    getByText('Metrics is loading.');
    await waitFor(() => getByText('Start Tensorboard'));
  });
});

function buildBasicExecution() {
  const execution = new Execution();
  execution.setId(123);
  return execution;
}

function buildClassificationMetricsArtifactType() {
  const artifactType = new ArtifactType();
  artifactType.setName('system.ClassificationMetrics');
  artifactType.setId(1);
  return artifactType;
}

function buildClassificationMetricsArtifact() {
  const artifact = new Artifact();
  artifact.setTypeId(1);
  return artifact;
}

function buildMetricsArtifactType() {
  const artifactType = new ArtifactType();
  artifactType.setName('system.Metrics');
  artifactType.setId(2);
  return artifactType;
}

function buildMetricsArtifact() {
  const artifact = new Artifact();
  artifact.setTypeId(2);
  return artifact;
}

function buildSystemArtifactType() {
  const artifactType = new ArtifactType();
  artifactType.setName('system.Artifact');
  artifactType.setId(3);
  return artifactType;
}

function buildV1LinkedSytemArtifact() {
  const artifact = new Artifact();
  artifact.setTypeId(3);
  artifact.setUri('minio://mlpipeline/override/mlpipeline_ui_metadata');

  const event = new Event();
  const path = new Event.Path();
  path.getStepsList().push(new Event.Path.Step().setKey('mlpipeline_ui_metadata'));
  event.setPath(path);

  const linkedArtifact: mlmdUtils.LinkedArtifact = { artifact: artifact, event: event };
  return linkedArtifact;
}
