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

import { Button } from './ui/button';
import { Checkbox } from './ui/checkbox';
import { TextField } from './ui/text-field';
import './RunFormFields.css';
import * as React from 'react';
import { useState } from 'react';
import { V2beta1RuntimeConfig } from 'src/apisv2beta1/run';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { ParameterType_ParameterTypeEnum } from 'src/generated/pipeline_spec/pipeline_spec';
import {
  convertInput,
  generateInputValidationErrMsg,
  getInitialParameterState,
  protoMap,
  type InitialParameterState,
  type ParameterErrorMessages,
  type RuntimeParameters,
  type SpecParameters,
} from 'src/lib/NewRunParametersUtils';
import Editor from './LazyEditor';

interface NewRunParametersProps {
  titleMessage: string;
  pipelineRoot?: string;
  // ComponentInputsSpec_ParameterSpec
  specParameters: SpecParameters;
  clonedRuntimeConfig?: V2beta1RuntimeConfig;
  initialParameterState?: InitialParameterState;
  handlePipelineRootChange?: (pipelineRoot?: string) => void;
  handleParameterChange?: (parameters: RuntimeParameters) => void;
  setIsValidInput?: (isValid: boolean) => void;
}

function NewRunParametersV2(props: NewRunParametersProps) {
  const {
    specParameters,
    clonedRuntimeConfig,
    initialParameterState: providedInitialParameterState,
    handlePipelineRootChange,
    handleParameterChange,
    setIsValidInput,
  } = props;
  const clonedPipelineRoot = clonedRuntimeConfig?.pipeline_root;
  const [customPipelineRootChecked, setCustomPipelineRootChecked] = useState(!!clonedPipelineRoot);
  const [customPipelineRoot, setCustomPipelineRoot] = useState(
    clonedPipelineRoot ?? props.pipelineRoot,
  );
  const initialParameterState = React.useMemo(
    () =>
      providedInitialParameterState ??
      getInitialParameterState(specParameters, clonedRuntimeConfig),
    [clonedRuntimeConfig, providedInitialParameterState, specParameters],
  );
  const [errorMessages, setErrorMessages] = useState<ParameterErrorMessages>(
    initialParameterState.errorMessages,
  );
  const [updatedParameters, setUpdatedParameters] = useState<RuntimeParameters>(
    initialParameterState.updatedParameters,
  );

  return (
    <div className='kfp-run-form-fields kfp-parameter-fields'>
      <h3>Pipeline Root</h3>
      <div>
        Pipeline Root represents an artifact repository, refer to{' '}
        <ExternalLink href='https://www.kubeflow.org/docs/components/pipelines/concepts/pipeline-root/'>
          Pipeline Root Documentation
        </ExternalLink>
        .
      </div>

      <label className='kfp-form-check'>
        <Checkbox
          checked={customPipelineRootChecked}
          aria-label='Set custom pipeline root.'
          onCheckedChange={(checked) => {
            setCustomPipelineRootChecked(checked);
            if (!checked) {
              setCustomPipelineRoot(undefined);
              handlePipelineRootChange?.(undefined);
            }
          }}
        />
        Custom Pipeline Root
      </label>
      {customPipelineRootChecked && (
        <TextField
          id={'[pipeline-root]'}
          label={'pipeline-root'}
          value={customPipelineRoot || ''}
          onChange={(ev) => {
            setCustomPipelineRoot(ev.target.value);
            if (handlePipelineRootChange) {
              handlePipelineRootChange(ev.target.value);
            }
          }}
        />
      )}
      <h3>Run parameters</h3>
      <div>{props.titleMessage}</div>

      {!!Object.keys(specParameters).length && (
        <div>
          {Object.entries(specParameters).map(([k, v]) => {
            const param: Param = {
              key: `${k} - ${protoMap.get(ParameterType_ParameterTypeEnum[v.parameterType])}`,
              value: updatedParameters[k],
              type: v.parameterType,
              errorMsg: errorMessages[k],
              literals: v.literals,
            };

            return (
              <div key={k}>
                <ParamEditor
                  id={k}
                  onChange={(value) => {
                    const nextUpdatedParameters: RuntimeParameters = {
                      ...updatedParameters,
                      [k]: value,
                    };
                    setUpdatedParameters(nextUpdatedParameters);
                    const parametersInRealType: RuntimeParameters = {};
                    Object.entries(nextUpdatedParameters).forEach(([k1, paramStr]) => {
                      parametersInRealType[k1] = convertInput(
                        paramStr,
                        specParameters[k1].parameterType,
                      );
                    });
                    if (handleParameterChange) {
                      handleParameterChange(parametersInRealType);
                    }

                    const nextErrorMessages: ParameterErrorMessages = {
                      ...errorMessages,
                      [k]: generateInputValidationErrMsg(
                        parametersInRealType[k],
                        specParameters[k].parameterType,
                        specParameters[k].isOptional,
                      ),
                    };
                    setErrorMessages(nextErrorMessages);

                    if (setIsValidInput) {
                      setIsValidInput(
                        Object.values(nextErrorMessages).every(
                          (errorMessage) => errorMessage === null,
                        ),
                      );
                    }
                  }}
                  param={param}
                />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default NewRunParametersV2;

interface Param {
  key: string;
  value: any;
  type: ParameterType_ParameterTypeEnum;
  errorMsg: string | null;
  literals?: (string | number | boolean)[];
}

interface ParamEditorProps {
  id: string;
  onChange: (value: string) => void;
  param: Param;
}

interface ParamEditorState {
  isEditorOpen: boolean;
  isInJsonForm: boolean;
  isJsonField: boolean;
}

class ParamEditor extends React.Component<ParamEditorProps, ParamEditorState> {
  public static getDerivedStateFromProps(
    nextProps: ParamEditorProps,
    prevState: ParamEditorState,
  ): { isInJsonForm: boolean; isJsonField: boolean } {
    let isJson: boolean;
    let paramType = nextProps.param.type;

    switch (paramType) {
      case ParameterType_ParameterTypeEnum.LIST:
      case ParameterType_ParameterTypeEnum.STRUCT:
        isJson = true;
        break;
      case ParameterType_ParameterTypeEnum.STRING:
      case ParameterType_ParameterTypeEnum.BOOLEAN:
      case ParameterType_ParameterTypeEnum.NUMBER_INTEGER:
      case ParameterType_ParameterTypeEnum.NUMBER_DOUBLE:
        isJson = false;
        break;
      default:
        isJson = false;
    }

    return {
      isInJsonForm: isJson,
      isJsonField: prevState.isJsonField || isJson,
    };
  }

  public state = {
    isEditorOpen: false,
    isInJsonForm: false,
    isJsonField: false,
  };

  public render(): React.JSX.Element | null {
    const { id, onChange, param } = this.props;

    if (param.literals?.length) {
      const literalStrings = param.literals.map(String);
      // Option indices distinguish the unselected placeholder from a supported empty-string literal.
      const selectedIndex = param.value == null ? -1 : literalStrings.indexOf(String(param.value));
      return (
        <div className='kfp-native-field'>
          <label htmlFor={id}>{param.key}</label>
          <select
            id={id}
            value={selectedIndex < 0 ? '' : String(selectedIndex)}
            onChange={(event) => onChange(literalStrings[Number(event.target.value)])}
            aria-invalid={!!param.errorMsg}
            aria-describedby={param.errorMsg ? `${id}-error` : undefined}
          >
            {selectedIndex < 0 && (
              <option value='' disabled>
                Select a value
              </option>
            )}
            {literalStrings.map((literal, index) => (
              <option key={index} value={String(index)}>
                {literal === '' ? 'Empty string' : literal}
              </option>
            ))}
          </select>
          {param.errorMsg && (
            <div className='kfp-form-error' role='alert' id={`${id}-error`}>
              {param.errorMsg}
            </div>
          )}
        </div>
      );
    }

    const onClick = () => {
      if (this.state.isInJsonForm) {
        let paramType = param.type;
        let displayValue;
        switch (paramType) {
          case ParameterType_ParameterTypeEnum.LIST:
            displayValue = JSON.parse(param.value || '[]');
            break;
          case ParameterType_ParameterTypeEnum.STRUCT:
            displayValue = JSON.parse(param.value || '{}');
            break;
          default:
            // TODO(jlyaoyuli): If the type from PipelineSpec is either LIST or STURCT,
            // but the user-input or default value is invalid JSON form, show error message.
            displayValue = JSON.parse('');
        }

        // TODO(zijianjoy): JSON format needs to be struct or list type.
        if (this.state.isEditorOpen) {
          onChange(JSON.stringify(displayValue) || '');
        } else {
          onChange(JSON.stringify(displayValue, null, 2) || '');
        }
      }
      this.setState({
        isEditorOpen: !this.state.isEditorOpen,
      });
    };

    return (
      <>
        <TextField
          id={id}
          label={param.key}
          value={param.value ?? ''}
          disabled={this.state.isJsonField && this.state.isEditorOpen}
          onChange={(event) => onChange(event.target.value)}
          error={param.errorMsg}
          trailingContent={
            this.state.isJsonField ? (
              <Button variant='secondary' size='sm' onClick={onClick}>
                {this.state.isEditorOpen ? 'Close Json Editor' : 'Open Json Editor'}
              </Button>
            ) : undefined
          }
        />
        {this.state.isJsonField && this.state.isEditorOpen && (
          <div className='kfp-parameter-editor'>
            <Editor
              width='100%'
              minLines={3}
              maxLines={20}
              mode='json'
              theme='github'
              editorProps={{ $blockScrolling: Infinity }}
              highlightActiveLine={true}
              showGutter={true}
              readOnly={false}
              onChange={(text) => onChange(text || '')}
              value={param.value || ''}
            />
          </div>
        )}
      </>
    );
  }
}
