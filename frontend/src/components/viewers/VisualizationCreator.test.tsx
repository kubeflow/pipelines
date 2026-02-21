/*
 * Copyright 2019 The Kubeflow Authors
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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { PlotType } from './Viewer';
import VisualizationCreator, { VisualizationCreatorConfig } from './VisualizationCreator';
import { ApiVisualizationType } from '../../apis/visualization';
import Select from '@material-ui/core/Select';
import renderer from 'react-test-renderer';

vi.mock('../Editor', () => ({
  default: ({ placeholder }: { placeholder?: string }) => (
    <div data-testid='editor' data-placeholder={placeholder || ''} />
  ),
}));

type VisualizationCreatorState = VisualizationCreator['state'];

type VisualizationCreatorRender = ReturnType<typeof render>;

class VisualizationCreatorWrapper {
  private _instance: VisualizationCreator;
  private _renderResult: VisualizationCreatorRender;

  public constructor(instance: VisualizationCreator, renderResult: VisualizationCreatorRender) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): VisualizationCreator {
    return this._instance;
  }

  public setState(nextState: Partial<VisualizationCreatorState>): void {
    act(() => {
      this._instance.setState(nextState);
    });
  }

  public renderResult(): VisualizationCreatorRender {
    return this._renderResult;
  }
}

function renderVisualizationCreator(
  configs: VisualizationCreatorConfig[],
): VisualizationCreatorWrapper {
  const ref = React.createRef<VisualizationCreator>();
  const renderResult = render(<VisualizationCreator ref={ref} configs={configs} />);
  if (!ref.current) {
    throw new Error('VisualizationCreator instance not available');
  }
  return new VisualizationCreatorWrapper(ref.current, renderResult);
}

describe('VisualizationCreator', () => {
  it('does not render component when no config is provided', () => {
    const { container } = render(<VisualizationCreator configs={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders component when empty config is provided', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { asFragment } = render(<VisualizationCreator configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders component when isBusy is not provided', () => {
    const config: VisualizationCreatorConfig = {
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { asFragment } = render(<VisualizationCreator configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders component when onGenerate is not provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { asFragment } = render(<VisualizationCreator configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders component when all parameters in config are provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { asFragment } = render(<VisualizationCreator configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not render an Editor component if a visualization type is not specified', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    render(<VisualizationCreator configs={[config]} />);
    expect(screen.queryByTestId('editor')).toBeNull();
  });

  it('renders an Editor component if a visualization type is specified', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ selectedType: ApiVisualizationType.ROCCURVE });
    expect(screen.getAllByTestId('editor').length).toBe(1);
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
  });

  it('renders two Editor components if the CUSTOM visualization type is specified', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ selectedType: ApiVisualizationType.CUSTOM });
    expect(screen.getAllByTestId('editor').length).toBe(2);
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
  });

  it('has a disabled BusyButton if selectedType is an undefined', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ source: 'gs://ml-pipeline/data.csv' });
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).toBeDisabled();
  });

  it('has a disabled BusyButton if source is an empty string', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ selectedType: ApiVisualizationType.ROCCURVE });
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).toBeDisabled();
  });

  it('has a disabled BusyButton if onGenerate is not provided as a prop', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      selectedType: ApiVisualizationType.ROCCURVE,
      source: 'gs://ml-pipeline/data.csv',
    });
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).toBeDisabled();
  });

  it('has a disabled BusyButton if isBusy is true', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: true,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      selectedType: ApiVisualizationType.ROCCURVE,
      source: 'gs://ml-pipeline/data.csv',
    });
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).toBeDisabled();
  });

  it('has an enabled BusyButton if onGenerate is provided and source and selectedType are set', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      selectedType: ApiVisualizationType.ROCCURVE,
      source: 'gs://ml-pipeline/data.csv',
    });
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).not.toBeDisabled();
  });

  it('calls onGenerate when BusyButton is clicked', () => {
    const onGenerate = vi.fn();
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      arguments: '{}',
      selectedType: ApiVisualizationType.ROCCURVE,
      source: 'gs://ml-pipeline/data.csv',
    });
    fireEvent.click(screen.getByRole('button', { name: 'Generate Visualization' }));
    expect(onGenerate).toHaveBeenCalled();
  });

  it('passes state as parameters to onGenerate when BusyButton is clicked', () => {
    const onGenerate = vi.fn();
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      arguments: '{}',
      selectedType: ApiVisualizationType.ROCCURVE,
      source: 'gs://ml-pipeline/data.csv',
    });
    fireEvent.click(screen.getByRole('button', { name: 'Generate Visualization' }));
    expect(onGenerate).toHaveBeenCalledWith(
      '{}',
      'gs://ml-pipeline/data.csv',
      ApiVisualizationType.ROCCURVE,
    );
  });

  it('renders the provided arguments', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({
      arguments: JSON.stringify({ is_generated: 'True' }),
      selectedType: ApiVisualizationType.ROCCURVE,
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
  });

  it('renders a provided source', () => {
    const source = 'gs://ml-pipeline/data.csv';
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ source });
    expect(
      screen.getByPlaceholderText('File path or path pattern of data within GCS.'),
    ).toHaveValue(source);
  });

  it('renders the selected visualization type', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    wrapper.setState({ selectedType: ApiVisualizationType.ROCCURVE });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
  });

  it('renders the custom type when it is allowed', () => {
    const config: VisualizationCreatorConfig = {
      allowCustomVisualizations: true,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { asFragment } = render(<VisualizationCreator configs={[config]} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('disables all select and input fields when busy', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: true,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const tree = renderer.create(<VisualizationCreator configs={[config]} />);
    const select = tree.root.findByType(Select);
    expect(select.props.disabled).toBe(true);
    render(<VisualizationCreator configs={[config]} />);
    expect(
      screen.getByPlaceholderText('File path or path pattern of data within GCS.'),
    ).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Generate Visualization' })).toBeDisabled();
  });

  it('has an argument placeholder for every visualization type', () => {
    const types = Object.keys(ApiVisualizationType)
      .map((key: string) => key.replace('_', ''))
      .filter((key: string, i: number, arr: string[]) => arr.indexOf(key) === i);
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const wrapper = renderVisualizationCreator([config]);
    for (const type of types) {
      const placeholder = (wrapper.instance() as any).getArgumentPlaceholderForType(type);
      expect(placeholder).not.toBeNull();
    }
  });

  it('returns friendly display name', () => {
    expect(VisualizationCreator.prototype.getDisplayName()).toBe('Visualization Creator');
  });

  it('can be configured as collapsed initially and clicks to open', () => {
    const baseConfig: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: vi.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
      collapsedInitially: false,
    };
    const { container: baseContainer } = render(<VisualizationCreator configs={[baseConfig]} />);
    const { container } = render(
      <VisualizationCreator
        configs={[
          {
            ...baseConfig,
            collapsedInitially: true,
          },
        ]}
      />,
    );
    expect(container).toMatchInlineSnapshot(`
      <div>
        <button
          class="MuiButtonBase-root MuiButton-root MuiButton-text"
          tabindex="0"
          type="button"
        >
          <span
            class="MuiButton-label"
          >
            create visualizations manually
          </span>
          <span
            class="MuiTouchRipple-root"
          />
        </button>
      </div>
    `);
    const button = screen.getByText('create visualizations manually');
    fireEvent.click(button);
    expect(container.innerHTML).toEqual(baseContainer.innerHTML);
  });
});
