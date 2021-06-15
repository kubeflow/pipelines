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

import {
  Artifact,
  ArtifactCustomProperties,
  ArtifactProperties,
  Execution,
  getMetadataValue,
  getResourceProperty,
} from '@kubeflow/frontend';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import React from 'react';
import { useQuery } from 'react-query';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { color, commonCss, padding, spacing } from 'src/Css';
import { getInputArtifactsInExecution, getOutputArtifactsInExecution } from 'src/lib/MlmdUtils';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { stylesheet } from 'typestyle';
import { ArtifactLink } from '../ArtifactLink';
import Banner from '../Banner';
import DetailsTable, { ValueComponentProps } from '../DetailsTable';
import { ExecutionTitle } from './ExecutionTitle';

const css = stylesheet({
  key: {
    color: color.strong,
    flex: '0 0 50%',
    fontWeight: 'bold',
    maxWidth: 300,
  },
  row: {
    borderBottom: `1px solid ${color.divider}`,
    display: 'flex',
    padding: `${spacing.units(-5)}px ${spacing.units(-6)}px`,
  },
  valueText: {
    flexGrow: 1,
    overflow: 'hidden',
    overflowWrap: 'break-word',
  },
});

type ParamList = Array<KeyValue<string>>;

export interface IOTabProps {
  execution: Execution;
}

export function InputOutputTab({ execution }: IOTabProps) {
  const executionId = execution.getId();

  // Retrieves input and output artifacts from Metadata store.
  const {
    isSuccess: isOutputArtifactSuccess,
    error: outputArtifactError,
    data: outputArtifactData,
  } = useQuery<Artifact[], Error>(
    ['execution_output_artifact', { id: executionId }],
    () => getOutputArtifactsInExecution(execution),
    { staleTime: Infinity },
  );

  const {
    isSuccess: isInputArtifactSuccess,
    error: inputArtifactError,
    data: inputArtifactData,
  } = useQuery<Artifact[], Error>(
    ['execution_input_artifact', { id: executionId }],
    () => getInputArtifactsInExecution(execution),
    { staleTime: Infinity },
  );

  // Restructs artifacts and parameters for visualization.
  const inputParams: ParamList = extractInputFromExecution(execution);
  const outputParams: ParamList = extractOutputFromExecution(execution);
  let inputArtifacts: ParamList = [];
  if (isInputArtifactSuccess && inputArtifactData) {
    inputArtifacts = getArtifactParamList(inputArtifactData);
  }
  let outputArtifacts: ParamList = [];
  if (isOutputArtifactSuccess && outputArtifactData) {
    outputArtifacts = getArtifactParamList(outputArtifactData);
  }

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <ExecutionTitle execution={execution} />

          {outputArtifactError && (
            <Banner
              message='Error in retrieving output Artifacts.'
              mode='error'
              additionalInfo={outputArtifactError.message}
            />
          )}
          {!outputArtifactError && inputArtifactError && (
            <Banner
              message='Error in retrieving input Artifacts.'
              mode='error'
              additionalInfo={inputArtifactError.message}
            />
          )}

          <div className={commonCss.header}>{'Input'}</div>
          <DetailsTable
            key={`input-parameters-${executionId}`}
            title='Parameters'
            fields={inputParams}
          />

          <ArtifactPreview
            key={`input-artifacts-${executionId}`}
            title='Artifacts'
            fields={inputArtifacts}
          />

          <div className={padding(20, 'tb')} />
          <hr />

          <div className={commonCss.header}>{'Output'}</div>
          <DetailsTable
            key={`output-parameters-${executionId}`}
            title='Parameters'
            fields={outputParams}
          />

          <ArtifactPreview
            key={`output-artifacts-${executionId}`}
            title='Artifacts'
            fields={outputArtifacts}
          />
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

function getArtifactParamList(inputArtifactData: Artifact[]): ParamList {
  return inputArtifactData.map(artifact => [
    (getResourceProperty(artifact, ArtifactProperties.NAME) ||
      getResourceProperty(artifact, ArtifactCustomProperties.NAME, true) ||
      '') as string,
    artifact.getUri(),
  ]);
}

interface ArtifactPreviewProps extends ValueComponentProps<string> {
  fields: Array<KeyValue<string>>;
  title?: string;
}

const ArtifactPreview: React.FC<ArtifactPreviewProps> = (props: ArtifactPreviewProps) => {
  const { fields, title } = props;
  return (
    <>
      {!!title && <div className={commonCss.header2}>{title}</div>}
      <div>
        {fields.map((f, i) => {
          const [key, value] = f;
          if (typeof value === 'string') {
            return (
              <div key={i} className={css.row}>
                <span className={css.key}>{key}</span>
                <span className={css.valueText}>
                  <span>{value ? <ArtifactLink artifactUri={value} /> : `${value || ''}`}</span>
                </span>
              </div>
            );
          }
          return null;
        })}
      </div>
    </>
  );
};
