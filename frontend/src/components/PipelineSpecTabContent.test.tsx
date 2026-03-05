/*
 * Copyright 2022 The Kubeflow Authors
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
import { render } from '@testing-library/react';
import { PipelineSpecTabContent } from './PipelineSpecTabContent';

describe('PipelineSpecTabContent', () => {
  it('renders with a JSON template string', () => {
    const jsonTemplate = JSON.stringify({ key: 'value', nested: { a: 1 } });
    const { container } = render(<PipelineSpecTabContent templateString={jsonTemplate} />);
    expect(container).toBeInTheDocument();
  });

  it('renders with a YAML template string', () => {
    const yamlTemplate = 'key: value\nnested:\n  a: 1\n';
    const { container } = render(<PipelineSpecTabContent templateString={yamlTemplate} />);
    expect(container).toBeInTheDocument();
  });

  it('renders with a simple YAML key-value', () => {
    const { container } = render(<PipelineSpecTabContent templateString='name: test' />);
    expect(container).toBeInTheDocument();
  });
});
