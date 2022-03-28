/*
 * Copyright 2019,2021 The Kubeflow Authors
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
  ArtifactCustomProperties,
  ArtifactProperties,
  ExecutionCustomProperties,
  ExecutionProperties,
} from './Api';
import { ArtifactTypeMap, ExecutionTypeMap } from './LineageApi';
import { Artifact, Execution, Value } from 'src/third_party/mlmd';
import { LineageTypedResource } from './LineageTypes';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { ArtifactHelpers, ExecutionHelpers } from './MlmdUtils';

const ARTIFACT_FIELD_REPOS = [ArtifactProperties, ArtifactCustomProperties];
const EXECUTION_FIELD_REPOS = [ExecutionProperties, ExecutionCustomProperties];

type RepoType =
  | typeof ArtifactCustomProperties
  | typeof ArtifactProperties
  | typeof ExecutionCustomProperties
  | typeof ExecutionProperties;

export function getResourceProperty(
  resource: Artifact | Execution,
  propertyName: string,
  fromCustomProperties = false,
): string | number | Struct | null {
  const props = fromCustomProperties
    ? resource.getCustomPropertiesMap()
    : resource.getPropertiesMap();

  return (props && props.get(propertyName) && getMetadataValue(props.get(propertyName))) || null;
}

export function getResourcePropertyViaFallBack(
  res: Artifact | Execution,
  fieldRepos: RepoType[],
  fields: string[],
): string {
  const prop =
    fields.reduce(
      (value: string, field: string) =>
        value ||
        fieldRepos.reduce(
          (v: string, repo: RepoType, isCustomProp) =>
            v || // eslint-disable-next-line no-sequences
            ((field in repo && getResourceProperty(res, repo[field], !!isCustomProp)) as string),
          '',
        ),
      '',
    ) || '';
  return prop as string;
}

export function getResourceName(typedResource: LineageTypedResource): string {
  return typedResource.type === 'artifact'
    ? ArtifactHelpers.getName(typedResource.resource)
    : ExecutionHelpers.getName(typedResource.resource);
}

/**
 * Promisified sleep operation
 * @param t Time to sleep for in ms
 */
export const sleep = (t: number): Promise<void> => new Promise(res => setTimeout(res, t));

export function getResourceDescription(typedResource: LineageTypedResource): string {
  return getResourcePropertyViaFallBack(
    typedResource.resource,
    typedResource.type === 'artifact' ? ARTIFACT_FIELD_REPOS : EXECUTION_FIELD_REPOS,
    ['RUN_ID', 'RUN', 'PIPELINE_NAME', 'WORKSPACE'],
  );
}

export function getTypeName(typeId: number, artifactTypes: ArtifactTypeMap): string {
  return artifactTypes && artifactTypes.get(typeId!)
    ? artifactTypes.get(typeId!)!.getName()
    : String(typeId);
}

export function getExecutionTypeName(typeId: number, executionTypes: ExecutionTypeMap): string {
  return executionTypes && executionTypes.get(typeId!)
    ? executionTypes.get(typeId!)!.getName()
    : String(typeId);
}

export function getMetadataValue(value?: Value): string | number | Struct | undefined {
  if (!value) {
    return '';
  }

  switch (value.getValueCase()) {
    case Value.ValueCase.DOUBLE_VALUE:
      return value.getDoubleValue();
    case Value.ValueCase.INT_VALUE:
      return value.getIntValue();
    case Value.ValueCase.STRING_VALUE:
      return value.getStringValue();
    case Value.ValueCase.STRUCT_VALUE:
      return value.getStructValue();
    case Value.ValueCase.VALUE_NOT_SET:
      return '';
  }
}
