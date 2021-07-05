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

import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import React from 'react';
import { useQuery } from 'react-query';
import { Link } from 'react-router-dom';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { commonCss, padding } from 'src/Css';
import {
  filterEventWithInputArtifact,
  filterEventWithOutputArtifact,
  getArtifactName,
  getLinkedArtifactsByExecution,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { getMetadataValue } from 'src/mlmd/library';
import { Execution } from 'src/third_party/mlmd';
import ArtifactPreview from '../ArtifactPreview';
import Banner from '../Banner';
import DetailsTable from '../DetailsTable';
import { RoutePageFactory } from '../Router';
import { ExecutionTitle } from './ExecutionTitle';

type ParamList = Array<KeyValue<string>>;

export interface IOTabProps {
  execution: Execution;
  namespace: string | undefined;
}

export function InputOutputTab({ execution, namespace }: IOTabProps) {
  const executionId = execution.getId();

  // Retrieves input and output artifacts from Metadata store.
  const { isSuccess, error, data } = useQuery<LinkedArtifact[], Error>(
    ['execution_artifact', { id: executionId, state: execution.getLastKnownState() }],
    () => getLinkedArtifactsByExecution(execution),
    { staleTime: Infinity },
  );

  // Restructs artifacts and parameters for visualization.
  const inputParams = extractInputFromExecution(execution);
  const outputParams = extractOutputFromExecution(execution);
  let inputArtifacts: ParamList = [];
  let outputArtifacts: ParamList = [];
  if (isSuccess && data) {
    inputArtifacts = getArtifactParamList(filterEventWithInputArtifact(data));
    outputArtifacts = getArtifactParamList(filterEventWithOutputArtifact(data));
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
  return extractParamFromExecution(execution, /input:(?<inputName>.+)/, 'inputName');
}

function extractOutputFromExecution(execution: Execution): KeyValue<string>[] {
  return extractParamFromExecution(execution, /output:(?<outputName>.+)/, 'outputName');
}

function extractParamFromExecution(
  execution: Execution,
  pattern: RegExp,
  groupName: string,
): KeyValue<string>[] {
  const result: KeyValue<string>[] = [];
  execution.getCustomPropertiesMap().forEach((value, key) => {
    const found = key.match(pattern);
    if (found?.groups?.[groupName]) {
      result.push([found.groups[groupName], prettyPrintValue(getMetadataValue(value))]);
    }
  });
  return result;
}

function prettyPrintValue(value: string | number | Struct | undefined): string {
  if (value == null) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number') {
    return JSON.stringify(value);
  }
  // value is Struct
  const jsObject = value.toJavaScript();
  // When Struct is converted to js object, it may contain a top level "struct"
  // or "list" key depending on its type, but the key is meaningless and we can
  // omit it in visualization.
  return JSON.stringify(jsObject?.struct || jsObject?.list || jsObject, null, 2);
}

function getArtifactParamList(inputArtifacts: LinkedArtifact[]): ParamList {
  return inputArtifacts.map(linkedArtifact => {
    const key = getArtifactName(linkedArtifact);
    const artifactId = linkedArtifact.artifact.getId();
    const artifactElement = RoutePageFactory.artifactDetails(artifactId) ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(artifactId)}>
        {key}
      </Link>
    ) : (
      key
    );
    return [artifactElement, linkedArtifact.artifact.getUri()];
  });
}
