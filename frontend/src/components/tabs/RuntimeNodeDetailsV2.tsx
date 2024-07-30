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

import { Button } from '@material-ui/core';
import * as React from 'react';
import { useState } from 'react';
import { FlowElement } from 'react-flow-renderer';
// import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import {
  KubernetesExecutorConfig,
  PvcMount,
} from 'src/generated/platform_spec/kubernetes_platform';
import { useQuery } from 'react-query';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage } from 'src/lib/Utils';
import { getTaskKeyFromNodeKey, NodeTypeNames } from 'src/lib/v2/StaticFlow';
import {
  EXECUTION_KEY_CACHED_EXECUTION_ID,
  getArtifactTypeName,
  getArtifactTypes,
  KfpExecutionProperties,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import LogViewer from 'src/components/LogViewer';
import { getResourceStateText, ResourceType } from 'src/components/ResourceInfo';
import { MetricsVisualizations } from 'src/components/viewers/MetricsVisualizations';
import { ArtifactTitle } from 'src/components/tabs/ArtifactTitle';
import InputOutputTab, {
  getArtifactParamList,
  ParamList,
} from 'src/components/tabs/InputOutputTab';
import { convertYamlToPlatformSpec, convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { PlatformDeploymentConfig } from 'src/generated/pipeline_spec/pipeline_spec';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';

export const LOGS_DETAILS = 'logs_details';
export const LOGS_BANNER_MESSAGE = 'logs_banner_message';
export const LOGS_BANNER_ADDITIONAL_INFO = 'logs_banner_additional_info';
export const K8S_PLATFORM_KEY = 'kubernetes';

const NODE_INFO_UNKNOWN = (
  <div className='relative flex flex-col h-screen'>
    <div className='absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2'>
      Unable to retrieve node info.
    </div>
  </div>
);

const NODE_STATE_UNAVAILABLE = (
  <div className='relative flex flex-col h-screen'>
    <div className='absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2'>
      Content is not available yet.
    </div>
  </div>
);

interface RuntimeNodeDetailsV2Props {
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  pipelineJobString?: string;
  runId?: string;
  element?: FlowElement<FlowElementDataBase> | null;
  elementMlmdInfo?: NodeMlmdInfo | null;
  namespace: string | undefined;
}

export function RuntimeNodeDetailsV2({
  layers,
  onLayerChange,
  pipelineJobString,
  runId,
  element,
  elementMlmdInfo,
  namespace,
}: RuntimeNodeDetailsV2Props) {
  if (!element) {
    return NODE_INFO_UNKNOWN;
  }

  return (() => {
    if (NodeTypeNames.EXECUTION === element.type) {
      return (
        <TaskNodeDetail
          pipelineJobString={pipelineJobString}
          runId={runId}
          element={element}
          execution={elementMlmdInfo?.execution}
          layers={layers}
          namespace={namespace}
        ></TaskNodeDetail>
      );
    } else if (NodeTypeNames.ARTIFACT === element.type) {
      return (
        <ArtifactNodeDetail
          execution={elementMlmdInfo?.execution}
          linkedArtifact={elementMlmdInfo?.linkedArtifact}
          namespace={namespace}
        />
      );
    } else if (NodeTypeNames.SUB_DAG === element.type) {
      return (
        <SubDAGNodeDetail
          element={element}
          execution={elementMlmdInfo?.execution}
          layers={layers}
          onLayerChange={onLayerChange}
          namespace={namespace}
        />
      );
    }
    return NODE_INFO_UNKNOWN;
  })();
}

interface TaskNodeDetailProps {
  pipelineJobString?: string;
  runId?: string;
  element?: FlowElement<FlowElementDataBase> | null;
  execution?: Execution;
  layers: string[];
  namespace: string | undefined;
}

function TaskNodeDetail({
  pipelineJobString,
  runId,
  element,
  execution,
  layers,
  namespace,
}: TaskNodeDetailProps) {
  const { data: logsInfo } = useQuery<Map<string, string>, Error>(
    [execution],
    async () => {
      if (!execution) {
        throw new Error('No execution is found.');
      }
      return await getLogsInfo(execution, runId);
    },
    { enabled: !!execution },
  );

  const logsDetails = logsInfo?.get(LOGS_DETAILS);
  const logsBannerMessage = logsInfo?.get(LOGS_BANNER_MESSAGE);
  const logsBannerAdditionalInfo = logsInfo?.get(LOGS_BANNER_ADDITIONAL_INFO);

  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Input/Output', 'Task Details', 'Logs']}
        selectedTab={selectedTab}
        onSwitch={tab => setSelectedTab(tab)}
      />
      <div className={commonCss.page}>
        {/* Input/Output tab */}
        {selectedTab === 0 &&
          (() => {
            if (execution) {
              return <InputOutputTab execution={execution} namespace={namespace} />;
            }
            return NODE_STATE_UNAVAILABLE;
          })()}

        {/* Task Details tab */}
        {selectedTab === 1 && (
          <div className={padding(20)}>
            <DetailsTable title='Task Details' fields={getTaskDetailsFields(element, execution)} />
            <DetailsTable
              title='Volume Mounts'
              fields={getNodeVolumeMounts(layers, pipelineJobString, element)}
            />
          </div>
        )}
        {/* Logs tab */}
        {selectedTab === 2 && (
          <div className={commonCss.page}>
            {logsBannerMessage && (
              <React.Fragment>
                <Banner message={logsBannerMessage} additionalInfo={logsBannerAdditionalInfo} />
              </React.Fragment>
            )}
            {!logsBannerMessage && (
              <div className={commonCss.pageOverflowHidden} data-testid={'logs-view-window'}>
                <LogViewer logLines={(logsDetails || '').split(/[\r\n]+/)} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function getTaskDetailsFields(
  element?: FlowElement<FlowElementDataBase> | null,
  execution?: Execution,
): Array<KeyValue<string>> {
  const details: Array<KeyValue<string>> = [];
  if (element) {
    details.push(['Task ID', element.id || '-']);
    if (execution) {
      // Static execution info.
      details.push([
        'Task name',
        execution
          .getCustomPropertiesMap()
          .get('display_name')
          ?.getStringValue() || '-',
      ]);

      // Runtime execution info.
      const stateText = getResourceStateText({
        resourceType: ResourceType.EXECUTION,
        resource: execution,
        typeName: 'Execution',
      });
      details.push(['Status', stateText || '-']);

      const createdAt = new Date(execution.getCreateTimeSinceEpoch()).toString();
      details.push(['Created At', createdAt]);

      const lastUpdatedTime = execution.getLastUpdateTimeSinceEpoch();
      let finishedAt = '-';
      if (
        lastUpdatedTime &&
        (execution.getLastKnownState() === Execution.State.COMPLETE ||
          execution.getLastKnownState() === Execution.State.FAILED ||
          execution.getLastKnownState() === Execution.State.CACHED ||
          execution.getLastKnownState() === Execution.State.CANCELED)
      ) {
        finishedAt = new Date(lastUpdatedTime).toString();
      }
      details.push(['Finished At', finishedAt]);
    }
  }

  return details;
}

function getNodeVolumeMounts(
  layers: string[],
  pipelineJobString?: string,
  element?: FlowElement<FlowElementDataBase> | null,
): Array<KeyValue<string>> {
  if (!pipelineJobString || !element) {
    return [];
  }

  const taskKey = getTaskKeyFromNodeKey(element.id);
  const pipelineSpec = convertYamlToV2PipelineSpec(pipelineJobString);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  const platformSpec = convertYamlToPlatformSpec(pipelineJobString);

  // Currently support kubernetes platform
  if (!platformSpec || !platformSpec.platforms[K8S_PLATFORM_KEY]) {
    return [];
  }

  const k8sDeploymentSpec = PlatformDeploymentConfig.fromJSON(
    platformSpec.platforms[K8S_PLATFORM_KEY].deploymentSpec,
  );
  const matchedExecutorObj = Object.entries(k8sDeploymentSpec.executors).find(
    ([executorName]) => executorName === componentSpec?.executorLabel,
  );

  let volumeMounts: Array<KeyValue<string>> = [];
  if (matchedExecutorObj) {
    const executor = KubernetesExecutorConfig.fromJSON(matchedExecutorObj[1]);
    const pvcMounts = Object.values(executor.pvcMount).map(pvcm => PvcMount.fromJSON(pvcm));
    volumeMounts = pvcMounts.map(pvcm => [pvcm.mountPath, pvcm.taskOutputParameter?.producerTask]);
  }

  return volumeMounts;
}

async function getLogsInfo(execution: Execution, runId?: string): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  let podName = '';
  let podNameSpace = '';
  let cachedExecutionId = '';
  let logsDetails = '';
  let logsBannerMessage = '';
  let logsBannerAdditionalInfo = '';
  const customPropertiesMap = execution.getCustomPropertiesMap();

  if (execution) {
    podName = customPropertiesMap.get(KfpExecutionProperties.POD_NAME)?.getStringValue() || '';
    podNameSpace = customPropertiesMap.get('namespace')?.getStringValue() || '';
    cachedExecutionId =
      customPropertiesMap.get(EXECUTION_KEY_CACHED_EXECUTION_ID)?.getStringValue() || '';
  }

  // TODO(jlyaoyuli): Consider to link to the cached execution.
  if (cachedExecutionId) {
    logsInfo.set(LOGS_DETAILS, 'This step output is taken from cache.');
    return logsInfo; // Early return if it is from cache.
  }

  try {
    logsDetails = await Apis.getPodLogs(runId!, podName, podNameSpace);
    logsInfo.set(LOGS_DETAILS, logsDetails);
  } catch (err) {
    let errMsg = await errorToMessage(err);
    logsBannerMessage = 'Failed to retrieve pod logs.';
    logsInfo.set(LOGS_BANNER_MESSAGE, logsBannerMessage);
    logsBannerAdditionalInfo = 'Error response: ' + errMsg;
    logsInfo.set(LOGS_BANNER_ADDITIONAL_INFO, logsBannerAdditionalInfo);
  }
  return logsInfo;
}

interface ArtifactNodeDetailProps {
  execution?: Execution;
  linkedArtifact?: LinkedArtifact;
  namespace: string | undefined;
}
function ArtifactNodeDetail({ execution, linkedArtifact, namespace }: ArtifactNodeDetailProps) {
  const { data } = useQuery<ArtifactType[], Error>(
    ['artifact_types', { linkedArtifact }],
    () => getArtifactTypes(),
    {},
  );

  const [selectedTab, setSelectedTab] = useState(0);
  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Artifact Info', 'Visualization']}
        selectedTab={selectedTab}
        onSwitch={tab => setSelectedTab(tab)}
      />
      <div className={padding(20)}>
        {/* Artifact Info tab */}
        {selectedTab === 0 && (
          <ArtifactInfo
            execution={execution}
            artifactTypes={data}
            linkedArtifact={linkedArtifact}
            namespace={namespace}
          ></ArtifactInfo>
        )}

        {/* Visualization tab */}
        {selectedTab === 1 && execution && (
          <MetricsVisualizations
            linkedArtifacts={linkedArtifact ? [linkedArtifact] : []}
            artifactTypes={data ? data : []}
            execution={execution}
            namespace={namespace}
          />
        )}
      </div>
    </div>
  );
}

interface ArtifactNodeDetailProps {
  execution?: Execution;
  artifactTypes?: ArtifactType[];
  linkedArtifact?: LinkedArtifact;
  namespace: string | undefined;
}

function ArtifactInfo({
  execution,
  artifactTypes,
  linkedArtifact,
  namespace,
}: ArtifactNodeDetailProps) {
  if (!execution || !linkedArtifact) {
    return NODE_STATE_UNAVAILABLE;
  }

  // Static Artifact information.
  const taskName =
    execution
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue() || '-';
  const artifactName =
    linkedArtifact.artifact
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue() || '-';
  let artifactTypeName = artifactTypes
    ? getArtifactTypeName(artifactTypes, [linkedArtifact])
    : ['-'];

  // Runtime artifact information.
  const createdAt = new Date(linkedArtifact.artifact.getCreateTimeSinceEpoch());

  // Artifact info rows.
  const artifactInfo = [
    ['Upstream Task Name', taskName],
    ['Artifact Name', artifactName],
    ['Artifact Type', artifactTypeName],
    ['Created At', createdAt],
  ];

  let artifactParamsWithSessionInfo = getArtifactParamList([linkedArtifact], artifactTypeName);
  let artifactParams: ParamList = [];

  if (artifactParamsWithSessionInfo) {
    artifactParams = artifactParamsWithSessionInfo.params;
  }

  return (
    <div>
      <ArtifactTitle artifact={linkedArtifact.artifact}></ArtifactTitle>
      {artifactInfo && (
        <div>
          <DetailsTable title='Artifact Info' fields={artifactInfo} />
        </div>
      )}

      <div>
        <DetailsTable<string>
          key={`artifact-url`}
          title='Artifact URI'
          fields={artifactParams}
          valueComponent={ArtifactPreview}
          valueComponentProps={{
            namespace: namespace,
            sessionMap: artifactParamsWithSessionInfo.sessionMap,
          }}
        />
      </div>
    </div>
  );
}

interface SubDAGNodeDetailProps {
  element: FlowElement<FlowElementDataBase>;
  execution?: Execution;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  namespace: string | undefined;
}

function SubDAGNodeDetail({
  element,
  execution,
  layers,
  onLayerChange,
  namespace,
}: SubDAGNodeDetailProps) {
  const taskKey = getTaskKeyFromNodeKey(element.id);
  // const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  // if (!componentSpec) {
  //   return NODE_INFO_UNKNOWN;
  // }

  const onSubDagOpenClick = () => {
    onLayerChange([...layers, taskKey]);
  };

  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div>
      <div className={commonCss.page}>
        <div className={padding(20, 'blr')}>
          <Button variant='contained' onClick={onSubDagOpenClick}>
            Open Sub-DAG
          </Button>
        </div>
        <MD2Tabs
          tabs={['Input/Output', 'Task Details']}
          selectedTab={selectedTab}
          onSwitch={tab => setSelectedTab(tab)}
        />
        <div className={commonCss.page}>
          {/* Input/Output tab */}
          {selectedTab === 0 &&
            (() => {
              if (execution) {
                return (
                  <InputOutputTab execution={execution} namespace={namespace}></InputOutputTab>
                );
              }
              return NODE_STATE_UNAVAILABLE;
            })()}

          {/* Task Details tab */}
          {selectedTab === 1 && (
            <div className={padding(20)}>
              <DetailsTable
                title='Task Details'
                fields={getTaskDetailsFields(element, execution)}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
