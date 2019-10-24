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
import { stylesheet } from 'typestyle';
import { color, spacing, commonCss } from '../Css';
import { KeyValue } from '../lib/StaticGraphParser';
import Editor from './Editor';
import 'brace';
import 'brace/ext/language_tools';
import 'brace/mode/json';
import 'brace/theme/github';
import { S3Artifact } from 'third_party/argo-ui/argo_template';

export const css = stylesheet({
  key: {
    color: color.strong,
    flex: '0 0 50%',
    fontWeight: 'bold',
    maxWidth: 300,
  },
  row: {
    borderBottom: `1px solid ${color.divider}`,
    display: 'flex',
    padding: `${spacing.units(-5)}px ${spacing.units(-6)}px`,
  },
  valueJson: {
    flexGrow: 1,
  },
  valueText: {
    maxWidth: 400,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
});

interface DetailsTableProps {
  fields: Array<KeyValue<string | S3Artifact>>;
  title?: string;
  valueComponent?: React.FC<S3Artifact>;
}

function isString(x: any): x is string {
  return typeof x === 'string';
}

export default (props: DetailsTableProps) => {
  return (
    <React.Fragment>
      {!!props.title && <div className={commonCss.header}>{props.title}</div>}
      <div>
        {props.fields.map((f, i) => {
          const [key, value] = f;

          // only try to parse json if value is a string
          if (isString(value)) {
            try {
              const parsedJson = JSON.parse(value);
              // Nulls, booleans, strings, and numbers can all be parsed as JSON, but we don't care
              // about rendering. Note that `typeOf null` returns 'object'
              if (parsedJson === null || typeof parsedJson !== 'object') {
                throw new Error(
                  'Parsed JSON was neither an array nor an object. Using default renderer',
                );
              }
              return (
                <div key={i} className={css.row}>
                  <span className={css.key}>{key}</span>
                  <Editor
                    width='100%'
                    height='300px'
                    mode='json'
                    theme='github'
                    highlightActiveLine={true}
                    showGutter={true}
                    readOnly={true}
                    value={JSON.stringify(parsedJson, null, 2) || ''}
                  />
                </div>
              );
            } catch (err) {
              // do nothing
            }
          }
          // If the value isn't a JSON object, just display it as is
          return (
            <div key={i} className={css.row}>
              <span className={css.key}>{key}</span>
              <span className={css.valueText}>
                {props.valueComponent && !!value && !isString(value)
                  ? props.valueComponent(value)
                  : value}
              </span>
            </div>
          );
        })}
      </div>
    </React.Fragment>
  );
};
