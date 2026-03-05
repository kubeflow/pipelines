/*
 * Copyright 2018-2019 The Kubeflow Authors
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
import { render, screen } from '@testing-library/react';
import StaticNodeDetails from './StaticNodeDetails';
import { SelectedNodeInfo } from '../lib/StaticGraphParser';

describe('StaticNodeDetails', () => {
  function createNodeInfo(overrides: Partial<SelectedNodeInfo> = {}): SelectedNodeInfo {
    const info = new SelectedNodeInfo();
    return Object.assign(info, overrides);
  }

  it('renders nothing for unknown node type', () => {
    const nodeInfo = createNodeInfo({ nodeType: 'unknown' });
    const { container } = render(<StaticNodeDetails nodeInfo={nodeInfo} />);
    expect(container.querySelector('div')).toBeInTheDocument();
  });

  it('renders container node details', () => {
    const nodeInfo = createNodeInfo({
      nodeType: 'container',
      inputs: [['input-param', 'input-value']],
      outputs: [['output-param', 'output-value']],
      args: ['--flag', 'value'],
      command: ['python', 'train.py'],
      image: 'tensorflow/tensorflow:latest',
      volumeMounts: [['data-vol', '/mnt/data']],
    });
    render(<StaticNodeDetails nodeInfo={nodeInfo} />);

    expect(screen.getByText('Input parameters')).toBeInTheDocument();
    expect(screen.getByText('Output parameters')).toBeInTheDocument();
    expect(screen.getByText('Arguments')).toBeInTheDocument();
    expect(screen.getByText('--flag')).toBeInTheDocument();
    expect(screen.getByText('value')).toBeInTheDocument();
    expect(screen.getByText('Command')).toBeInTheDocument();
    expect(screen.getByText('python')).toBeInTheDocument();
    expect(screen.getByText('train.py')).toBeInTheDocument();
    expect(screen.getByText('Image')).toBeInTheDocument();
    expect(screen.getByText('tensorflow/tensorflow:latest')).toBeInTheDocument();
    expect(screen.getByText('Volume Mounts')).toBeInTheDocument();
  });

  it('renders resource node details', () => {
    const nodeInfo = createNodeInfo({
      nodeType: 'resource',
      inputs: [['param1', 'val1']],
      outputs: [['param2', 'val2']],
      resource: [['manifest-key', 'manifest-value']],
    });
    render(<StaticNodeDetails nodeInfo={nodeInfo} />);

    expect(screen.getByText('Input parameters')).toBeInTheDocument();
    expect(screen.getByText('Output parameters')).toBeInTheDocument();
    expect(screen.getByText('Manifest')).toBeInTheDocument();
  });

  it('renders condition when present', () => {
    const nodeInfo = createNodeInfo({
      condition: '{{inputs.parameters.status}} == "success"',
    });
    render(<StaticNodeDetails nodeInfo={nodeInfo} />);

    expect(screen.getByText('Condition')).toBeInTheDocument();
    expect(
      screen.getByText('Run when: {{inputs.parameters.status}} == "success"'),
    ).toBeInTheDocument();
  });

  it('does not render condition section when condition is empty', () => {
    const nodeInfo = createNodeInfo({ condition: '' });
    render(<StaticNodeDetails nodeInfo={nodeInfo} />);
    expect(screen.queryByText('Condition')).toBeNull();
  });

  it('renders container details with empty arrays', () => {
    const nodeInfo = createNodeInfo({
      nodeType: 'container',
      args: [],
      command: [],
      image: '',
      inputs: [[]],
      outputs: [[]],
      volumeMounts: [[]],
    });
    const { container } = render(<StaticNodeDetails nodeInfo={nodeInfo} />);
    expect(container).toBeInTheDocument();
  });
});
