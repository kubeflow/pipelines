/*
 * Copyright 2022 The Kubeflow Authors
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
  ComponentInputsSpec_ParameterSpec,
  ParameterType_ParameterTypeEnum,
} from 'src/generated/pipeline_spec/pipeline_spec';

export type SpecParameters = { [key: string]: ComponentInputsSpec_ParameterSpec };
export type RuntimeParameters = { [key: string]: any };
export type ParameterErrorMessages = { [key: string]: string | null };

type InitialParameterState = {
  errorMessages: ParameterErrorMessages;
  isValid: boolean;
  runtimeParameters: RuntimeParameters;
  updatedParameters: RuntimeParameters;
};

type ClonedRuntimeConfig = {
  parameters?: RuntimeParameters;
  pipeline_root?: string;
};

const protoMap = new Map<string, string>([
  ['NUMBER_DOUBLE', 'double'],
  ['NUMBER_INTEGER', 'integer'],
  ['STRING', 'string'],
  ['BOOLEAN', 'boolean'],
  ['LIST', 'list'],
  ['STRUCT', 'dict'],
]);

export function convertInput(paramStr: string, paramType: ParameterType_ParameterTypeEnum): any {
  if (paramStr === '' && paramType !== ParameterType_ParameterTypeEnum.STRING) {
    return undefined;
  }
  switch (paramType) {
    case ParameterType_ParameterTypeEnum.BOOLEAN:
      if (paramStr === 'true' || paramStr === 'false') {
        return paramStr === 'true';
      }
      return null;
    case ParameterType_ParameterTypeEnum.STRING:
      return paramStr;
    case ParameterType_ParameterTypeEnum.NUMBER_INTEGER:
      if (Number.isInteger(Number(paramStr))) {
        return Number(paramStr);
      }
      return null;
    case ParameterType_ParameterTypeEnum.NUMBER_DOUBLE:
      if (!Number.isNaN(Number(paramStr))) {
        return Number(paramStr);
      }
      return null;
    case ParameterType_ParameterTypeEnum.LIST:
      if (!paramStr.trim().startsWith('[')) {
        return null;
      }
      try {
        return JSON.parse(paramStr);
      } catch (err) {
        return null;
      }
    case ParameterType_ParameterTypeEnum.STRUCT:
      if (!paramStr.trim().startsWith('{')) {
        return null;
      }
      try {
        return JSON.parse(paramStr);
      } catch (err) {
        return null;
      }
    default:
      console.log('Unknown paramter type: ' + paramType);
      return null;
  }
}

export function generateInputValidationErrMsg(
  parametersInRealType: any,
  paramType: ParameterType_ParameterTypeEnum,
  isOptional: boolean = false,
) {
  if (parametersInRealType === undefined && isOptional) {
    return null;
  }
  switch (parametersInRealType) {
    case undefined:
      return 'Missing parameter.';
    case null:
      return (
        'Invalid input. This parameter should be in ' +
        protoMap.get(ParameterType_ParameterTypeEnum[paramType]) +
        ' type'
      );
    default:
      return null;
  }
}

export function convertNonUserInputParamToString(
  specParameters: SpecParameters,
  key: string,
  value: any,
): string {
  let paramStr;
  if (!specParameters[key]) {
    return '';
  }
  switch (specParameters[key].parameterType) {
    case ParameterType_ParameterTypeEnum.STRUCT:
    case ParameterType_ParameterTypeEnum.LIST:
      paramStr = JSON.stringify(value);
      break;
    case ParameterType_ParameterTypeEnum.BOOLEAN:
    case ParameterType_ParameterTypeEnum.NUMBER_INTEGER:
    case ParameterType_ParameterTypeEnum.NUMBER_DOUBLE:
      paramStr = value.toString();
      break;
    default:
      paramStr = value;
  }
  return paramStr;
}

export function getInitialParameterState(
  specParameters: SpecParameters,
  clonedRuntimeConfig?: ClonedRuntimeConfig,
): InitialParameterState {
  if (clonedRuntimeConfig?.parameters) {
    const updatedParameters: RuntimeParameters = {};
    Object.entries(clonedRuntimeConfig.parameters).forEach(([key, value]) => {
      updatedParameters[key] = convertNonUserInputParamToString(specParameters, key, value);
    });

    return {
      errorMessages: {},
      isValid: true,
      runtimeParameters: clonedRuntimeConfig.parameters,
      updatedParameters,
    };
  }

  const updatedParameters: RuntimeParameters = {};
  const runtimeParameters: RuntimeParameters = {};
  const errorMessages: ParameterErrorMessages = {};
  let isValid = true;

  Object.keys(specParameters).forEach((key) => {
    if (specParameters[key].defaultValue !== undefined) {
      updatedParameters[key] = convertNonUserInputParamToString(
        specParameters,
        key,
        specParameters[key].defaultValue,
      );
      runtimeParameters[key] = convertInput(
        updatedParameters[key],
        specParameters[key].parameterType,
      );
      return;
    }

    isValid = false;
    errorMessages[key] = 'Missing parameter.';
  });

  return {
    errorMessages,
    isValid,
    runtimeParameters,
    updatedParameters,
  };
}
