/*
 * Copyright 2018 The Kubeflow Authors
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

import * as dagre from 'dagre';
import * as React from 'react';
import { fireEvent, render } from '@testing-library/react';
import { vi } from 'vitest';
import EnhancedGraph, { Graph } from './Graph';
import SuccessIcon from '@material-ui/icons/CheckCircle';
import Tooltip from '@material-ui/core/Tooltip';

function newGraph(): dagre.graphlib.Graph {
  const graph = new dagre.graphlib.Graph();
  graph.setGraph({});
  graph.setDefaultEdgeLabel(() => ({}));
  return graph;
}

const testIcon = (
  <Tooltip title='Test icon tooltip'>
    <SuccessIcon />
  </Tooltip>
);

const newNode = (label: string, isPlaceHolder?: boolean, color?: string, icon?: JSX.Element) => ({
  bgColor: color,
  height: 10,
  icon: icon || testIcon,
  isPlaceholder: isPlaceHolder || false,
  label,
  width: 10,
});

describe('Graph', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('handles an empty graph', () => {
    const { container } = render(<Graph graph={newGraph()} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders a graph with one node', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with two disparate nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with two connectd nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with two connectd nodes in reverse order', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node2', 'node1');
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a complex graph with six nodes and seven edges', () => {
    const graph = newGraph();
    graph.setNode('flipcoin1', newNode('flipcoin1'));
    graph.setNode('tails1', newNode('tails1'));
    graph.setNode('heads1', newNode('heads1'));
    graph.setNode('flipcoin2', newNode('flipcoin2'));
    graph.setNode('heads2', newNode('heads2'));
    graph.setNode('tails2', newNode('tails2'));

    graph.setEdge('flipcoin1', 'tails1');
    graph.setEdge('flipcoin1', 'heads1');
    graph.setEdge('tails1', 'flipcoin2');
    graph.setEdge('tails1', 'heads2');
    graph.setEdge('tails1', 'tails2');
    graph.setEdge('flipcoin2', 'heads2');
    graph.setEdge('flipcoin2', 'tails2');

    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with colored nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1', false, 'red'));
    graph.setNode('node2', newNode('node2', false, 'green'));
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with colored edges', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2', { color: 'red' });
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a graph with a placeholder node and edge', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1', false));
    graph.setNode('node2', newNode('node2', true));
    graph.setEdge('node1', 'node2', { isPlaceholder: true });
    const { asFragment } = render(<Graph graph={graph} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('calls onClick callback when node is clicked', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node2', 'node1');
    const spy = vi.fn();
    const { container } = render(<Graph graph={graph} onClick={spy} />);
    const node = container.querySelector('.graphNode');
    expect(node).not.toBeNull();
    fireEvent.click(node!);
    expect(spy).toHaveBeenCalledWith('node1');
  });

  it('renders a graph with a selected node', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    const { asFragment } = render(<Graph graph={graph} selectedNodeId='node1' />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('gracefully renders a graph with a selected node id that does not exist', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    const { asFragment } = render(<Graph graph={graph} selectedNodeId='node3' />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('shows an error message when the graph is invalid', () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const graph = newGraph();
    graph.setEdge('node1', 'node2');
    const onError = vi.fn();
    const { container } = render(<EnhancedGraph graph={graph} onError={onError} />);
    expect(onError).toHaveBeenCalledTimes(1);
    const [message, additionalInfo] = onError.mock.calls[0];
    expect(message).toEqual('There was an error rendering the graph.');
    expect(additionalInfo).toEqual(
      "There was an error rendering the graph. This is likely a bug in Kubeflow Pipelines. Error message: 'Graph definition is invalid. Cannot get node by 'node1'.'.",
    );
    expect(container.innerHTML).toMatchSnapshot();
    consoleErrorSpy.mockRestore();
  });
});
