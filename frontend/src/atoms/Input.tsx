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
import TextField, { TextFieldProps } from '@material-ui/core/TextField';
import { commonCss } from '../Css';
import { sanitizeProps } from '../lib/Utils';

interface InputProps extends TextFieldProps {
  height?: number | string;
  width?: number;
}

export default (props: InputProps) => {
  const { height, width, ...rest } = props;
  return (
    <TextField variant='outlined' className={commonCss.textField} spellCheck={false}
        style={{
          height: !!props.multiline ? 'auto' : (height || 40),
          maxWidth: 600,
          width: width || '100%' }}
        {...sanitizeProps(rest)}>
      {props.children}
    </TextField>
  );
};
