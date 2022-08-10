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

import * as React from 'react';
import { useState } from 'react';
import { FlowElement } from 'react-flow-renderer';
import { useQuery } from 'react-query';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { NodeTypeNames } from 'src/lib/v2/StaticFlow';
import { getArtifactTypeName, getArtifactTypes, LinkedArtifact } from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { Artifact, ArtifactType, Execution } from 'src/third_party/mlmd';
import ArtifactPreview from '../ArtifactPreview';
import DetailsTable from '../DetailsTable';
import { FlowElementDataBase } from '../graph/Constants';
import { getResourceStateText, ResourceType } from '../ResourceInfo';
import { MetricsVisualizations } from '../viewers/MetricsVisualizations';
import { ArtifactTitle } from './ArtifactTitle';
import InputOutputTab, { getArtifactParamList } from './InputOutputTab';

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
  element?: FlowElement<FlowElementDataBase> | null;
  elementMlmdInfo?: NodeMlmdInfo | null;
  namespace: string | undefined;
}

export function RuntimeNodeDetailsV2({
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
          element={element}
          execution={elementMlmdInfo?.execution}
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
    }
    return NODE_INFO_UNKNOWN;
  })();
}

interface TaskNodeDetailProps {
  element?: FlowElement<FlowElementDataBase> | null;
  execution?: Execution;
  namespace: string | undefined;
}

function TaskNodeDetail({ element, execution, namespace }: TaskNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  return (
    <div className={commonCss.page}>
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
              return <InputOutputTab execution={execution} namespace={namespace}></InputOutputTab>;
            }
            return NODE_STATE_UNAVAILABLE;
          })()}

        {/* Task Details tab */}
        {selectedTab === 1 && (
          <div className={padding(20)}>
            <DetailsTable title='Task Details' fields={getTaskDetailsFields(element, execution)} />
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
  const lastUpdatedTime = linkedArtifact.artifact.getLastUpdateTimeSinceEpoch();
  let finishedAt = '-';
  if (lastUpdatedTime && linkedArtifact.artifact.getState() === Artifact.State.LIVE) {
    finishedAt = new Date(lastUpdatedTime).toString();
  }

  // Artifact info rows.
  const artifactInfo = [
    ['Upstream Task Name', taskName],
    ['Artifact Name', artifactName],
    ['Artifact Type', artifactTypeName],
    ['Created At', createdAt],
    ['Finished At', finishedAt],
  ];

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
          title='Artifact URL'
          fields={getArtifactParamList([linkedArtifact], artifactTypeName)}
          valueComponent={ArtifactPreview}
          valueComponentProps={{
            namespace: namespace,
          }}
        />
      </div>
    </div>
  );
}
