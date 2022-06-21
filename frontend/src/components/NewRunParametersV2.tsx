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
import { ExternalLink } from 'src/atoms/ExternalLink';
import { ParameterType_ParameterTypeEnum } from 'src/generated/pipeline_spec/pipeline_spec';
import { RuntimeParameters, SpecParameters } from 'src/pages/NewRunV2';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, spacing } from '../Css';
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
  handlePipelineRootChange?: (pipelineRoot: string) => void;
  handleParameterChange?: (parameters: RuntimeParameters) => void;
}

function NewRunParametersV2(props: NewRunParametersProps) {
  const [customPipelineRootChecked, setCustomPipelineRootChecked] = useState(false);
  const [customPipelineRoot, setCustomPipelineRoot] = useState(props.pipelineRoot);

  const [updatedParameters, setUpdatedParameters] = useState({});
  const inputConverter = (paramStr: string) => {
    if (paramStr === 'True') {
      return true;
    } else if (paramStr === 'False') {
      return false;
    } else if (Number(paramStr)) {
      return Number(paramStr);
    } else {
      return paramStr;
    }
  };

  useEffect(() => {
    const runtimeParametersWithDefault: RuntimeParameters = {};
    Object.keys(props.specParameters).map(key => {
      if (props.specParameters[key].defaultValue) {
        // TODO(zijianjoy): Make sure to consider all types of parameters.
        runtimeParametersWithDefault[key] = {
          string_value: props.specParameters[key].defaultValue,
        };
      }
    });
    setUpdatedParameters(runtimeParametersWithDefault);
  }, [props.specParameters]);

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
          }}
          className={classes(commonCss.textField, css.textfield)}
        />
      )}
      <div className={commonCss.header}>Run parameters</div>
      <div>{props.titleMessage}</div>

      {!!Object.keys(props.specParameters).length && (
        <div>
          {Object.entries(props.specParameters).map(([k, v]) => {
            const param = {
              key: `${k} - ${ParameterType_ParameterTypeEnum[v.parameterType]}`,
              value: updatedParameters[k],
            };

            return (
              <ParamEditor
                key={k}
                id={k}
                onChange={value => {
                  const nextUpdatedParameters: RuntimeParameters = {};
                  Object.assign(nextUpdatedParameters, updatedParameters);
                  nextUpdatedParameters[k] = { string_value: value };
                  setUpdatedParameters(nextUpdatedParameters);
                  if (props.handleParameterChange) {
                    let parametersInRealType = {};
                    Object.entries(nextUpdatedParameters).map(([k, v]) => {
                      parametersInRealType[k] = inputConverter(v['string_value']);
                    });
                    props.handleParameterChange(parametersInRealType);
                  }
                }}
                param={param}
              />
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
    try {
      const displayValue = JSON.parse('');
      // Nulls, booleans, strings, and numbers can all be parsed as JSON, but we don't care
      // about rendering. Note that `typeOf null` returns 'object'
      if (displayValue === null || typeof displayValue !== 'object') {
        throw new Error('Parsed JSON was neither an array nor an object. Using default renderer');
      }
    } catch (err) {
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
        // TODO(zijianjoy): JSON format needs to be struct or list type.
        const displayValue = JSON.parse(param.value?.string_value || '');
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
            // label={param.name}
            // value={param.value || ''}
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
            value={param.value?.string_value || ''}
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
              value={param.value?.string_value || ''}
            />
          </div>
        )}
      </>
    );
  }
}
