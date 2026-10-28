/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { ComponentType } from 'react';
import type { KeyValue, ValueComponentProps } from 'src/lib/DetailsTableTypes';
import './RunInspection.css';

export interface InspectionFieldsProps<T> {
  fields: Array<KeyValue<string | T>>;
  title?: string;
  valueComponent?: ComponentType<ValueComponentProps<T>>;
  valueComponentProps?: Record<string, unknown>;
}

function structuredValue(value: unknown): string | undefined {
  if (typeof value === 'string') {
    try {
      const parsed: unknown = JSON.parse(value);
      if (parsed !== null && typeof parsed === 'object') return JSON.stringify(parsed, null, 2);
    } catch {
      // Ordinary text is displayed verbatim.
    }
  } else if (value !== null && typeof value === 'object') {
    return JSON.stringify(value, null, 2);
  }
  return undefined;
}

export function InspectionFields<T>({
  fields,
  title,
  valueComponent: ValueComponent,
  valueComponentProps,
}: InspectionFieldsProps<T>) {
  return (
    <section className='kfp-inspection-fields' aria-label={title}>
      {title && <h3>{title}</h3>}
      <dl>
        {fields.map(([name, value], index) => {
          const json =
            typeof value === 'string' || !ValueComponent ? structuredValue(value) : undefined;
          return (
            <div className='kfp-inspection-field' key={index}>
              <dt>{name}</dt>
              <dd>
                {json !== undefined ? (
                  <pre>{json}</pre>
                ) : ValueComponent && value !== undefined && value !== null ? (
                  <ValueComponent value={value} {...valueComponentProps} />
                ) : (
                  String(value ?? '')
                )}
              </dd>
            </div>
          );
        })}
      </dl>
    </section>
  );
}
