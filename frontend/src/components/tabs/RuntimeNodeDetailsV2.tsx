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

import { Button } from '@mui/material';
import * as React from 'react';
import { useState } from 'react';
// import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import {
  KubernetesExecutorConfig,
  PvcMount,
} from 'src/generated/platform_spec/kubernetes_platform';
import { useQuery } from '@tanstack/react-query';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage } from 'src/lib/Utils';
import { getTaskKeyFromNodeKey, NodeTypeNames, PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import {
  EXECUTION_KEY_CACHED_EXECUTION_ID,
  getArtifactName,
  getArtifactTypeName,
  getLinkedArtifactsByExecution,
  getStoreSessionInfoFromArtifact,
  filterEventWithOutputArtifact,
  KfpExecutionProperties,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { useArtifactTypes } from 'src/hooks/useArtifactTypes';
import { queryKeys } from 'src/hooks/queryKeys';
import WorkflowParser from 'src/lib/WorkflowParser';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import LogViewer from 'src/components/LogViewer';
import { getResourceStateText, ResourceType } from 'src/components/ResourceInfo';
import { MetricsVisualizations } from 'src/components/viewers/MetricsVisualizations';
import { ArtifactTitle } from 'src/components/tabs/ArtifactTitle';
import RuntimeInputOutputTab, {
  getArtifactParamList,
  ParamList,
} from 'src/components/tabs/InputOutputTab';
import { convertYamlToPlatformSpec, convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { PlatformDeploymentConfig } from 'src/generated/pipeline_spec/pipeline_spec';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';
import {
  usePodStatus,
  ERROR_REASONS,
  PodStatusInfo,
  ContainerStatusInfo,
  PodEventInfo,
  PodStateSnapshot,
} from 'src/hooks/usePodStatus';

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
  element?: PipelineFlowElement | null;
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
  element?: PipelineFlowElement | null;
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
  const {
    data: logsInfo,
    isError: logsQueryFailed,
    error: logsQueryError,
  } = useQuery<Map<string, string>, Error>({
    queryKey: queryKeys.executionLogs(execution?.getId(), namespace),
    queryFn: async () => {
      if (!execution) {
        throw new Error('No execution is found.');
      }
      return await getLogsInfo(execution, runId, namespace);
    },
    enabled: !!execution,
  });

  // Fetch pod status and events
  // Enabled even without execution to catch early failures like ImagePullBackOff
  const { data: podStatusInfo, error: podStatusError } = usePodStatus(
    execution,
    namespace,
    runId,
    element,
    layers,
    pipelineJobString,
  );

  const logsDetails = logsInfo?.get(LOGS_DETAILS);
  const logsBannerMessage =
    logsInfo?.get(LOGS_BANNER_MESSAGE) ||
    (logsQueryFailed ? 'Failed to retrieve pod logs.' : undefined);
  const logsBannerAdditionalInfo =
    logsInfo?.get(LOGS_BANNER_ADDITIONAL_INFO) || logsQueryError?.message;

  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Input/Output', 'Task Details', 'Logs', 'Pod Status']}
        selectedTab={selectedTab}
        onSwitch={(tab) => setSelectedTab(tab)}
      />
      <div className={commonCss.page}>
        {/* Input/Output tab */}
        {selectedTab === 0 &&
          (() => {
            if (execution) {
              return <RuntimeInputOutputTab execution={execution} namespace={namespace} />;
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
              <Banner message={logsBannerMessage} additionalInfo={logsBannerAdditionalInfo} />
            )}
            {!logsBannerMessage && (
              <div className={commonCss.pageOverflowHidden} data-testid={'logs-view-window'}>
                <LogViewer logLines={(logsDetails || '').split(/[\r\n]+/)} />
              </div>
            )}
          </div>
        )}
        {/* Pod Status tab */}
        {selectedTab === 3 && (
          <div className={padding(20)}>
            <PodStatusTab
              podStatus={podStatusInfo?.status || null}
              podEvents={podStatusInfo?.events || []}
              stateHistory={podStatusInfo?.stateHistory || []}
              error={podStatusError}
              cached={podStatusInfo?.cached}
              cachedAt={podStatusInfo?.cachedAt}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function getTaskDetailsFields(
  element?: PipelineFlowElement | null,
  execution?: Execution,
): Array<KeyValue<string>> {
  const details: Array<KeyValue<string>> = [];
  if (element) {
    details.push(['Task ID', element.id || '-']);
    if (execution) {
      // Static execution info.
      details.push([
        'Task name',
        execution.getCustomPropertiesMap().get('display_name')?.getStringValue() || '-',
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
  element?: PipelineFlowElement | null,
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
    const pvcMounts = Object.values(executor.pvcMount).map((pvcm) => PvcMount.fromJSON(pvcm));
    volumeMounts = pvcMounts.map((pvcm) => [
      pvcm.mountPath,
      pvcm.taskOutputParameter?.producerTask,
    ]);
  }

  return volumeMounts;
}

// Component to display pod status, lifecycle history, and events
interface PodStatusTabProps {
  podStatus: PodStatusInfo | null;
  podEvents: PodEventInfo[];
  stateHistory: PodStateSnapshot[];
  error?: Error | null;
  cached?: boolean;
  cachedAt?: number;
}

// Color helpers
const getPhaseColor = (phase?: string) => {
  switch (phase) {
    case 'Running':
      return '#4caf50';
    case 'Succeeded':
      return '#2196f3';
    case 'Failed':
      return '#f44336';
    case 'Pending':
      return '#ff9800';
    default:
      return '#9e9e9e';
  }
};

const getPhaseBackgroundColor = (phase?: string) => {
  switch (phase) {
    case 'Failed':
      return '#ffebee';
    case 'Running':
      return '#e8f5e9';
    case 'Succeeded':
      return '#e3f2fd';
    case 'Pending':
      return '#fff3e0';
    default:
      return '#fafafa';
  }
};

const getContainerStatusColor = (cs: ContainerStatusInfo) => {
  if (cs.reason && ERROR_REASONS.includes(cs.reason)) return '#f44336';
  if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) return '#f44336';
  if (cs.state === 'terminated' && cs.exitCode === 0) return '#2196f3';
  if (cs.state === 'terminated') return '#e57373';
  if (cs.state === 'running' && cs.ready) return '#4caf50';
  if (cs.state === 'waiting') return '#ff9800';
  return '#9e9e9e';
};

const getContainerBackgroundColor = (cs: ContainerStatusInfo) => {
  if (cs.reason && ERROR_REASONS.includes(cs.reason)) return '#ffebee';
  if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) return '#ffebee';
  return '#fff';
};

function PodStatusTab({
  podStatus,
  podEvents,
  stateHistory,
  error,
  cached,
  cachedAt,
}: PodStatusTabProps) {
  if (error) {
    return (
      <Banner message='Failed to retrieve pod status' additionalInfo={error.message} mode='error' />
    );
  }

  const formatTime = (timestamp?: number) => {
    if (!timestamp) return '';
    return new Date(timestamp).toLocaleString();
  };

  // Summarize a state snapshot into a short label
  const getStateLabel = (snapshot: PodStateSnapshot): string => {
    const phase = snapshot.phase || 'Unknown';
    if (snapshot.reason) {
      return `${phase} (${snapshot.reason})`;
    }
    return phase;
  };

  // Get container summary for a snapshot (only main container, filter out Argo-internal init/wait)
  const getContainerSummary = (snapshot: PodStateSnapshot): string[] => {
    if (!snapshot.containerStatuses) return [];
    return snapshot.containerStatuses
      .filter((cs) => cs.name === 'main')
      .map((cs) => {
        const parts: string[] = [];
        if (cs.state) parts.push(cs.state);
        if (cs.reason) parts.push(`(${cs.reason})`);
        if (cs.exitCode !== undefined) {
          parts.push(cs.exitCode === 0 ? 'exit:0' : `exit:${cs.exitCode}`);
        }
        return parts.join(' ');
      });
  };

  return (
    <div>
      {/* Cached Data Indicator */}
      {cached && (
        <div
          style={{
            backgroundColor: '#e3f2fd',
            border: '1px solid #90caf9',
            borderRadius: '4px',
            padding: '12px 16px',
            marginBottom: '16px',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <div>
            <strong style={{ color: '#1565c0' }}>Cached Data</strong>
            <div style={{ fontSize: '12px', color: '#666', marginTop: '2px' }}>
              Pod status and events cached from a previous query.
              {cachedAt && ` Last updated: ${formatTime(cachedAt)}`}
            </div>
          </div>
        </div>
      )}

      {/* Current Pod Status */}
      <div style={{ marginBottom: '24px' }}>
        <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>Current Status</h3>
        {podStatus ? (
          <div
            style={{
              border: `1px solid ${podStatus.phase === 'Failed' ? '#f44336' : '#e0e0e0'}`,
              borderRadius: '4px',
              padding: '16px',
              backgroundColor: getPhaseBackgroundColor(podStatus.phase),
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: '12px' }}>
              <span
                style={{
                  display: 'inline-block',
                  width: '12px',
                  height: '12px',
                  borderRadius: '50%',
                  backgroundColor: getPhaseColor(podStatus.phase),
                  marginRight: '8px',
                }}
              />
              <strong
                style={{
                  fontSize: '14px',
                  color: podStatus.phase === 'Failed' ? '#c62828' : 'inherit',
                }}
              >
                Phase: {podStatus.phase || 'Unknown'}
              </strong>
            </div>

            {podStatus.phase === 'Failed' && podStatus.reason && (
              <div
                style={{
                  backgroundColor: '#f44336',
                  color: '#fff',
                  padding: '12px',
                  borderRadius: '4px',
                  marginBottom: '12px',
                }}
              >
                <strong>Error: {podStatus.reason}</strong>
                {podStatus.message && (
                  <div style={{ marginTop: '4px', fontSize: '13px' }}>{podStatus.message}</div>
                )}
              </div>
            )}

            {podStatus.reason && podStatus.phase !== 'Failed' && (
              <div style={{ marginBottom: '8px' }}>
                <strong>Reason:</strong> {podStatus.reason}
              </div>
            )}
            {podStatus.message && podStatus.phase !== 'Failed' && (
              <div style={{ marginBottom: '8px' }}>
                <strong>Message:</strong> {podStatus.message}
              </div>
            )}
            {podStatus.nodeName && (
              <div style={{ marginBottom: '8px' }}>
                <strong>Node:</strong> {podStatus.nodeName}
              </div>
            )}
            {podStatus.podIP && (
              <div style={{ marginBottom: '8px' }}>
                <strong>Pod IP:</strong> {podStatus.podIP}
              </div>
            )}

            {/* Container Status - show only main container (filter out Argo-internal init/wait) */}
            {podStatus.containerStatuses &&
              podStatus.containerStatuses.filter((cs) => cs.name === 'main').length > 0 && (
                <div style={{ marginTop: '16px' }}>
                  <strong>Container:</strong>
                  {podStatus.containerStatuses
                    .filter((cs) => cs.name === 'main')
                    .map((cs, idx) => (
                      <div
                        key={idx}
                        style={{
                          marginTop: '8px',
                          padding: '12px',
                          backgroundColor: getContainerBackgroundColor(cs),
                          border: `1px solid ${cs.reason && ERROR_REASONS.includes(cs.reason) ? '#f44336' : '#e0e0e0'}`,
                          borderRadius: '4px',
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center' }}>
                          <span
                            style={{
                              display: 'inline-block',
                              width: '8px',
                              height: '8px',
                              borderRadius: '50%',
                              backgroundColor: getContainerStatusColor(cs),
                              marginRight: '8px',
                            }}
                          />
                          <strong>{cs.name}</strong>
                          <span style={{ marginLeft: '8px', color: '#666' }}>
                            {cs.state || 'unknown'}
                          </span>
                          <span
                            style={{
                              marginLeft: '4px',
                              color:
                                cs.state === 'terminated'
                                  ? cs.exitCode === 0
                                    ? '#2196f3'
                                    : '#c62828'
                                  : '#666',
                              fontWeight:
                                cs.state === 'terminated' && cs.exitCode !== 0 ? 500 : 'normal',
                            }}
                          >
                            {cs.state === 'terminated'
                              ? cs.exitCode === 0
                                ? '(Completed)'
                                : '(Failed)'
                              : cs.ready
                                ? '(Ready)'
                                : '(Not Ready)'}
                          </span>
                        </div>
                        {cs.restartCount > 0 && (
                          <div style={{ color: '#ff9800', marginTop: '4px' }}>
                            Restarts: {cs.restartCount}
                          </div>
                        )}
                        {cs.reason && (
                          <div
                            style={{
                              marginTop: '4px',
                              color: ERROR_REASONS.includes(cs.reason) ? '#c62828' : '#333',
                              fontWeight: ERROR_REASONS.includes(cs.reason) ? 500 : 'normal',
                            }}
                          >
                            Reason: {cs.reason}
                          </div>
                        )}
                        {cs.message && (
                          <div
                            style={{
                              color: '#666',
                              fontSize: '12px',
                              marginTop: '4px',
                              wordBreak: 'break-word',
                            }}
                          >
                            {cs.message}
                          </div>
                        )}
                        {cs.exitCode !== undefined && (
                          <div
                            style={{
                              color: cs.exitCode !== 0 ? '#f44336' : '#4caf50',
                              marginTop: '4px',
                              fontWeight: cs.exitCode !== 0 ? 500 : 'normal',
                            }}
                          >
                            Exit Code: {cs.exitCode}
                          </div>
                        )}
                      </div>
                    ))}
                </div>
              )}
          </div>
        ) : (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            Pod status not available. The pod may have been deleted.
          </div>
        )}
      </div>

      {/* Pod Lifecycle Timeline */}
      {stateHistory.length > 0 && (
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>
            Pod Lifecycle ({stateHistory.length} state changes)
          </h3>
          <div style={{ position: 'relative', paddingLeft: '24px' }}>
            {/* Vertical timeline line */}
            <div
              style={{
                position: 'absolute',
                left: '7px',
                top: '4px',
                bottom: '4px',
                width: '2px',
                backgroundColor: '#e0e0e0',
              }}
            />
            {stateHistory.map((snapshot, idx) => {
              const isLast = idx === stateHistory.length - 1;
              const containerSummary = getContainerSummary(snapshot);
              return (
                <div key={idx} style={{ position: 'relative', marginBottom: isLast ? 0 : '16px' }}>
                  {/* Timeline dot */}
                  <div
                    style={{
                      position: 'absolute',
                      left: '-20px',
                      top: '3px',
                      width: '12px',
                      height: '12px',
                      borderRadius: '50%',
                      backgroundColor: snapshot.isError ? '#f44336' : getPhaseColor(snapshot.phase),
                      border: '2px solid #fff',
                      boxShadow:
                        '0 0 0 1px ' +
                        (snapshot.isError ? '#f44336' : getPhaseColor(snapshot.phase)),
                    }}
                  />
                  <div
                    style={{
                      padding: '8px 12px',
                      borderRadius: '4px',
                      backgroundColor: snapshot.isError
                        ? '#ffebee'
                        : isLast
                          ? getPhaseBackgroundColor(snapshot.phase)
                          : '#fafafa',
                      border: `1px solid ${snapshot.isError ? '#ef9a9a' : '#e0e0e0'}`,
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                      }}
                    >
                      <span
                        style={{
                          fontWeight: 500,
                          color: snapshot.isError ? '#c62828' : '#333',
                          fontSize: '13px',
                        }}
                      >
                        {getStateLabel(snapshot)}
                      </span>
                      <span style={{ color: '#999', fontSize: '11px' }}>
                        {formatTime(snapshot.timestamp)}
                      </span>
                    </div>
                    {snapshot.message && (
                      <div
                        style={{
                          color: '#666',
                          fontSize: '12px',
                          marginTop: '4px',
                          wordBreak: 'break-word',
                        }}
                      >
                        {snapshot.message}
                      </div>
                    )}
                    {containerSummary.length > 0 && (
                      <div style={{ marginTop: '4px' }}>
                        {containerSummary.map((cs, csIdx) => (
                          <div key={csIdx} style={{ color: '#666', fontSize: '11px' }}>
                            {cs}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Pod Events */}
      <div>
        <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>
          Pod Events {podEvents.length > 0 && `(${podEvents.length})`}
        </h3>
        {podEvents.length > 0 ? (
          <div style={{ border: '1px solid #e0e0e0', borderRadius: '4px', overflow: 'hidden' }}>
            {podEvents.map((event, idx) => (
              <div
                key={idx}
                style={{
                  padding: '12px 16px',
                  borderBottom: idx < podEvents.length - 1 ? '1px solid #e0e0e0' : 'none',
                  backgroundColor: event.type === 'Warning' ? '#fff3e0' : '#fff',
                }}
              >
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}
                >
                  <span>
                    <span
                      style={{
                        color: event.type === 'Warning' ? '#ff9800' : '#4caf50',
                        fontWeight: 500,
                        marginRight: '8px',
                      }}
                    >
                      [{event.type}]
                    </span>
                    <strong>{event.reason}</strong>
                    {event.count > 1 && (
                      <span style={{ color: '#666', marginLeft: '8px' }}>(x{event.count})</span>
                    )}
                  </span>
                  <span style={{ color: '#666', fontSize: '12px' }}>
                    {event.lastTimestamp ? new Date(event.lastTimestamp).toLocaleString() : '-'}
                  </span>
                </div>
                <div style={{ color: '#333', fontSize: '13px' }}>{event.message}</div>
                {event.source && (
                  <div style={{ color: '#999', fontSize: '11px', marginTop: '4px' }}>
                    Source: {event.source}
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            No events available. Events may have expired or the pod may have been deleted.
          </div>
        )}
      </div>
    </div>
  );
}

async function getLogsInfo(
  execution: Execution,
  runId?: string,
  namespace?: string,
): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  let podName = '';
  let podNameSpace = '';
  let cachedExecutionId = '';
  let logsDetails = '';
  let logsBannerMessage = '';
  let logsBannerAdditionalInfo = '';
  const customPropertiesMap = execution.getCustomPropertiesMap();
  const createdAt = new Date(execution.getCreateTimeSinceEpoch()).toISOString().split('T')[0];

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
    logsDetails = await Apis.getPodLogs(runId!, podName, podNameSpace, createdAt);
    logsInfo.set(LOGS_DETAILS, logsDetails);
  } catch (err) {
    // Primary method failed, try to get logs from executor-logs artifact in MLMD
    console.log('Pod logs retrieval failed, attempting to fetch executor-logs artifact from MLMD');
    try {
      const linkedArtifacts = await getLinkedArtifactsByExecution(execution);
      const outputArtifacts = filterEventWithOutputArtifact(linkedArtifacts);
      const executorLogsArtifact = outputArtifacts.find(
        (artifact) => getArtifactName(artifact) === 'executor-logs',
      );

      if (executorLogsArtifact) {
        const uri = executorLogsArtifact.artifact.getUri();
        const storagePath = WorkflowParser.parseStoragePath(uri);
        const providerInfo = getStoreSessionInfoFromArtifact(executorLogsArtifact);
        const artifactNamespace = namespace || podNameSpace;

        logsDetails = await Apis.readFile({
          path: storagePath,
          providerInfo: providerInfo,
          namespace: artifactNamespace,
        });
        logsInfo.set(LOGS_DETAILS, logsDetails);
        return logsInfo;
      }
    } catch (artifactErr) {
      console.error('Failed to retrieve executor-logs artifact:', artifactErr);
    }

    // Both methods failed
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
  const { data } = useArtifactTypes();

  const [selectedTab, setSelectedTab] = useState(0);
  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Artifact Info', 'Visualization']}
        selectedTab={selectedTab}
        onSwitch={(tab) => setSelectedTab(tab)}
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

interface ArtifactInfoProps {
  execution?: Execution;
  artifactTypes?: ArtifactType[];
  linkedArtifact?: LinkedArtifact;
  namespace: string | undefined;
}

function ArtifactInfo({ execution, artifactTypes, linkedArtifact, namespace }: ArtifactInfoProps) {
  if (!execution || !linkedArtifact) {
    return NODE_STATE_UNAVAILABLE;
  }

  // Static Artifact information.
  const taskName = execution.getCustomPropertiesMap().get('display_name')?.getStringValue() || '-';
  const artifactName =
    linkedArtifact.artifact.getCustomPropertiesMap().get('display_name')?.getStringValue() || '-';
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
  element: PipelineFlowElement;
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
          onSwitch={(tab) => setSelectedTab(tab)}
        />
        <div className={commonCss.page}>
          {/* Input/Output tab */}
          {selectedTab === 0 &&
            (() => {
              if (execution) {
                return (
                  <RuntimeInputOutputTab
                    execution={execution}
                    namespace={namespace}
                  ></RuntimeInputOutputTab>
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
