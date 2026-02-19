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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { LineageActionBar, LineageActionBarProps, LineageActionBarState } from './LineageActionBar';
import { buildTestModel, testModel } from './TestUtils';
import { Artifact } from 'src/third_party/mlmd';

type LineageActionBarInstance = LineageActionBar;

type LineageActionBarRender = ReturnType<typeof render>;

class LineageActionBarWrapper {
  private _instance: LineageActionBarInstance;
  private _renderResult: LineageActionBarRender;

  public constructor(instance: LineageActionBarInstance, renderResult: LineageActionBarRender) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): LineageActionBarInstance {
    return this._instance;
  }

  public state<K extends keyof LineageActionBarState>(
    key?: K,
  ): LineageActionBarState | LineageActionBarState[K] {
    const state = this._instance.state;
    return key ? state[key] : state;
  }

  public unmount(): void {
    this._renderResult.unmount();
  }

  public renderResult(): LineageActionBarRender {
    return this._renderResult;
  }
}

function renderActionBar(props: LineageActionBarProps): LineageActionBarWrapper {
  const ref = React.createRef<LineageActionBarInstance>();
  const renderResult = render(<LineageActionBar ref={ref} {...props} />);
  if (!ref.current) {
    throw new Error('LineageActionBar instance not available');
  }
  return new LineageActionBarWrapper(ref.current, renderResult);
}

describe('LineageActionBar', () => {
  const setLineageViewTarget = vi.fn();

  afterEach(() => {
    setLineageViewTarget.mockReset();
  });

  it('renders correctly for a given initial target', () => {
    const { asFragment, unmount } = render(
      <LineageActionBar initialTarget={testModel} setLineageViewTarget={vi.fn()} />,
    );
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('does not update the LineageView target when the current breadcrumb is clicked', () => {
    const wrapper = renderActionBar({
      initialTarget: testModel,
      setLineageViewTarget,
    });

    expect(wrapper.state('history').length).toBe(1);
    expect(setLineageViewTarget).not.toHaveBeenCalled();

    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[0]);

    expect(setLineageViewTarget).not.toHaveBeenCalled();
    expect(wrapper.state('history').length).toBe(1);
    wrapper.unmount();
  });

  it('updates the LineageView target model when an inactive breadcrumb is clicked', () => {
    const wrapper = renderActionBar({
      initialTarget: testModel,
      setLineageViewTarget,
    });

    act(() => {
      wrapper.instance().setState({
        history: [testModel, buildTestModel()],
      });
    });

    expect(wrapper.state('history').length).toBe(2);
    expect(setLineageViewTarget).not.toHaveBeenCalled();

    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[0]);

    expect(setLineageViewTarget).toHaveBeenCalledTimes(1);
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('adds the artifact to the history state and DOM when pushHistory() is called', () => {
    const wrapper = renderActionBar({
      initialTarget: testModel,
      setLineageViewTarget,
    });

    expect(wrapper.state('history').length).toBe(1);
    act(() => {
      wrapper.instance().pushHistory(new Artifact());
    });
    expect(wrapper.state('history').length).toBe(2);

    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('sets history to the initial prop when the reset button is clicked', () => {
    const wrapper = renderActionBar({
      initialTarget: testModel,
      setLineageViewTarget,
    });

    expect(wrapper.state('history').length).toBe(1);
    act(() => {
      wrapper.instance().pushHistory(buildTestModel());
    });
    act(() => {
      wrapper.instance().pushHistory(buildTestModel());
    });
    expect(wrapper.state('history').length).toBe(3);

    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[buttons.length - 1]);

    expect(wrapper.state('history').length).toBe(1);
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });
});
