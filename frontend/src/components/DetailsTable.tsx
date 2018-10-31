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
import { color, spacing } from '../Css';

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
}

export default (props: DetailsTableProps) => {
  return (
    <div className={css.root}>
      {props.fields.map((f, i) => (
        <div key={i} className={css.row}>
          <span className={css.key}>{f[0]}</span>
          <span>{f[1]}</span>
        </div>
      ))}
    </div>
  );
};
