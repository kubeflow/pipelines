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

import * as dagre from 'dagre';
import * as React from 'react';
import { shallow, mount } from 'enzyme';
import EnhancedGraph, { Graph } from './Graph';
import SuccessIcon from '@material-ui/icons/CheckCircle';
import Tooltip from '@material-ui/core/Tooltip';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => '' };
    return Component;
  },
}));
jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  useTranslation: () => {
    return {
      t: (str: any) => str,
      i18n: {
        changeLanguage: () => new Promise(() => {}),
      },
    };
  },
}));
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

beforeEach(() => {
  jest.restoreAllMocks();
});

describe('Graph', () => {
  it('handles an empty graph', () => {
    expect(shallow(<Graph graph={newGraph()} />)).toMatchSnapshot();
  });

  it('renders a graph with one node', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with two disparate nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with two connectd nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with two connectd nodes in reverse order', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node2', 'node1');
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
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

    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with colored nodes', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1', false, 'red'));
    graph.setNode('node2', newNode('node2', false, 'green'));
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with colored edges', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2', { color: 'red' });
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('renders a graph with a placeholder node and edge', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1', false));
    graph.setNode('node2', newNode('node2', true));
    graph.setEdge('node1', 'node2', { isPlaceholder: true });
    expect(shallow(<Graph graph={graph} />)).toMatchSnapshot();
  });

  it('calls onClick callback when node is clicked', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node2', 'node1');
    const spy = jest.fn();
    const tree = shallow(<Graph graph={graph} onClick={spy} />);
    tree
      .find('.node')
      .at(0)
      .simulate('click');
    expect(spy).toHaveBeenCalledWith('node1');
  });

  it('renders a graph with a selected node', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    expect(shallow(<Graph graph={graph} selectedNodeId='node1' />)).toMatchSnapshot();
  });

  it('gracefully renders a graph with a selected node id that does not exist', () => {
    const graph = newGraph();
    graph.setNode('node1', newNode('node1'));
    graph.setNode('node2', newNode('node2'));
    graph.setEdge('node1', 'node2');
    expect(shallow(<Graph graph={graph} selectedNodeId='node3' />)).toMatchSnapshot();
  });

  it('shows an error message when the graph is invalid', () => {
    const consoleErrorSpy = jest.spyOn(console, 'error');
    consoleErrorSpy.mockImplementation(() => null);
    const graph = newGraph();
    graph.setEdge('node1', 'node2');
    const onError = jest.fn();
    expect(mount(<EnhancedGraph graph={graph} onError={onError} />).html()).toMatchSnapshot();
    expect(onError).toHaveBeenCalledTimes(1);
    const [message, additionalInfo] = onError.mock.calls[0];
    expect(message).toEqual('errorRenderGraph');
    expect(additionalInfo).toEqual(
      "errorRenderGraph bugKubeflowError: 't is not a function'.",
    );
  });
});
