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

import React from 'react';
import { useQuery } from 'react-query';
import { Link } from 'react-router-dom';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { commonCss, padding } from 'src/Css';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { getMetadataValue } from 'src/mlmd/library';
import {
  filterEventWithInputArtifact,
  filterEventWithOutputArtifact,
  getArtifactName,
  getArtifactTypeName,
  getArtifactTypes,
  getLinkedArtifactsByExecution,
  getStoreSessionInfoFromArtifact,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import ArtifactPreview from '../ArtifactPreview';
import Banner from '../Banner';
import DetailsTable from '../DetailsTable';
import { RoutePageFactory } from '../Router';
import { ExecutionTitle } from './ExecutionTitle';

export type ParamList = Array<KeyValue<string>>;
export type URIToSessionInfo = Map<string, string | undefined>;
export interface ArtifactParamsWithSessionInfo {
  params: ParamList;
  sessionMap: URIToSessionInfo;
}

export interface ArtifactLocation {
  uri: string;
  store_session_info: string | undefined;
}

export interface IOTabProps {
  execution: Execution;
  namespace: string | undefined;
}

export function InputOutputTab({ execution, namespace }: IOTabProps) {
  const executionId = execution.getId();

  // TODO(jlyaoyuli): Showing input/output parameter for unexecuted node (retrieves from PipelineSpec).
  // TODO(jlyaoyuli): Display other information (container, args, image, command)

  // Retrieves input and output artifacts from Metadata store.
  const { isSuccess, error, data: linkedArtifacts } = useQuery<LinkedArtifact[], Error>(
    ['execution_artifact', { id: executionId, state: execution.getLastKnownState() }],
    () => getLinkedArtifactsByExecution(execution),
    { staleTime: Infinity },
  );

  const { data: artifactTypes } = useQuery<ArtifactType[], Error>(
    ['artifact_types', { linkedArtifact: linkedArtifacts }],
    () => getArtifactTypes(),
    {},
  );

  const artifactTypeNames =
    linkedArtifacts && artifactTypes ? getArtifactTypeName(artifactTypes, linkedArtifacts) : [];

  // Restructs artifacts and parameters for visualization.
  const inputParams = extractInputFromExecution(execution);
  const outputParams = extractOutputFromExecution(execution);
  let inputArtifactsWithSessionInfo: ArtifactParamsWithSessionInfo | undefined;
  let outputArtifactsWithSessionInfo: ArtifactParamsWithSessionInfo | undefined;
  if (isSuccess && linkedArtifacts) {
    inputArtifactsWithSessionInfo = getArtifactParamList(
      filterEventWithInputArtifact(linkedArtifacts),
      artifactTypeNames,
    );
    outputArtifactsWithSessionInfo = getArtifactParamList(
      filterEventWithOutputArtifact(linkedArtifacts),
      artifactTypeNames,
    );
  }

  let inputArtifacts: ParamList = [];
  let outputArtifacts: ParamList = [];

  if (inputArtifactsWithSessionInfo) {
    inputArtifacts = inputArtifactsWithSessionInfo.params;
  }

  if (outputArtifactsWithSessionInfo) {
    outputArtifacts = outputArtifactsWithSessionInfo.params;
  }

  let isIoEmpty = false;
  if (
    inputParams.length === 0 &&
    outputParams.length === 0 &&
    inputArtifacts.length === 0 &&
    outputArtifacts.length === 0
  ) {
    isIoEmpty = true;
  }

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <ExecutionTitle execution={execution} />

          {error && (
            <Banner
              message='Error in retrieving Artifacts.'
              mode='error'
              additionalInfo={error.message}
            />
          )}

          {isSuccess && isIoEmpty && (
            <Banner message='There is no input/output parameter or artifact.' mode='info' />
          )}

          {inputParams.length > 0 && (
            <div>
              <DetailsTable
                key={`input-parameters-${executionId}`}
                title='Input Parameters'
                fields={inputParams}
              />
            </div>
          )}

          {inputArtifacts.length > 0 && (
            <div>
              <DetailsTable<string>
                key={`input-artifacts-${executionId}`}
                title='Input Artifacts'
                fields={inputArtifacts}
                valueComponent={ArtifactPreview}
                valueComponentProps={{
                  namespace: namespace,
                  sessionMap: inputArtifactsWithSessionInfo?.sessionMap,
                }}
              />
            </div>
          )}

          {outputParams.length > 0 && (
            <div>
              <DetailsTable
                key={`output-parameters-${executionId}`}
                title='Output Parameters'
                fields={outputParams}
              />
            </div>
          )}

          {outputArtifacts.length > 0 && (
            <div>
              <DetailsTable<string>
                key={`output-artifacts-${executionId}`}
                title='Output Artifacts'
                fields={outputArtifacts}
                valueComponent={ArtifactPreview}
                valueComponentProps={{
                  namespace: namespace,
                  sessionMap: outputArtifactsWithSessionInfo?.sessionMap,
                }}
              />
            </div>
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}

export default InputOutputTab;

function extractInputFromExecution(execution: Execution): KeyValue<string>[] {
  return extractParamFromExecution(execution, 'inputs');
}

function extractOutputFromExecution(execution: Execution): KeyValue<string>[] {
  return extractParamFromExecution(execution, 'outputs');
}

function extractParamFromExecution(execution: Execution, name: string): KeyValue<string>[] {
  const result: KeyValue<string>[] = [];
  execution.getCustomPropertiesMap().forEach((value, key) => {
    if (key === name) {
      const param = getMetadataValue(value);
      if (typeof param == 'object') {
        Object.entries(param.toJavaScript()).forEach(parameter => {
          result.push([parameter[0], JSON.stringify(parameter[1])]);
        });
      }
    }
  });
  return result;
}

export function getArtifactParamList(
  inputArtifacts: LinkedArtifact[],
  artifactTypeNames: string[],
): ArtifactParamsWithSessionInfo {
  let sessMap: URIToSessionInfo = new Map<string, string | undefined>();

  let params = Object.values(inputArtifacts).map((linkedArtifact, index) => {
    let key = getArtifactName(linkedArtifact);
    if (
      key &&
      (artifactTypeNames[index] === 'system.Metrics' ||
        artifactTypeNames[index] === 'system.ClassificationMetrics')
    ) {
      key += ' (This is an empty file by default)';
    }
    const artifactId = linkedArtifact.artifact.getId();
    const artifactElement = RoutePageFactory.artifactDetails(artifactId) ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(artifactId)}>
        {key}
      </Link>
    ) : (
      key
    );

    const uri = linkedArtifact.artifact.getUri();
    const sessInfo = getStoreSessionInfoFromArtifact(linkedArtifact);
    sessMap.set(uri, sessInfo);

    return [artifactElement, uri];
  });

  return { params: params, sessionMap: sessMap };
}
