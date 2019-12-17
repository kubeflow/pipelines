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

export interface NewRunParametersState {
  isBeingEdited: { [key: number]: boolean };
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

class NewRunParameters extends React.Component<NewRunParametersProps, NewRunParametersState> {
  constructor(props: any) {
    super(props);

    this.state = {
      isBeingEdited: {},
    };
  }

  public render(): JSX.Element | null {
    const { handleParamChange, initialParams, titleMessage } = this.props;

    this.props.initialParams.map((param: ApiParameter, i: number) => {
      try {
        const parsedJson = JSON.parse(param.value || '');
        // Nulls, booleans, strings, and numbers can all be parsed as JSON, but we don't care
        // about rendering. Note that `typeOf null` returns 'object'
        if (parsedJson === null || typeof parsedJson !== 'object') {
          throw new Error('Parsed JSON was neither an array nor an object. Using default renderer');
        }

        if (typeof this.state.isBeingEdited[i] === 'undefined') {
          this.state.isBeingEdited[i] = false;
        }
      } catch (err) {
        // do nothing
      }
    });

    return (
      <div>
        <div className={commonCss.header}>Run parameters</div>
        <div>{titleMessage}</div>
        {!!initialParams.length && (
          <div>
            {initialParams.map((param, i) => {
              if (typeof this.state.isBeingEdited[i] !== 'undefined') {
                return (
                  <div key={i}>
                    <TextField
                      id={`newRunPipelineParam${i}`}
                      disabled={this.state.isBeingEdited[i]}
                      variant='outlined'
                      label={param.name}
                      value={param.value || ''}
                      onChange={ev => handleParamChange(i, ev.target.value || '')}
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
                                const parsedJson = JSON.parse(param.value || '');
                                if (this.state.isBeingEdited[i]) {
                                  handleParamChange(i, JSON.stringify(parsedJson) || '');
                                  this.setState({
                                    isBeingEdited: {
                                      ...this.state.isBeingEdited,
                                      [i]: false,
                                    },
                                  });
                                } else {
                                  handleParamChange(i, JSON.stringify(parsedJson, null, 2) || '');
                                  this.setState({
                                    isBeingEdited: {
                                      ...this.state.isBeingEdited,
                                      [i]: true,
                                    },
                                  });
                                }
                              }}
                              style={{ padding: '3px 5px', margin: 0 }}
                            >
                              {this.state.isBeingEdited[i] ? 'Close Editor' : 'Open Editor'}
                            </Button>
                          </InputAdornment>
                        ),
                        readOnly: false,
                      }}
                    />
                    {this.state.isBeingEdited[i] && (
                      <div className={css.row}>
                        <Editor
                          width='100%'
                          height='300px'
                          mode='json'
                          theme='github'
                          highlightActiveLine={true}
                          showGutter={true}
                          readOnly={false}
                          onChange={text => handleParamChange(i, text || '')}
                          value={param.value || ''}
                        />
                      </div>
                    )}
                  </div>
                );
              } else {
                return (
                  <TextField
                    id={`newRunPipelineParam${i}`}
                    key={i}
                    variant='outlined'
                    label={param.name}
                    value={param.value || ''}
                    onChange={ev => handleParamChange(i, ev.target.value || '')}
                    style={{ maxWidth: 600 }}
                    className={commonCss.textField}
                  />
                );
              }
            })}
          </div>
        )}
      </div>
    );
  }
}

export default NewRunParameters;
