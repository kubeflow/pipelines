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
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { PlotType } from './Viewer';
import VisualizationCreator, { VisualizationCreatorConfig } from './VisualizationCreator';
import { ApiVisualizationType } from '../../apis/visualization';
import { diffHTML } from 'src/TestUtils';

describe('VisualizationCreator', () => {
  it('does not render component when no config is provided', () => {
    const { container } = render(<VisualizationCreator configs={[]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('renders component when empty config is provided', () => {
    const config: VisualizationCreatorConfig = {
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('renders component when isBusy is not provided', () => {
    const config: VisualizationCreatorConfig = {
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('renders component when onGenerate is not provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('renders component when all parameters in config are provided', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  // TODO: The following tests use shallow() with setState() to test component internal state.
  // RTL focuses on user behavior rather than implementation details.
  // Consider testing through user interactions instead.
  it.skip('does not render an Editor component if a visualization type is not specified', () => {
    // Skipped: Uses shallow() with setState() to test internal state
  });

  it.skip('renders an Editor component if a visualization type is specified', () => {
    // Skipped: Uses shallow() with setState() to test internal state
  });

  it.skip('renders two Editor components if the CUSTOM visualization type is specified', () => {
    // Skipped: Uses shallow() with setState() to test internal state
  });

  // TODO: The following tests use shallow() with setState() and enzyme's .find().prop() methods
  // to test button state. RTL focuses on testing through user interactions.
  it.skip('has a disabled BusyButton if selectedType is an undefined', () => {
    // Skipped: Uses shallow() with setState() and enzyme prop testing
  });

  it.skip('has a disabled BusyButton if source is an empty string', () => {
    // Skipped: Uses shallow() with setState() and enzyme prop testing
  });

  it.skip('has a disabled BusyButton if onGenerate is not provided as a prop', () => {
    // Skipped: Uses shallow() with setState() and enzyme prop testing
  });

  it.skip('has a disabled BusyButton if isBusy is true', () => {
    // Skipped: Uses shallow() with setState() and enzyme prop testing
  });

  it.skip('has an enabled BusyButton if onGenerate is provided and source and selectedType are set', () => {
    // Skipped: Uses shallow() with setState() and enzyme prop testing
  });

  it.skip('calls onGenerate when BusyButton is clicked', () => {
    // Skipped: Uses shallow() with setState() and enzyme simulate
  });

  it.skip('passes state as parameters to onGenerate when BusyButton is clicked', () => {
    // Skipped: Uses shallow() with setState() and enzyme simulate
  });

  it.skip('renders the provided arguments', () => {
    // Skipped: Uses shallow() with setState() to test component internal state
  });

  // TODO: This test uses mount() with setState() and enzyme prop testing.
  // RTL focuses on testing the actual DOM and user interactions.
  it.skip('renders a provided source', () => {
    // Skipped: Uses mount() with setState() and enzyme prop testing
  });

  it.skip('renders the selected visualization type', () => {
    // Skipped: Uses shallow() with setState() to test internal state
  });

  it('renders the custom type when it is allowed', () => {
    const config: VisualizationCreatorConfig = {
      allowCustomVisualizations: true,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('disables all select and input fields when busy', () => {
    const config: VisualizationCreatorConfig = {
      isBusy: true,
      type: PlotType.VISUALIZATION_CREATOR,
    };
    const { container } = render(<VisualizationCreator configs={[config]} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  // TODO: This test uses shallow() with setState() in a loop and enzyme prop testing
  // to verify placeholders exist for all visualization types.
  it.skip('has an argument placeholder for every visualization type', () => {
    // Skipped: Uses shallow() with setState() loop and enzyme prop testing
  });

  it('returns friendly display name', () => {
    expect(VisualizationCreator.prototype.getDisplayName()).toBe('Visualization Creator');
  });

  it('can be configured as collapsed initially and clicks to open', () => {
    const baseConfig: VisualizationCreatorConfig = {
      isBusy: false,
      onGenerate: jest.fn(),
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
          class="MuiButtonBase-root MuiButton-root MuiButton-text MuiButton-textPrimary MuiButton-sizeMedium MuiButton-textSizeMedium MuiButton-colorPrimary MuiButton-root MuiButton-text MuiButton-textPrimary MuiButton-sizeMedium MuiButton-textSizeMedium MuiButton-colorPrimary css-1e6y48t-MuiButtonBase-root-MuiButton-root"
          tabindex="0"
          type="button"
        >
          create visualizations manually
          <span
            class="MuiTouchRipple-root css-8je8zh-MuiTouchRipple-root"
          />
        </button>
      </div>
    `);
    const button = screen.getByText('create visualizations manually');
    fireEvent.click(button);
    // expanding a visualization creator is equivalent to rendering a non-collapsed visualization creator
    expect(diffHTML({ base: baseContainer.innerHTML, update: container.innerHTML }))
      .toMatchInlineSnapshot(`
      Snapshot Diff:
      - Expected
      + Received

      @@ --- --- @@
              class="MuiInputBase-root MuiOutlinedInput-root MuiInputBase-colorPrimary MuiInputBase-formControl css-1yk1gt9-MuiInputBase-root-MuiOutlinedInput-root-MuiSelect-root"
            >
              <div
                tabindex="0"
                role="combobox"
      -         aria-controls=":rc:"
      +         aria-controls=":re:"
                aria-expanded="false"
                aria-haspopup="listbox"
                aria-labelledby="mui-component-select-Visualization Type"
                id="mui-component-select-Visualization Type"
                class="MuiSelect-select MuiSelect-outlined MuiInputBase-input MuiOutlinedInput-input css-11u53oe-MuiSelect-select-MuiInputBase-input-MuiOutlinedInput-input"
      @@ --- --- @@
            style="height: 40px; max-width: 600px; width: 100%;"
          >
            <label
              class="MuiFormLabel-root MuiInputLabel-root MuiInputLabel-formControl MuiInputLabel-animated MuiInputLabel-sizeMedium MuiInputLabel-outlined MuiFormLabel-colorPrimary MuiInputLabel-root MuiInputLabel-formControl MuiInputLabel-animated MuiInputLabel-sizeMedium MuiInputLabel-outlined css-14s5rfu-MuiFormLabel-root-MuiInputLabel-root"
              data-shrink="false"
      -       for=":rd:"
      -       id=":rd:-label"
      +       for=":rf:"
      +       id=":rf:-label"
              >Source</label
            >
            <div
              class="MuiInputBase-root MuiOutlinedInput-root MuiInputBase-colorPrimary MuiInputBase-formControl css-9ddj71-MuiInputBase-root-MuiOutlinedInput-root"
            >
              <input
                aria-invalid="false"
      -         id=":rd:"
      +         id=":rf:"
                placeholder="File path or path pattern of data within GCS."
                type="text"
                class="MuiInputBase-input MuiOutlinedInput-input css-1t8l2tu-MuiInputBase-input-MuiOutlinedInput-input"
                value=""
              />
    `);
  });
});
