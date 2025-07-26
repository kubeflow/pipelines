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

import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import * as React from 'react';
import NewRunParameters, { NewRunParametersProps } from './NewRunParameters';

describe('NewRunParameters', () => {
  // TODO: Failing, snapshot is not matching
  it.skip('shows parameters', () => {
    const props = {
      handleParamChange: jest.fn(),
      initialParams: [{ name: 'testParam', value: 'testVal' }],
      titleMessage: 'Specify parameters required by the pipeline',
    } as NewRunParametersProps;
    const { asFragment } = render(<NewRunParameters {...props} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('does not display any text fields if there are no parameters', () => {
    const props = {
      handleParamChange: jest.fn(),
      initialParams: [],
      titleMessage: 'This pipeline has no parameters',
    } as NewRunParametersProps;
    const { asFragment } = render(<NewRunParameters {...props} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('clicking the open editor button for json parameters displays an editor', () => {
    const handleParamChange = jest.fn();
    const props = {
      handleParamChange,
      initialParams: [{ name: 'testParam', value: '{"test":"value"}' }],
      titleMessage: 'Specify json parameters required by the pipeline',
    } as NewRunParametersProps;
    render(<NewRunParameters {...props} />);

    // Initially the button should show "Open Json Editor"
    const openEditorButton = screen.getByRole('button', { name: /Open Json Editor/i });
    fireEvent.click(openEditorButton);

    expect(handleParamChange).toHaveBeenCalledTimes(1);
    expect(handleParamChange).toHaveBeenLastCalledWith(0, '{\n  "test": "value"\n}');

    // After clicking, the button should change to "Close Json Editor"
    expect(screen.getByRole('button', { name: /Close Json Editor/i })).toBeInTheDocument();
  });

  it('fires handleParamChange callback on change', () => {
    const handleParamChange = jest.fn();
    const props = {
      handleParamChange,
      initialParams: [
        { name: 'testParam1', value: 'testVal1' },
        { name: 'testParam2', value: 'testVal2' },
      ],
      titleMessage: 'Specify parameters required by the pipeline',
    } as NewRunParametersProps;

    render(<NewRunParameters {...props} />);

    const input = screen.getByDisplayValue('testVal2');
    fireEvent.change(input, { target: { value: 'test param value' } });

    expect(handleParamChange).toHaveBeenCalledTimes(1);
    expect(handleParamChange).toHaveBeenLastCalledWith(1, 'test param value');
  });
});
