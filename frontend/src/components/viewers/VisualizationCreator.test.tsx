/*
 * Copyright 2019 Google LLC
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
import { shallow } from 'enzyme';
import { PlotType } from './Viewer';
import VisualizationCreator, { VisualizationCreatorConfig } from './VisualizationCreator';
import { ApiVisualizationType } from '../../apis/visualization';

describe('VisualizationCreator', () => {
  it('does not render component when no config is provided', () => {
    const tree = shallow(<VisualizationCreator configs={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders component when empty config is provided', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders component when isBusy is not provided', () => {
    const config: VisualizationCreatorConfig = {
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders component when onGenerate is not provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders component when all parameters in config are provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    expect(tree).toMatchSnapshot();
  });

  it('has a disabled BusyButton if selectedType is undefined', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      inputPath: 'gs://ml-pipeline/data.csv',
    });
    expect(tree.find('BusyButton').at(0).prop('disabled')).toBe(true);
  });

  it('has a disabled BusyButton if inputPath is an empty string', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      // inputPath by default is set to ''
      selectedType: ApiVisualizationType.CURVE,
    });
    expect(tree.find('BusyButton').at(0).prop('disabled')).toBe(true);
  });

  it('has a disabled BusyButton if onGenerate is not provided as a prop', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      inputPath: 'gs://ml-pipeline/data.csv',
      selectedType: ApiVisualizationType.CURVE,
    });
    expect(tree.find('BusyButton').at(0).prop('disabled')).toBe(true);
  });

  it('has a disabled BusyButton if isBusy is true', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: true,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      inputPath: 'gs://ml-pipeline/data.csv',
      selectedType: ApiVisualizationType.CURVE,
    });
    expect(tree.find('BusyButton').at(0).prop('disabled')).toBe(true);
  });

  it('has an enabled BusyButton if onGenerate is provided and inputPath and selectedType are set', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      inputPath: 'gs://ml-pipeline/data.csv',
      selectedType: ApiVisualizationType.CURVE,
    });
    expect(tree.find('BusyButton').at(0).prop('disabled')).toBe(false);
  });

  it('calls onGenerate when BusyButton is clicked', () => {
    const onGenerate = jest.fn();
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      arguments: '{}',
      inputPath: 'gs://ml-pipeline/data.csv',
      selectedType: ApiVisualizationType.CURVE,
    });
    tree.find('BusyButton').at(0).simulate('click');
    expect(onGenerate).toBeCalled();
  });

  it('passes state as parameters to onGenerate when BusyButton is clicked', () => {
    const onGenerate = jest.fn();
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      arguments: '{}',
      inputPath: 'gs://ml-pipeline/data.csv',
      selectedType: ApiVisualizationType.CURVE,
    });
    tree.find('BusyButton').at(0).simulate('click');
    expect(onGenerate).toBeCalledWith('{}', 'gs://ml-pipeline/data.csv', ApiVisualizationType.CURVE);
  });

  it('renders the provided arguments correctly', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      arguments: JSON.stringify({is_generated: 'True'}),
    });
    expect(tree).toMatchSnapshot();
  });

  it('renders the provided input path correctly', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      inputPath: 'gs://ml-pipeline/data.csv',
    });
    expect(tree).toMatchSnapshot();
  });

  it('renders the selected visualization type correctly', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    tree.setState({
      selectedType: ApiVisualizationType.CURVE,
    });
    expect(tree).toMatchSnapshot();
  });

  it('disables all select and input fields when busy', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: true,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = shallow(<VisualizationCreator configs={[config]} />);
    // toMatchSnapshot is used rather than three individual checks for the
    // disabled prop due to an issue where the Input components are not
    // selectable by tree.find().
    expect(tree).toMatchSnapshot();
  });

  it('returns friendly display name', () => {
    expect(VisualizationCreator.prototype.getDisplayName()).toBe('Visualization Creator');
  });
});
