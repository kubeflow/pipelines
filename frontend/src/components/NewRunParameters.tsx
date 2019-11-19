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
import TextField from '@material-ui/core/TextField';
import { ApiParameter } from '../apis/pipeline';

export interface NewRunParametersProps {
  initialParams: ApiParameter[];
  titleMessage: string;
  handleParamChange: (index: number, value: string) => void;
}

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
            {initialParams.map((param, i) => (
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
            ))}
          </div>
        )}
      </div>
    );
  }
}

export default NewRunParameters;
