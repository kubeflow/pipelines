/*
 * Copyright 2023 The Kubeflow Authors
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
import { BuildInfoContext } from '../lib/BuildInfo';
import PrivateSharedSelector, { PrivateSharedSelectorProps } from './PrivateSharedSelector';
import { PipelineTabsHeaders } from '../pages/PrivateAndSharedPipelines';

function generateProps(): PrivateSharedSelectorProps {
  return {
    onChange: jest.fn(),
  };
}

describe('PrivateSharedSelector', () => {
  it('it renders correctly', async () => {
    const tree = render(<PrivateSharedSelector {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('it changes checked input on click', async () => {
    render(
      <BuildInfoContext.Provider value={{ apiServerMultiUser: true }}>
        <PrivateSharedSelector {...generateProps()} />
      </BuildInfoContext.Provider>,
    );

    const privateInput = screen.getByLabelText(PipelineTabsHeaders.PRIVATE) as HTMLInputElement;
    const sharedInput = screen.getByLabelText(PipelineTabsHeaders.SHARED) as HTMLInputElement;
    expect(privateInput.checked).toBe(true);
    expect(sharedInput.checked).toBe(false);

    fireEvent.click(sharedInput);

    expect(privateInput.checked).toBe(false);
    expect(sharedInput.checked).toBe(true);
  });
});
