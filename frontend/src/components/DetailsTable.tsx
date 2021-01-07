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
import { useTranslation } from 'react-i18next';

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
    // flexGrow expands value text to full width.
    flexGrow: 1,
    // For current use-cases, value text shouldn't be very long. It will be not readable when it's long.
    // Therefore, it's easier we just show it completely in the UX.
    overflow: 'hidden',
    // Sometimes, urls will be unbreakable for a long string. overflow-wrap: break-word
    // allows breaking an url at middle of a word so it will not overflow and get hidden.
    overflowWrap: 'break-word',
  },
});

export interface ValueComponentProps<T> {
  value?: string | T;
  [key: string]: any;
}

interface DetailsTableProps<T> {
  fields: Array<KeyValue<string | T>>;
  title?: string;
  valueComponent?: React.FC<ValueComponentProps<T>>;
  valueComponentProps?: { [key: string]: any };
}

function isString(x: any): x is string {
  return typeof x === 'string';
}

const DetailsTable = <T extends {}>(props: DetailsTableProps<T>) => {
  const { fields, title, valueComponent: ValueComponent, valueComponentProps } = props;
  const { t } = useTranslation('common');
  return (
    <React.Fragment>
      {!!title && <div className={commonCss.header}>{title}</div>}
      <div>
        {fields.map((f, i) => {
          const [key, value] = f;

          // only try to parse json if value is a string
          if (isString(value)) {
            try {
              const parsedJson = JSON.parse(value);
              // Nulls, booleans, strings, and numbers can all be parsed as JSON, but we don't care
              // about rendering. Note that `typeOf null` returns 'object'
              if (parsedJson === null || typeof parsedJson !== 'object') {
                throw new Error(t('parsedJsonNotArrayObject'));
              }
              return (
                <div key={i} className={css.row}>
                  <span className={css.key}>{key}</span>
                  <Editor
                    width='100%'
                    minLines={3}
                    maxLines={20}
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
          // If a ValueComponent and a value is provided, render the value with
          // the ValueComponent. Otherwise render the value as a string (empty string if null or undefined).
          return (
            <div key={i} className={css.row}>
              <span className={css.key}>{key}</span>
              <span className={css.valueText}>
                {ValueComponent && value ? (
                  <ValueComponent value={value} {...valueComponentProps} />
                ) : (
                  `${value || ''}`
                )}
              </span>
            </div>
          );
        })}
      </div>
    </React.Fragment>
  );
};

export default DetailsTable;
