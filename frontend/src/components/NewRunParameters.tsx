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
import { classes, stylesheet } from 'typestyle';
import { color, spacing } from '../Css';
import Editor from './Editor';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export interface NewRunParametersProps {
  initialParams: ApiParameter[];
  titleMessage: string;
  handleParamChange: (index: number, value: string) => void;
  t: TFunction;
}

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

class NewRunParameters extends React.Component<NewRunParametersProps> {
  public render(): JSX.Element | null {
    const { handleParamChange, initialParams, titleMessage, t } = this.props;

    return (
      <div>
        <div className={commonCss.header}>{t('runParams')}</div>
        <div>{titleMessage}</div>
        {!!initialParams.length && (
          <div>
            {initialParams.map((param, i) => {
              return (
                <ParamEditor
                  key={i}
                  id={`newRunPipelineParam${i}`}
                  onChange={(value: string) => handleParamChange(i, value)}
                  param={param}
                  t={t}
                />
              );
            })}
          </div>
        )}
      </div>
    );
  }
}

interface ParamEditorProps {
  id: string;
  onChange: (value: string) => void;
  param: ApiParameter;
  t: TFunction;
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
    const { id, onChange, param, t } = this.props;

    const onClick = () => {
      if (this.state.isInJsonForm) {
        const displayValue = JSON.parse(param.value || '');
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
            label={param.name}
            value={param.value || ''}
            onChange={ev => onChange(ev.target.value || '')}
            className={classes(commonCss.textField, css.textfield)}
            InputProps={{
              classes: { disabled: css.nonEditableInput },
              endAdornment: (
                <InputAdornment position='end'>
                  <Button className={css.button} color='secondary' onClick={onClick}>
                    {this.state.isEditorOpen ? t('closeJsonEditor') : t('openJsonEditor')}
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
            label={param.name}
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

export default withTranslation('experiments')(NewRunParameters);
