// Copyright 2026 The Kubeflow Authors
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

import { Link } from 'react-router-dom';
import { ReactElement } from 'react';
import { InputOutputsIOArtifact, V2beta1PipelineTask } from 'src/apisv2beta1/run';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import { RoutePageFactory } from 'src/components/Router';
import { commonCss, padding } from 'src/Css';
import {
  flattenArtifactGroups,
  formatParameters,
  getArtifactDisplayName,
  getArtifactSessionInfo,
} from 'src/lib/v2/RuntimeArtifactUtils';

export interface RuntimeInputOutputTabProps {
  task: V2beta1PipelineTask;
  namespace?: string;
}

export function RuntimeInputOutputTab({ task, namespace }: RuntimeInputOutputTabProps) {
  const inputParameters = formatParameters(task.inputs?.parameters);
  const outputParameters = formatParameters(task.outputs?.parameters);
  const inputArtifacts = buildArtifactRows(task.inputs?.artifacts);
  const outputArtifacts = buildArtifactRows(task.outputs?.artifacts);
  const isEmpty =
    !inputParameters.length &&
    !outputParameters.length &&
    !inputArtifacts.rows.length &&
    !outputArtifacts.rows.length;

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <h3>{task.display_name || task.name || 'Task'}</h3>
          {isEmpty && (
            <Banner message='There is no input/output parameter or artifact.' mode='info' />
          )}
          {!!inputParameters.length && (
            <DetailsTable title='Input Parameters' fields={inputParameters} />
          )}
          {!!inputArtifacts.rows.length && (
            <DetailsTable<string>
              title='Input Artifacts'
              fields={inputArtifacts.rows}
              valueComponent={ArtifactPreview}
              valueComponentProps={{ namespace, sessionMap: inputArtifacts.sessionMap }}
            />
          )}
          {!!outputParameters.length && (
            <DetailsTable title='Output Parameters' fields={outputParameters} />
          )}
          {!!outputArtifacts.rows.length && (
            <DetailsTable<string>
              title='Output Artifacts'
              fields={outputArtifacts.rows}
              valueComponent={ArtifactPreview}
              valueComponentProps={{ namespace, sessionMap: outputArtifacts.sessionMap }}
            />
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}

function buildArtifactRows(groups: InputOutputsIOArtifact[] | undefined) {
  const rows: Array<[string | ReactElement | undefined, string]> = [];
  const sessionMap = new Map<string, string | undefined>();
  for (const { artifact, artifactKey, index } of flattenArtifactGroups(groups)) {
    const displayName = getArtifactDisplayName(artifact, artifactKey, index);
    const label = artifact.artifact_id ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(artifact.artifact_id)}>
        {displayName}
      </Link>
    ) : (
      displayName
    );
    const uri = artifact.uri || '';
    rows.push([label, uri]);
    sessionMap.set(uri, getArtifactSessionInfo(artifact));
  }
  return { rows, sessionMap };
}

export default RuntimeInputOutputTab;
