/*
 * Copyright 2018 Google LLC
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

import * as React from 'react';
import { commonCss } from '../Css';
import Button from '@material-ui/core/Button';
import InputAdornment from '@material-ui/core/InputAdornment';
import TextField from '@material-ui/core/TextField';
import { ApiParameter } from '../apis/pipeline';
import { stylesheet } from 'typestyle';
import { color, spacing } from '../Css';
import Editor from './Editor';

export interface NewRunParametersProps {
  initialParams: ApiParameter[];
  titleMessage: string;
  handleParamChange: (index: number, value: string) => void;
}

const css = stylesheet({
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
});

class NewRunParameters extends React.Component<NewRunParametersProps> {
  constructor(props: any) {
    super(props);
  }

  public render(): JSX.Element | null {
    const { handleParamChange, initialParams, titleMessage } = this.props;

    return (
      <div>
        <div className={commonCss.header}>Run parameters</div>
        <div>{titleMessage}</div>
        {!!initialParams.length && (
          <div>
            {initialParams.map((param, i) => {
              return (
                <EnhancedTextField
                  key={i}
                  param={param}
                  index={i}
                  handleParamChange={handleParamChange}
                />
              );
            })}
          </div>
        )}
      </div>
    );
  }
}

interface EnhancedTextFieldProps {
  param: ApiParameter;
  index: number;
  handleParamChange: (index: number, value: string) => void;
}

interface EnhancedTextFieldState {
  isEditorOpen: boolean;
  isInJsonForm: boolean;
  isJsonField: boolean;
}

class EnhancedTextField extends React.Component<EnhancedTextFieldProps, EnhancedTextFieldState> {
  public static getDerivedStateFromProps(
    nextProps: EnhancedTextFieldProps,
    prevState: EnhancedTextFieldState,
  ): EnhancedTextFieldState {
    let isJson = true;
    try {
      const displayValue = JSON.parse(nextProps.param.value || '');
      // Nulls, booleans, strings, and numbers can all be parsed as JSON, but we don't care
      // about rendering. Note that `typeOf null` returns 'object'
      if (displayValue === null || typeof displayValue !== 'object') {
        throw new Error('Parsed JSON was neither an array nor an object. Using default renderer');
      }
    } catch (err) {
      isJson = false;
    }
    return {
      isEditorOpen: prevState.isEditorOpen,
      isInJsonForm: isJson,
      isJsonField: prevState.isJsonField || isJson,
    };
  }

  constructor(props: any) {
    super(props);

    this.state = {
      isEditorOpen: false,
      isInJsonForm: false,
      isJsonField: false,
    };
  }

  public render(): JSX.Element | null {
    const { param, index, handleParamChange } = this.props;

    if (this.state.isJsonField) {
      return (
        <>
          <TextField
            id={`newRunPipelineParam${index}`}
            disabled={this.state.isEditorOpen}
            variant='outlined'
            label={param.name}
            value={param.value || ''}
            onChange={ev => handleParamChange(index, ev.target.value || '')}
            style={{ maxWidth: 600 }}
            className={commonCss.textField}
            InputProps={{
              classes: { disabled: css.nonEditableInput },
              endAdornment: (
                <InputAdornment position='end'>
                  <Button
                    color='secondary'
                    id='chooseExperimentBtn'
                    onClick={() => {
                      if (this.state.isInJsonForm) {
                        const displayValue = JSON.parse(param.value || '');
                        if (this.state.isEditorOpen) {
                          handleParamChange(index, JSON.stringify(displayValue) || '');
                        } else {
                          handleParamChange(index, JSON.stringify(displayValue, null, 2) || '');
                        }
                      }
                      this.setState({
                        isEditorOpen: !this.state.isEditorOpen,
                      });
                    }}
                    style={{ padding: '3px 5px', margin: 0 }}
                  >
                    {this.state.isEditorOpen ? 'Close Editor' : 'Open Editor'}
                  </Button>
                </InputAdornment>
              ),
              readOnly: false,
            }}
          />
          {this.state.isEditorOpen && (
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
                onChange={text => handleParamChange(index, text || '')}
                value={param.value || ''}
              />
            </div>
          )}
        </>
      );
    } else {
      return (
        <TextField
          id={`newRunPipelineParam${index}`}
          key={index}
          variant='outlined'
          label={param.name}
          value={param.value || ''}
          onChange={ev => handleParamChange(index, ev.target.value || '')}
          style={{ maxWidth: 600 }}
          className={commonCss.textField}
        />
      );
    }
  }
}

export default NewRunParameters;
