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

import 'codemirror/lib/codemirror.css';
import 'codemirror/mode/javascript/javascript.js';
import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color, spacing, commonCss } from '../Css';
import { UnControlled as CodeMirror } from 'react-codemirror2';

export const css = stylesheet({
  key: {
    color: color.strong,
    flex: '0 0 50%',
    fontWeight: 'bold',
  },
  root: {
    maxWidth: 700,
  },
  row: {
    borderBottom: `1px solid ${color.divider}`,
    display: 'flex',
    padding: `${spacing.units(-5)}px ${spacing.units(-6)}px`,
  },
});

interface DetailsTableProps {
  fields: string[][];
  title?: string;
}

export default (props: DetailsTableProps) => {
  return (<React.Fragment>
    {!!props.title && <div className={commonCss.header}>{props.title}</div>}
    <div className={css.root}>
      {props.fields.map((f, i) => {
        try{
          const parsedJson = JSON.parse(f[1]);
          return (
            <div key={i} className={css.row}>
              <span className={css.key}>{f[0]}</span>
              <CodeMirror
                  value={JSON.stringify(parsedJson, null, 2) || ''}
                  editorDidMount={(editor) => editor.refresh()}
                  options={{
                    lineNumbers: true,
                    lineWrapping: true,
                    mode: 'application/json',
                    readOnly: true,
                    theme: 'default',
                  }}
                />
            </div>
          );
        } catch (err) {
          // If this isn't JSON, just print it as is
          return (
            <div key={i} className={css.row}>
              <span className={css.key}>{f[0]}</span>
              <span>{f[1]}</span>
            </div>
          );
        }
      })}
    </div>
  </React.Fragment>
  );
};
