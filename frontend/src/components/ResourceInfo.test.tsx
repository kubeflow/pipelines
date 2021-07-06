/*
 * Copyright 2021 The Kubeflow Authors
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
import { ResourceInfo, ResourceType } from './ResourceInfo';
import { getByTestId, render, screen } from '@testing-library/react';
import { Artifact, Value } from 'src/third_party/mlmd';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';

describe('ResourceInfo', () => {
  it('renders metrics artifact', async () => {
    const metrics = new Artifact();
    metrics.getCustomPropertiesMap().set('name', new Value().setStringValue('metrics'));
    metrics.getCustomPropertiesMap().set(
      'confidenceMetrics',
      new Value().setStructValue(
        Struct.fromJavaScript({
          list: [
            {
              confidenceThreshold: 2,
              falsePositiveRate: 0,
              recall: 0,
            },
            {
              confidenceThreshold: 1,
              falsePositiveRate: 0,
              recall: 0.33962264150943394,
            },
            {
              confidenceThreshold: 0.9,
              falsePositiveRate: 0,
              recall: 0.6037735849056604,
            },
          ],
        }),
      ),
    );
    render(
      <ResourceInfo
        resourceType={ResourceType.ARTIFACT}
        resource={metrics}
        typeName='System.ClassificationMetrics'
      />,
    );
    expect(screen.getByRole('heading', { level: 1 }).textContent).toEqual(
      'Type: System.ClassificationMetrics',
    );
    expect(screen.getAllByRole('heading', { level: 2 }).map(h => h.textContent))
      .toMatchInlineSnapshot(`
      Array [
        "Properties",
        "Custom Properties",
      ]
    `);
    const keyEquals = (key: string) => (property: HTMLElement) => {
      return getByTestId(property, 'resource-info-property-key').textContent === key;
    };
    function value(property: HTMLElement): HTMLElement {
      return getByTestId(property, 'resource-info-property-value');
    }
    const nameProperty = screen.getAllByTestId('resource-info-property').find(keyEquals('name'));
    expect(nameProperty).toBeTruthy();
    expect(value(nameProperty!).textContent).toEqual('metrics');

    const confidenceMetricsProperty = screen
      .getAllByTestId('resource-info-property')
      .find(keyEquals('confidenceMetrics'));
    expect(value(confidenceMetricsProperty!).children).toMatchInlineSnapshot(`
      HTMLCollection [
        <pre>
          [
        {
          "confidenceThreshold": 2,
          "falsePositiveRate": 0,
          "recall": 0
        },
        {
          "confidenceThreshold": 1,
          "falsePositiveRate": 0,
          "recall": 0.33962264150943394
        },
        {
          "confidenceThreshold": 0.9,
          "falsePositiveRate": 0,
          "recall": 0.6037735849056604
        }
      ]
        </pre>,
      ]
    `);
  });
});
