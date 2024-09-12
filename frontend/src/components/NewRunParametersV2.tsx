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

import { Button, Checkbox, FormControlLabel, InputAdornment, TextField } from '@material-ui/core';
import * as React from 'react';
import { useEffect, useState } from 'react';
import { PipelineSpecRuntimeConfig } from 'src/apis/run';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { ParameterType_ParameterTypeEnum } from 'src/generated/pipeline_spec/pipeline_spec';
import { RuntimeParameters, SpecParameters } from 'src/pages/NewRunV2';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, spacing, padding } from '../Css';
import Editor from './Editor';

const css = stylesheet({
  button: {
    margin: 0,
    padding: '3px 5px',
  },
  key: {
    color: color.strong,
    flex: '0 0 50%',
    fontWeight: 'bold',
    maxWidth: 300,
  },
  nonEditableInput: {
    color: color.secondaryText,
  },
  row: {
    borderBottom: `1px solid ${color.divider}`,
    display: 'flex',
    padding: `${spacing.units(-5)}px ${spacing.units(-6)}px`,
  },
  textfield: {
    maxWidth: 600,
  },
});

interface NewRunParametersProps {
  titleMessage: string;
  pipelineRoot?: string;
  // ComponentInputsSpec_ParameterSpec
  specParameters: SpecParameters;
  clonedRuntimeConfig?: PipelineSpecRuntimeConfig;
  handlePipelineRootChange?: (pipelineRoot?: string) => void;
  handleParameterChange?: (parameters: RuntimeParameters) => void;
  setIsValidInput?: (isValid: boolean) => void;
}

const protoMap = new Map<string, string>([
  ['NUMBER_DOUBLE', 'double'],
  ['NUMBER_INTEGER', 'integer'],
  ['STRING', 'string'],
  ['BOOLEAN', 'boolean'],
  ['LIST', 'list'],
  ['STRUCT', 'dict'],
]);

function convertInput(paramStr: string, paramType: ParameterType_ParameterTypeEnum): any {
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
      // TODO: (jlyaoyuli) Validate if the type of parameters matches the value
      // If it doesn't throw an error message next to the TextField.
      console.log('Unknown paramter type: ' + paramType);
      return null;
  }
}

function generateInputValidationErrMsg(
  parametersInRealType: any,
  paramType: ParameterType_ParameterTypeEnum,
) {
  let errorMessage;
  switch (parametersInRealType) {
    case undefined:
      errorMessage = 'Missing parameter.';
      break;
    // TODO(jlyaoyuli): tell the error difference between mismatch type or invalid JSON form.
    case null:
      errorMessage =
        'Invalid input. This parameter should be in ' +
        protoMap.get(ParameterType_ParameterTypeEnum[paramType]) +
        ' type';
      break;
    default:
      errorMessage = null;
  }
  return errorMessage;
}

function convertNonUserInputParamToString(
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

function NewRunParametersV2(props: NewRunParametersProps) {
  const {
    specParameters,
    clonedRuntimeConfig,
    handlePipelineRootChange,
    handleParameterChange,
    setIsValidInput,
  } = props;
  const [customPipelineRootChecked, setCustomPipelineRootChecked] = useState(false);
  const [customPipelineRoot, setCustomPipelineRoot] = useState(props.pipelineRoot);
  const [errorMessages, setErrorMessages] = useState<string[]>([]);

  const [updatedParameters, setUpdatedParameters] = useState({});
  useEffect(() => {
    if (clonedRuntimeConfig && clonedRuntimeConfig.parameters) {
      const clonedRuntimeParametersStr: RuntimeParameters = {};
      // Convert cloned parameter to string type first to avoid error from convertInput
      Object.entries(clonedRuntimeConfig.parameters).forEach(entry => {
        clonedRuntimeParametersStr[entry[0]] = convertNonUserInputParamToString(
          specParameters,
          entry[0],
          entry[1],
        );
      });
      setUpdatedParameters(clonedRuntimeParametersStr);
      // Directly using cloned paramters guarantees input is valid and no error message
      setErrorMessages([]);
      if (setIsValidInput) {
        setIsValidInput(true);
      }

      if (clonedRuntimeConfig && handleParameterChange) {
        handleParameterChange(clonedRuntimeConfig.parameters);
      }
      return;
    }
    // TODO(jlyaoyuli): If we have parameters from run, put original default value next to the paramKey
    const runtimeParametersWithDefault: RuntimeParameters = {};
    let allParamtersWithDefault = true;
    let errMsg: string[] = [];
    Object.keys(specParameters).forEach(key => {
      if (specParameters[key].defaultValue) {
        // TODO(zijianjoy): Make sure to consider all types of parameters.
        // Convert default value to string type first to avoid error from convertInput
        runtimeParametersWithDefault[key] = convertNonUserInputParamToString(
          specParameters,
          key,
          specParameters[key].defaultValue,
        );
      } else {
        allParamtersWithDefault = false;
        errMsg[key] = 'Missing parameter.';
      }
    });
    setUpdatedParameters(runtimeParametersWithDefault);
    setErrorMessages(errMsg);
    if (setIsValidInput) {
      setIsValidInput(allParamtersWithDefault);
    }
  }, [clonedRuntimeConfig, specParameters, handleParameterChange, setIsValidInput]);

  useEffect(() => {
    if (clonedRuntimeConfig?.pipeline_root) {
      setCustomPipelineRootChecked(true);
      setCustomPipelineRoot(clonedRuntimeConfig.pipeline_root);
    }
  }, [clonedRuntimeConfig]);

  return (
    <div>
      <div className={commonCss.header}>Pipeline Root</div>
      <div>
        Pipeline Root represents an artifact repository, refer to{' '}
        <ExternalLink href='https://www.kubeflow.org/docs/components/pipelines/overview/pipeline-root/'>
          Pipeline Root Documentation
        </ExternalLink>
        .
      </div>

      <div>
        <FormControlLabel
          label='Custom Pipeline Root'
          control={
            <Checkbox
              color='primary'
              checked={customPipelineRootChecked}
              onChange={(event, checked) => {
                setCustomPipelineRootChecked(checked);
                if (!checked) {
                  setCustomPipelineRoot(undefined);
                  if (handlePipelineRootChange) {
                    handlePipelineRootChange(undefined);
                  }
                }
              }}
              inputProps={{ 'aria-label': 'Set custom pipeline root.' }}
            />
          }
        />
      </div>
      {customPipelineRootChecked && (
        <TextField
          id={'[pipeline-root]'}
          variant='outlined'
          label={'pipeline-root'}
          value={customPipelineRoot || ''}
          onChange={ev => {
            setCustomPipelineRoot(ev.target.value);
            if (handlePipelineRootChange) {
              handlePipelineRootChange(ev.target.value);
            }
          }}
          className={classes(commonCss.textField, css.textfield)}
        />
      )}
      <div className={commonCss.header}>Run parameters</div>
      <div>{props.titleMessage}</div>

      {!!Object.keys(specParameters).length && (
        <div>
          {Object.entries(specParameters).map(([k, v]) => {
            const param = {
              key: `${k} - ${protoMap.get(ParameterType_ParameterTypeEnum[v.parameterType])}`,
              value: updatedParameters[k],
              type: v.parameterType,
              errorMsg: errorMessages[k],
            };

            return (
              <div>
                <ParamEditor
                  key={k}
                  id={k}
                  onChange={value => {
                    let allInputsValid: boolean = true;
                    let parametersInRealType: RuntimeParameters = {};
                    const nextUpdatedParameters: RuntimeParameters = {};

                    Object.assign(nextUpdatedParameters, updatedParameters);
                    nextUpdatedParameters[k] = value;
                    setUpdatedParameters(nextUpdatedParameters);
                    Object.entries(nextUpdatedParameters).forEach(([k1, paramStr]) => {
                      parametersInRealType[k1] = convertInput(
                        paramStr,
                        specParameters[k1].parameterType,
                      );
                    });
                    if (handleParameterChange) {
                      handleParameterChange(parametersInRealType);
                    }

                    errorMessages[k] = generateInputValidationErrMsg(
                      parametersInRealType[k],
                      specParameters[k].parameterType,
                    );
                    setErrorMessages(errorMessages);

                    Object.values(errorMessages).forEach(errorMessage => {
                      allInputsValid = allInputsValid && errorMessage === null;
                    });

                    if (setIsValidInput) {
                      setIsValidInput(allInputsValid);
                    }
                  }}
                  param={param}
                />
                <div className={classes(padding(20, 'r'))} style={{ color: 'red' }}>
                  {param.errorMsg}
                </div>
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
  errorMsg: string;
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
    let isJson = true;
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

  public render(): JSX.Element | null {
    const { id, onChange, param } = this.props;

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
        {this.state.isJsonField ? (
          <TextField
            id={id}
            disabled={this.state.isEditorOpen}
            variant='outlined'
            label={param.key}
            value={param.value || ''}
            onChange={ev => onChange(ev.target.value || '')}
            className={classes(commonCss.textField, css.textfield)}
            InputProps={{
              classes: { disabled: css.nonEditableInput },
              endAdornment: (
                <InputAdornment position='end'>
                  <Button className={css.button} color='secondary' onClick={onClick}>
                    {this.state.isEditorOpen ? 'Close Json Editor' : 'Open Json Editor'}
                  </Button>
                </InputAdornment>
              ),
              readOnly: false,
            }}
          />
        ) : (
          <TextField
            id={id}
            variant='outlined'
            label={param.key}
            //TODO(zijianjoy): Convert defaultValue to correct type.
            value={param.value || ''}
            onChange={ev => onChange(ev.target.value || '')}
            className={classes(commonCss.textField, css.textfield)}
          />
        )}
        {this.state.isJsonField && this.state.isEditorOpen && (
          <div className={css.row}>
            <Editor
              width='100%'
              minLines={3}
              maxLines={20}
              mode='json'
              theme='github'
              highlightActiveLine={true}
              showGutter={true}
              readOnly={false}
              onChange={text => onChange(text || '')}
              value={param.value || ''}
            />
          </div>
        )}
      </>
    );
  }
}
