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
  Button,
  Checkbox,
  FormControl,
  FormControlLabel,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  TextField,
} from '@mui/material';
import * as React from 'react';
import { useState } from 'react';
import { PipelineSpecRuntimeConfig } from 'src/apis/run';
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
    <div>
      <div className={commonCss.header}>Pipeline Root</div>
      <div>
        Pipeline Root represents an artifact repository, refer to{' '}
        <ExternalLink href='https://www.kubeflow.org/docs/components/pipelines/concepts/pipeline-root/'>
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
          onChange={(ev) => {
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

    if (param.literals?.length) {
      const getLiteralString = (l: string | number | boolean) => String(l);
      const literalStrings = param.literals.map(getLiteralString);
      const stringValue = getLiteralString(param.value);
      const isValueInLiterals =
        param.value != null && param.value !== '' && literalStrings.includes(stringValue);
      const selectValue = isValueInLiterals ? stringValue : '';

      return (
        <FormControl variant='outlined' className={classes(commonCss.textField, css.textfield)}>
          <InputLabel id={`${id}-label`}>{param.key}</InputLabel>
          <Select
            labelId={`${id}-label`}
            id={id}
            value={selectValue}
            onChange={(ev) => onChange(String(ev.target.value))}
            label={param.key}
            aria-label={param.key}
            displayEmpty
            renderValue={(selected) =>
              selected === '' ? (
                <span style={{ color: '#aaa' }}>Select a value</span>
              ) : (
                String(selected)
              )
            }
          >
            {literalStrings.map((literal) => (
              <MenuItem key={literal} value={literal}>
                {literal}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
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
        {this.state.isJsonField ? (
          <TextField
            id={id}
            disabled={this.state.isEditorOpen}
            variant='outlined'
            label={param.key}
            value={param.value || ''}
            onChange={(ev) => onChange(ev.target.value || '')}
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
            onChange={(ev) => onChange(ev.target.value || '')}
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
